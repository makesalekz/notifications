package biz

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"firebase.google.com/go/v4/messaging"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/nats-io/nats.go/jetstream"

	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/internal/data"
	"gitlab.calendaria.team/services/notifications/internal/data/dialer"
	"gitlab.calendaria.team/services/utils/v2/auth"
	u_struc "gitlab.calendaria.team/services/utils/v2/struc"
	u_badge "gitlab.calendaria.team/services/utils/v4/badge"
	u_nats "gitlab.calendaria.team/services/utils/v4/nats"
)

type PushDispatchContext struct {
	UserID         int64
	Devices        []*ent.Device
	BadgeCount     int
	WithSound      bool
	WithVibration  bool
	LocalizedTitle string
	LocalizedBody  string
	MessageData    map[string]string
	ImageURL       string
}

type FcmUsecase struct {
	fcmClient         data.FcmClient
	log               *log.Helper
	devicesRepo       data.DevicesRepo
	localizer         *data.Localizer
	notificationsRepo data.NotificationsRepo
	qm                u_nats.IQueueManager
	badgeClient       u_badge.IBadgeClient
	chatsRemote       dialer.IChatsRemote
	eventsRemote      dialer.IEventsRemote
	iam               dialer.IIamRemote
	contactsRemote    dialer.IContactsRemote
}

func NewFcmUsecase(
	logger log.Logger,
	devicesRepo data.DevicesRepo,
	notificationsRepo data.NotificationsRepo,
	localizer *data.Localizer,
	qm u_nats.IQueueManager,
	badgeClient u_badge.IBadgeClient,
	fcmClient data.FcmClient,
	chatsRemote dialer.IChatsRemote,
	eventsRemote dialer.IEventsRemote,
	iam dialer.IIamRemote,
	contactsRemote dialer.IContactsRemote,
) (*FcmUsecase, error) {
	uc := &FcmUsecase{
		log:               log.NewHelper(logger),
		devicesRepo:       devicesRepo,
		notificationsRepo: notificationsRepo,
		qm:                qm,
		localizer:         localizer,
		badgeClient:       badgeClient,
		fcmClient:         fcmClient,
		chatsRemote:       chatsRemote,
		eventsRemote:      eventsRemote,
		iam:               iam,
		contactsRemote:    contactsRemote,
	}

	qm.AddConsumer(QueueFCM, uc.sendNotifications)
	qm.AddConsumer(QueueFCMSilent, uc.sendSilentPushes)

	return uc, nil
}

func (uc *FcmUsecase) sendNotifications(ctx context.Context, m jetstream.Msg) bool {
	notification := u_struc.FirebaseNotification{}
	err := json.Unmarshal(m.Data(), &notification)
	if err != nil {
		uc.log.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		return true
	}

	uc.log.WithContext(ctx).Debugf("sendNotifications: %v", notification)

	ok := uc.sendMessage(ctx, notification, true)
	if ok {
		listDto := make([]*data.NotificationDto, len(notification.UsersIds))

		for i, userID := range notification.UsersIds {
			dto := &data.NotificationDto{
				UserID: userID,
				Title:  notification.Title,
				Text:   notification.Body,
			}

			_ = dto.ParseAndSetNotificationData(notification.Data)

			listDto[i] = dto
		}

		if notification.Type.IsValid() && notification.Title != "" && notification.Type != u_struc.Chat {
			_, err2 := uc.notificationsRepo.CreateNotifications(ctx, listDto)
			if err2 != nil {
				uc.log.Errorf("sendNotifications: notificationsRepo.CreateNotifications: %s", err2.Error())
			}
		}
	}

	return ok
}

func (uc *FcmUsecase) sendMessage(ctx context.Context, msg u_struc.FirebaseNotification, isFirst bool) bool {
	if msg.Type.IsValid() {
		if msg.Title == "" && msg.Body == "" {
			uc.log.Debug("sendMessage: No title or body")
			return true
		}
	}

	result, err := uc.SendUserNotifications(ctx, msg, isFirst)
	if err != nil {
		uc.log.Errorf("sendMessage: SendUserNotifications: %s", err.Error())
		return false
	}

	var inactiveTokens []string
	for _, dispatchResult := range result {
		if len(dispatchResult.InactiveTokens) > 0 {
			inactiveTokens = append(inactiveTokens, dispatchResult.InactiveTokens...)
		}
	}
	go uc.deleteInactiveTokens(ctx, inactiveTokens)

	var candidatesToReFetch []int64
	for userID, dispatchResult := range result {
		if dispatchResult.NeedsReFetch {
			candidatesToReFetch = append(candidatesToReFetch, userID)
		}
	}

	if len(candidatesToReFetch) > 0 {
		uc.fetchBadges(ctx, candidatesToReFetch)
		newMsg := msg
		newMsg.UsersIds = candidatesToReFetch
		uc.sendMessage(ctx, newMsg, false)
	}

	return err == nil
}

