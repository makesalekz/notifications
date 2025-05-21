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
		uc.log.Warnf("sendMessage: SendUserNotifications: %s", err.Error())
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
		uc.log.Debugf("sendMessage: re-fetching badges for %v", candidatesToReFetch)
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
}

func (uc *FcmUsecase) SendUserNotifications(
	ctx context.Context, msg u_struc.FirebaseNotification, isFirst bool,
) (map[int64]PushDispatchResult, error) {
	userDevicesMap, err := uc.GetUserDevices(ctx, msg.UsersIds)
	if err != nil {
		return nil, err
	}

	if len(userDevicesMap) == 0 {
		uc.log.Debug("SendUserNotifications: No devices found")
		return map[int64]PushDispatchResult{}, nil
	}

	userSettings, err := uc.iam.GetUsersSettings(ctx, msg.UsersIds)
	if err != nil {
		uc.log.Warnf("SendUserNotifications: iam.GetUsersSettings: %s", err.Error())
	}

	result := make(map[int64]PushDispatchResult)

	processedMsg := msg
	if msg.Type == u_struc.Chat {
		for _, devices := range userDevicesMap {
			uc.ProcessChatNotification(ctx, &msg, devices)
			break
		}
	}

	for userID, devices := range userDevicesMap {
		withSound, withVibration := uc.GetUserNotificationSettings(ctx, userID, userSettings)

		badgeCount, badgeErr := uc.GetAndIncrementBadge(ctx, userID, msg.Type, isFirst)

		if badgeErr != nil && isFirst {
			result[userID] = PushDispatchResult{NeedsReFetch: true}
			uc.log.Warnf("SendUserNotifications: failed to get badges for user %d: %v", userID, badgeErr)
			continue
		}

		langDevicesMap := uc.GroupDevicesByLanguage(devices)
		for lang, langDevices := range langDevicesMap {
			localizedTitle, localizedBody := uc.LocalizeNotification(ctx, &processedMsg, lang)

			dispatchCtx := PushDispatchContext{
				UserID:         userID,
				Devices:        langDevices,
				BadgeCount:     badgeCount,
				WithSound:      withSound,
				WithVibration:  withVibration,
				LocalizedTitle: localizedTitle,
				LocalizedBody:  localizedBody,
				MessageData:    processedMsg.Data,
				ImageURL:       processedMsg.Image,
			}

			inactiveTokens := uc.DispatchPushNotifications(ctx, dispatchCtx, processedMsg)

			currentResult := result[userID]
			currentResult.InactiveTokens = append(currentResult.InactiveTokens, inactiveTokens...)
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
	ctx context.Context, userID int64, notifType u_struc.NotificationType, increment bool,
) (int, error) {
	badges, err := uc.badgeClient.GetBadges(ctx, userID)
	if err != nil {
		return 0, err
	}

	totalBadges := int64(1)
	for _, count := range badges {
		totalBadges += count
	}

	if increment {
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

func (uc *FcmUsecase) LocalizeNotification(ctx context.Context, msg *u_struc.FirebaseNotification, lang string) (
	string, string,
) {
	if lang == "null" {
		return msg.Title, msg.Body
	}

	localizedTitle := msg.Title
	localizedBody := msg.Body

	dto := &data.NotificationDto{}
	if err := dto.ParseAndSetNotificationData(msg.Data); err == nil && dto.Type != nil {
		body, err := uc.localizer.GetLocalizedMessage(
			lang, *dto.Type, dto.GetConvertedMap(), dto.PluralCount,
		)
		if err == nil {
			localizedBody = body
		}
	}

	return localizedTitle, localizedBody
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
				uc.log.Debugf("DispatchPushNotifications: sent successfully (%s)", deviceMessage.Notification.Body)
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
	ctx context.Context, user map[string]interface{}, userDevices []*ent.Device,
) string {
	if user == nil || len(userDevices) == 0 {
		return ""
	}

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

	receiverID := userDevices[0].UserID

	ctxWithUserID := auth.AppendAuthIds(ctx, receiverID, 0)

	contacts, err := uc.contactsRemote.GetContactsByUserId(ctxWithUserID, receiverID)
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

func (uc *FcmUsecase) ProcessChatNotification(
	ctx context.Context, msg *u_struc.FirebaseNotification, userDevices []*ent.Device,
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
	imageURL := msg.Image
	title := msg.Title
	body := msg.Body

	contactName := ""
	if len(user) > 0 && len(userDevices) > 0 {
		contactName = uc.getAuthorNameFromUser(ctx, user, userDevices)
	}

	displayAuthorName := authorName
	if contactName != "" {
		displayAuthorName = contactName
	}

	if chatJSON, ok := msg.Data["chat"]; ok {
		var chat map[string]interface{}
		err := json.Unmarshal([]byte(chatJSON), &chat)
		if err != nil {
			uc.log.Debugf("failed to parse chat json: %v", err)
			return title, imageURL
		}

		chatType, _ := chat["type"].(string)
		groupName, hasGroupName := chat["title"].(string)

		if chatType == "GROUP" || chatType == "EVENT" {
			// Для групповых чатов и чатов событий
			if hasGroupName && groupName != "" {
				title = groupName
			}

			// Установка обложки группы/события как изображения
			if cover, hasCover := chat["cover"].(string); hasCover && cover != "" {
				imageURL = cover
			}

			// Добавляем имя автора в начало текста сообщения
			if displayAuthorName != "" && !isSystemNotification(msg.Data) {
				body = displayAuthorName + ": " + body
			}
		} else if chatType == "DIRECT" {
			if displayAuthorName != "" {
				title = displayAuthorName
			}

			if authorAvatar != "" {
				imageURL = authorAvatar
			}
		}

		if notificationType, hasNotificationType := msg.Data["type"]; hasNotificationType {
			lang := "ru"
			if len(userDevices) > 0 && userDevices[0].Language != "" {
				lang = userDevices[0].Language
			}

			dto := &data.NotificationDto{}
			if err := dto.ParseAndSetNotificationData(msg.Data); err == nil {
				dto.Type = &notificationType

				if contactName != "" && len(dto.GetConvertedMap()) > 0 {
					if userData, ok := dto.GetConvertedMap()["user"]; ok {
						if userMap, ok := userData.(map[string]interface{}); ok {
							userMap["name"] = contactName
						}
					}
				}

				if oldTitle, ok := msg.Data["old_title"]; ok {
					dto.GetConvertedMap()["oldTitle"] = oldTitle
					dto.GetConvertedMap()["newTitle"] = chat["title"]
				}

				if notificationType == "chat.update" {
					if metadataRaw, ok := msg.Data["metadata"]; ok {
						var translatedParts []string
						parts := strings.Split(metadataRaw, ",")

						for _, part := range parts {
							part = normalizeKey(part)
							translated, err := uc.localizer.GetLocalizedMessage(
								lang, "chat.update_metadata."+part, nil, nil,
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

				localizedBody, err := uc.localizer.GetLocalizedMessage(
					lang, notificationType, dto.GetConvertedMap(), dto.PluralCount,
				)
				if err == nil {
					body = localizedBody
				}

				if (chatType == "GROUP" || chatType == "EVENT") &&
					(notificationType == "message.new" || notificationType == "message.photo") {
					body = displayAuthorName + ": " + localizedBody
				}
			}
		}
	}

	msg.Title = title
	msg.Body = body
	msg.Image = imageURL

	return title, imageURL
}

func normalizeKey(input string) string {
	return strings.ReplaceAll(
		strings.ToLower(strings.TrimSpace(input)),
		" ", "_",
	)
}

func isSystemNotification(data map[string]string) bool {
	if notifType, ok := data["type"]; ok {
		systemTypes := []string{
			"chat.update", "member.added", "GROUP_ADDED", "GROUP_TITLE_CHANGED",
			"GROUP_COVER_CHANGED", "MEMBER_ADDED", "TITLE_CHANGED", "COVER_CHANGED",
		}
		for _, t := range systemTypes {
			if notifType == t {
				return true
			}
		}
	}
	return false
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
	uc.log.Debugf("deleteInactiveTokens: deleted %d tokens", len(tokens))
}

func (uc *FcmUsecase) fetchBadges(ctx context.Context, userIDs []int64) {
	if len(userIDs) == 0 {
		return
	}

	var (
		eventsCountMap   = make(map[int64]int32)
		chatsCountMap    = make(map[int64]int32)
		contactsCountMap = make(map[int64]int32)
		wg               sync.WaitGroup
	)

	wg.Add(3)

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

	go func() {
		defer wg.Done()
		contactsCount, err := uc.notificationsRepo.CountUnreadNotificationsByType(
			ctx, userIDs, u_struc.Contact.Value(),
		)
		if err != nil {
			uc.log.Errorf("fetchBadges: failed to get contacts count: %v", err)
			return
		}

		for k, v := range contactsCount {
			contactsCountMap[k] = v
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

		if count, ok := contactsCountMap[userID]; ok {
			badges[u_struc.Contact] = int64(count)
		}

		err := uc.badgeClient.SetBadges(ctx, userID, badges)
		if err != nil {
			uc.log.Errorf("fetchBadges: failed to set badges for user %d: %v", userID, err)
		} else {
			uc.log.Infof("fetchBadges: updated badges for user %d: %v", userID, badges)
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

	uc.log.Debugf("sendNotifications: %v", notification)

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
			} else {
				uc.log.Debugf("sendMessage: sent successfully (%s)", message.Token)
			}
		}
	}

	return err == nil
}