type PushDispatchResult struct {
	InactiveTokens []string
	NeedsReFetch   bool
	msg            PushDispatchContext
}

func (uc *FcmUsecase) SendUserNotifications(
	ctx context.Context, msg u_struc.FirebaseNotification, isFirst bool,
) (map[int64]PushDispatchResult, error) {
	userDevicesMap, err := uc.GetUserDevices(ctx, msg.UsersIds)
	if err != nil {
		return nil, err
	}

	if len(userDevicesMap) == 0 {
		return map[int64]PushDispatchResult{}, nil
	}

	userSettings, err := uc.iam.GetUsersSettings(ctx, msg.UsersIds)
	if err != nil {
		uc.log.Warnf("SendUserNotifications: iam.GetUsersSettings: %s", err.Error())
	}

	result := make(map[int64]PushDispatchResult)

	processedMsg := msg

	contactName, contactAvatar := "", ""
	for userID, devices := range userDevicesMap {
		if contactName == "" {
			contactName, contactAvatar = uc.ExtractContactNameAndAvatar(ctx, &msg, userID)
		}

		withSound, withVibration := uc.GetUserNotificationSettings(ctx, userID, userSettings)

		badgeCount, badgeErr := uc.GetAndIncrementBadge(ctx, userID, msg.Type, msg.Data["type"], isFirst)

		if badgeErr != nil && isFirst {
			result[userID] = PushDispatchResult{NeedsReFetch: true}
			uc.log.Warnf("SendUserNotifications: failed to get badges for user %d: %v", userID, badgeErr)
			continue
		}

		langDevicesMap := uc.GroupDevicesByLanguage(devices)
		for lang, langDevices := range langDevicesMap {
			localizedTitle, localizedBody, coverImage := uc.LocalizeNotification(
				&processedMsg, contactName, contactAvatar, lang,
			)

			dispatchCtx := PushDispatchContext{
				UserID:         userID,
				Devices:        langDevices,
				BadgeCount:     badgeCount,
				WithSound:      withSound,
				WithVibration:  withVibration,
				LocalizedTitle: localizedTitle,
				LocalizedBody:  localizedBody,
				MessageData:    processedMsg.Data,
				ImageURL:       coverImage,
			}

			inactiveTokens := uc.DispatchPushNotifications(ctx, dispatchCtx, processedMsg)

			currentResult := result[userID]
			currentResult.InactiveTokens = append(currentResult.InactiveTokens, inactiveTokens...)
			currentResult.msg = dispatchCtx
			result[userID] = currentResult
		}
	}

	return result, nil
}

func (uc *FcmUsecase) GetUserDevices(ctx context.Context, userIds []int64) (map[int64][]*ent.Device, error) {
	devices, err := uc.devicesRepo.GetDevicesForUsers(ctx, userIds)
	if err != nil {
		uc.log.Warnf("GetUserDevices: devicesRepo.GetDevicesForUsers: %s", err.Error())
		return nil, err
	}

	userDevicesMap := make(map[int64][]*ent.Device)
	for _, device := range devices {
		userDevicesMap[device.UserID] = append(userDevicesMap[device.UserID], device)
	}

	return userDevicesMap, nil
}

func (uc *FcmUsecase) GetUserNotificationSettings(
	ctx context.Context, userID int64, userSettings map[int64]map[string]string,
) (bool, bool) {
	withSound := true
	withVibration := true

	userSettingsMap, ok := userSettings[userID]
	if ok {
		if soundSetting, hasSoundSetting := userSettingsMap["NOTIFICATION_SOUND_ENABLED"]; hasSoundSetting {
			if soundSetting == "false" {
				withSound = false
			}
		}

		if vibrationSetting, hasVibrationSetting := userSettingsMap["NOTIFICATION_VIBRATION_ENABLED"]; hasVibrationSetting {
			if vibrationSetting == "false" {
				withVibration = false
			}
		}
	}
	return withSound, withVibration
}

func (uc *FcmUsecase) GetAndIncrementBadge(
	ctx context.Context, userID int64, notifType u_struc.NotificationType, dataType string, increment bool,
) (int, error) {
	badges, err := uc.badgeClient.GetBadges(ctx, userID)
	if err != nil {
		return 0, err
	}

	totalBadges := int64(1)
	for _, count := range badges {
		totalBadges += count
	}

	if increment && (notifType == u_struc.Chat || (notifType == u_struc.Event && dataType == "NEW_INVITE")) {
		err = uc.badgeClient.IncrementBadge(ctx, userID, notifType)
		if err != nil {
			uc.log.Warnf("GetAndIncrementBadge: failed to increment badge for user %d: %v", userID, err)
		}
	}

	return int(totalBadges), nil
}

func (uc *FcmUsecase) GroupDevicesByLanguage(devices []*ent.Device) map[string][]*ent.Device {
	langDevicesMap := make(map[string][]*ent.Device)
	for _, device := range devices {
		lang := "null"
		if device.Language != "" {
			lang = device.Language
		}
		langDevicesMap[lang] = append(langDevicesMap[lang], device)
	}
	return langDevicesMap
}

func (uc *FcmUsecase) LocalizeNotification(
	msg *u_struc.FirebaseNotification, contactName, contactAvatar, lang string,
) (
	string, string, string,
) {
	if lang == "null" {
		return msg.Title, msg.Body, msg.Image
	}

	localizedTitle := msg.Title
	localizedBody := msg.Body
	imageURL := msg.Image

	chatType := ""

	dto := &data.NotificationDto{}

	if err := dto.ParseAndSetNotificationData(msg.Data); err == nil && dto.Type != nil {
		if contactName != "" && len(dto.GetConvertedMap()) > 0 {
			if userData, ok := dto.GetConvertedMap()["user"]; ok {
				if userMap, ok := userData.(map[string]interface{}); ok {
					userMap["name"] = contactName
				}
			}
		}

		if chatJSON, ok := msg.Data["chat"]; ok {
			var chat map[string]interface{}
			err = json.Unmarshal([]byte(chatJSON), &chat)
			if err != nil {
				uc.log.Debugf("failed to parse chat json: %v", err)
			}

			chatType, _ = chat["type"].(string)
			groupName, hasGroupName := chat["title"].(string)

			if chatType == "GROUP" || chatType == "EVENT" {
				if hasGroupName && groupName != "" {
					localizedTitle = groupName
				}

				if cover, hasCover := chat["cover"].(string); hasCover && cover != "" {
					imageURL = cover
				}
			} else if chatType == "DIRECT" {
				if contactName != "" {
					localizedTitle = contactName
				}

				if contactAvatar != "" {
					imageURL = contactAvatar
				}
			}
		}

		if *dto.Type == "chat.update" || *dto.Type == "EVENT_UPDATED" {
			if metadataRaw, ok := msg.Data["metadata"]; ok {
				var translatedParts []string
				parts := strings.Split(metadataRaw, ",")

				for _, part := range parts {
					part = normalizeKey(part)
					key := ""
					if *dto.Type == "chat.update" {
						key = "chat.update_metadata." + part
					} else if *dto.Type == "EVENT_UPDATED" {
						key = "event.update_metadata." + part
					}

					translated, err := uc.localizer.GetLocalizedMessage(
						lang, key, nil, nil,
					)
					if err != nil {
						translated = part
					}
					translatedParts = append(translatedParts, translated)
				}

				metadataString := strings.Join(translatedParts, ", ")
				dto.GetConvertedMap()["metadata"] = metadataString
			}
		}

		body, err := uc.localizer.GetLocalizedMessage(
			lang, *dto.Type, dto.GetConvertedMap(), dto.PluralCount,
		)
		if err == nil {
			localizedBody = body
		}
		if err != nil {
			uc.log.Debugf("failed to localize message: %v", err)
		}

		if (chatType == "GROUP" || chatType == "EVENT") &&
			(*dto.Type == "message.new" || *dto.Type == "message.photo") {
			localizedBody = contactName + ": " + localizedBody
		}
	}

	return localizedTitle, localizedBody, imageURL
}

func (uc *FcmUsecase) BuildPushMessage(
	ctx context.Context, dispatchCtx PushDispatchContext, notificationType u_struc.NotificationType,
) *messaging.Message {
	message := &messaging.Message{
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					MutableContent: true,
					Badge:          &dispatchCtx.BadgeCount,
				},
			},
		},
		Android: &messaging.AndroidConfig{
			Notification: &messaging.AndroidNotification{
				NotificationCount: &dispatchCtx.BadgeCount,
			},
		},
	}

	if dispatchCtx.WithSound {
		message.APNS.Payload.Aps.Sound = "default"
		message.Android.Notification.Sound = "default"
	} else {
		message.APNS.Payload.Aps.Sound = ""
		message.Android.Notification.Sound = ""
	}

	if dispatchCtx.WithVibration {
		if message.Data == nil {
			message.Data = make(map[string]string)
		}
		message.Data["vibrate"] = "true"
	}

	if len(dispatchCtx.MessageData) > 0 {
		message.Data = dispatchCtx.MessageData
	}

	if dispatchCtx.LocalizedTitle != "" || dispatchCtx.LocalizedBody != "" || dispatchCtx.ImageURL != "" {
		message.Notification = &messaging.Notification{
			Title:    dispatchCtx.LocalizedTitle,
			Body:     dispatchCtx.LocalizedBody,
			ImageURL: dispatchCtx.ImageURL,
		}
	}

	return message
}

func (uc *FcmUsecase) DispatchPushNotifications(
	ctx context.Context, dispatchCtx PushDispatchContext, originalMsg u_struc.FirebaseNotification,
) []string {
	inactiveTokens := make([]string, 0)

	message := uc.BuildPushMessage(ctx, dispatchCtx, originalMsg.Type)

	for _, device := range dispatchCtx.Devices {
		deviceMessage := *message
		deviceMessage.Token = device.Token

		err := uc.fcmClient.Send(ctx, device.Token, &deviceMessage)
		if err != nil {
			uc.log.Debugf("DispatchPushNotifications: invalid token %s: %v", device.Token, err)
			inactiveTokens = append(inactiveTokens, device.Token)
		} else {
			if deviceMessage.Notification != nil {
				uc.log.Debugf("DispatchPushNotifications: sent successfully (%v)", deviceMessage.Notification)
			} else {
				uc.log.Debugf("DispatchPushNotifications: sent successfully (silent push)")
			}
		}
	}

	return inactiveTokens
}

func (uc *FcmUsecase) RegisterDevice(ctx context.Context, device data.DeviceDto) error {
	err := uc.devicesRepo.CreateDevice(ctx, device)
	if err != nil {
		return err
	}

	if device.OldToken != "" {
		_, err = uc.devicesRepo.DeleteDevice(
			ctx, data.DeviceKey{
				UserID: device.UserID,
				Token:  device.OldToken,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (uc *FcmUsecase) UnregisterDevice(ctx context.Context, deviceKey data.DeviceKey) error {
	_, err := uc.devicesRepo.DeleteDevice(ctx, deviceKey)

	return err
}

func (uc *FcmUsecase) getAuthorNameFromUser(
	ctx context.Context, user map[string]interface{}, receiverID int64,
) string {
	authorID, ok := user["id"]
	if !ok {
		return ""
	}

	var authorIDInt int64
	switch id := authorID.(type) {
	case int64:
		authorIDInt = id
	case float64:
		authorIDInt = int64(id)
	case int:
		authorIDInt = int64(id)
	case string:
		parsed, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			uc.log.Debugf("failed to parse author id string: %v", err)
			return ""
		}
		authorIDInt = parsed
	default:
		uc.log.Debugf("unexpected author id type: %T", id)
		return ""
	}

	authorName := ""
	if name, ok := user["name"].(string); ok && name != "" {
		authorName = name
	} else if username, ok := user["username"].(string); ok && username != "" {
		authorName = username
	}

	ctxWithUserID := auth.AppendAuthIds(ctx, receiverID, 0)
	contacts, err := uc.contactsRemote.GetContactsByUserId(ctxWithUserID, authorIDInt)
	if err != nil {
		uc.log.Debugf("failed to get contacts for user %d: %v", receiverID, err)
		return authorName
	}

	for _, contact := range contacts {
		if contact.UserId != nil && contact.GetUserId() == authorIDInt && contact.GetLabel() != "" {
			return contact.GetLabel()
		}
	}

	return authorName
}

func (uc *FcmUsecase) getAuthorAvatar(user map[string]interface{}) string {
	if user == nil {
		return ""
	}

	if avatar, ok := user["avatar"].(string); ok && avatar != "" {
		return avatar
	}

	return ""
}

func (uc *FcmUsecase) ExtractContactNameAndAvatar(
	ctx context.Context, msg *u_struc.FirebaseNotification, receiverID int64,
) (string, string) {
	var user map[string]interface{}
	var authorName string

	if userJSON, ok := msg.Data["user"]; ok {
		err := json.Unmarshal([]byte(userJSON), &user)
		if err != nil {
			uc.log.Debugf("failed to parse user json: %v", err)
		} else {
			if name, ok := user["name"].(string); ok && name != "" {
				authorName = name
			} else if username, ok := user["username"].(string); ok && username != "" {
				authorName = username
			}
		}
	}

	authorAvatar := uc.getAuthorAvatar(user)

	contactName := uc.getAuthorNameFromUser(ctx, user, receiverID)

	displayAuthorName := authorName
	if contactName != "" {
		displayAuthorName = contactName
	}

	return displayAuthorName, authorAvatar
}

func normalizeKey(input string) string {
	return strings.ReplaceAll(
		strings.ToLower(strings.TrimSpace(input)),
		" ", "_",
	)
}

func (uc *FcmUsecase) deleteInactiveTokens(ctx context.Context, tokens []string) {
	if len(tokens) == 0 {
		return
	}

	_, err := uc.devicesRepo.DeleteDevicesByTokens(ctx, tokens)
	if err != nil {
		uc.log.Errorf("deleteInactiveTokens: devicesRepo.DeleteDevicesByTokens: %s", err.Error())
		return
	}
}

func (uc *FcmUsecase) fetchBadges(ctx context.Context, userIDs []int64) {
	if len(userIDs) == 0 {
		return
	}

	// todo: Add notifications count from db notificationsRepo.GetUnreadNotificationsCount after adding read status
	var (
		eventsCountMap = make(map[int64]int32)
		chatsCountMap  = make(map[int64]int32)
		wg             sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		events, err := uc.eventsRemote.GetEventsCount(ctx, userIDs)
		if err != nil {
			uc.log.Errorf("fetchBadges: failed to get events count: %v", err)
			return
		}

		for k, v := range events {
			eventsCountMap[k] = v
		}
	}()

	go func() {
		defer wg.Done()
		chats, err := uc.chatsRemote.CountUnreadMessages(ctx, userIDs)
		if err != nil {
			uc.log.Errorf("fetchBadges: failed to get chats count: %v", err)
			return
		}

		for k, v := range chats {
			chatsCountMap[k] = v
		}
	}()

	wg.Wait()

	for _, userID := range userIDs {
		badges := make(map[u_struc.NotificationType]int64)

		if count, ok := eventsCountMap[userID]; ok {
			badges[u_struc.Event] = int64(count)
		}

		if count, ok := chatsCountMap[userID]; ok {
			badges[u_struc.Chat] = int64(count)
		}

		err := uc.badgeClient.SetBadges(ctx, userID, badges)
		if err != nil {
			uc.log.Errorf("fetchBadges: failed to set badges for user %d: %v", userID, err)
		}
	}
}

func (uc *FcmUsecase) sendSilentPushes(ctx context.Context, m jetstream.Msg) bool {
	notification := u_struc.FirebaseNotification{}
	err := json.Unmarshal(m.Data(), &notification)
	if err != nil {
		uc.log.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		return true
	}

	ok := uc.sendSilentMessage(ctx, notification)

	return ok
}

func (uc *FcmUsecase) sendSilentMessage(ctx context.Context, notification u_struc.FirebaseNotification) bool {
	devices, err := uc.devicesRepo.GetDevicesForUsers(ctx, notification.UsersIds)
	if err != nil {
		uc.log.Warnf("sendMessage: devicesRepo.GetDevicesForUsers: %s", err.Error())
		return false
	}

	if len(devices) == 0 {
		uc.log.Debug("sendMessage: No devices found")
		return true
	}

	userDevicesMap := make(map[int64][]*ent.Device)
	for _, device := range devices {
		userDevicesMap[device.UserID] = append(userDevicesMap[device.UserID], device)
	}

	baseMessage := &messaging.Message{
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					ContentAvailable: true,
				},
			},
		},
		Android: &messaging.AndroidConfig{
			Notification: &messaging.AndroidNotification{},
		},
	}

	for userID, userDevices := range userDevicesMap {
		badges, badgeErr := uc.badgeClient.GetBadges(ctx, userID)
		if badgeErr != nil {
			uc.log.Warnf("sendMessage: failed to get badges for user %d: %v", userID, badgeErr)
			continue
		}

		totalBadges := int64(0)
		for _, count := range badges {
			totalBadges += count
		}

		badgeCount := int(totalBadges)
		baseMessage.Data = map[string]string{
			"badge": strconv.Itoa(badgeCount),
		}
		baseMessage.APNS.Payload.Aps.Badge = &badgeCount
		baseMessage.Android.Notification.NotificationCount = &badgeCount
		baseMessage.Android.Data = map[string]string{
			"badge": strconv.Itoa(badgeCount),
		}

		for _, device := range userDevices {
			message := *baseMessage

			message.Token = device.Token
			err = uc.fcmClient.Send(ctx, device.Token, &message)
			if err != nil {
				uc.log.Debugf("sendMessage: invalid token %s: %v", device.Token, err)
			}
		}
	}

	return err == nil
}
