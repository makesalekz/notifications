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

	users_v1 "gitlab.calendaria.team/services/iam/api/iam/v1"
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

	qm.AddConsumer(QueueFCM, uc.handlePushNotifications)
	qm.AddConsumer(QueueFCMSilent, uc.handleSilentPushNotifications)

	return uc, nil
}

func (uc *FcmUsecase) handlePushNotifications(ctx context.Context, m jetstream.Msg) bool {
	notification := u_struc.FirebaseNotification{}
	err := json.Unmarshal(m.Data(), &notification)
	if err != nil {
		uc.log.Errorf("handlePushNotifications: json.Unmarshal: %s", err.Error())
		return true
	}

	uc.log.WithContext(ctx).Debugf("handlePushNotifications: %v", notification)

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
				uc.log.Errorf("handlePushNotifications: notificationsRepo.CreateNotifications: %s", err2.Error())
			}
		}
	}

	return true
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

	go uc.deleteInactiveTokens(ctx, result.InactiveTokens)

	if isFirst && len(result.NeedsReFetch) > 0 {
		uc.fetchBadges(ctx, result.NeedsReFetch)
		newMsg := msg
		newMsg.UsersIds = result.NeedsReFetch
		uc.sendMessage(ctx, newMsg, false)
	}

	return true
}

type PushDispatchResult struct {
	InactiveTokens []string
	NeedsReFetch   []int64
	msg            PushDispatchContext
}

func (uc *FcmUsecase) SendUserNotifications(
	ctx context.Context, msg u_struc.FirebaseNotification, isFirst bool,
) (*PushDispatchResult, error) {
	userDevicesMap, err := uc.GetUserDevices(ctx, msg.UsersIds)
	if err != nil {
		return nil, err
	}

	if len(userDevicesMap) == 0 {
		return &PushDispatchResult{}, nil
	}

	userSettings, err := uc.iam.GetUsersSettings(ctx, msg.UsersIds)
	if err != nil {
		uc.log.Errorf("SendUserNotifications: iam.GetUsersSettings: %s", err.Error())
	}

	result := &PushDispatchResult{
		InactiveTokens: make([]string, 0),
		NeedsReFetch:   make([]int64, 0),
	}
	processedMsg := msg

	for userID, devices := range userDevicesMap {
		contactName, contactAvatar := uc.ExtractContactNameAndAvatar(ctx, &msg, userID)

		withSound, withVibration := uc.GetUserNotificationSettings(ctx, userID, userSettings)

		badgeCount, badgeErr := uc.GetAndIncrementBadge(ctx, userID, msg.Type, msg.Data["type"], isFirst)

		if badgeErr != nil && isFirst {
			result.NeedsReFetch = append(result.NeedsReFetch, userID)
			uc.log.Errorf("SendUserNotifications: failed to get badges for user %d: %v", userID, badgeErr)
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

			inactiveTokens := uc.DispatchPushNotifications(ctx, dispatchCtx)

			result.InactiveTokens = append(result.InactiveTokens, inactiveTokens...)
			result.msg = dispatchCtx
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
	// Log incoming parameters
	if msgDataJSON, err := json.Marshal(msg.Data); err == nil {
		uc.log.Debugf(
			"LocalizeNotification: Input params - lang=%s, contactName=%s, contactAvatar=%s, title=%s, body=%s, msgData=%s",
			lang, contactName, contactAvatar, msg.Title, msg.Body, string(msgDataJSON),
		)
	}

	if lang == "null" {
		uc.log.Debugf("LocalizeNotification: Lang is null, returning original values")
		return msg.Title, msg.Body, msg.Image
	}

	localizedTitle := msg.Title
	localizedBody := msg.Body
	imageURL := msg.Image

	chatType := ""

	dto := &data.NotificationDto{}

	if err := dto.ParseAndSetNotificationData(msg.Data); err == nil && dto.Type != nil {
		// Log parsed DTO
		if dtoJSON, err := json.Marshal(dto); err == nil {
			uc.log.Debugf("LocalizeNotification: Parsed DTO - %s", string(dtoJSON))
		}

		// Log converted map
		if convertedMapJSON, err := json.Marshal(dto.GetConvertedMap()); err == nil {
			uc.log.Debugf("LocalizeNotification: ConvertedMap - %s", string(convertedMapJSON))
		}

		if contactName != "" && len(dto.GetConvertedMap()) > 0 {
			if userData, ok := dto.GetConvertedMap()["user"]; ok {
				if userMap, ok := userData.(map[string]interface{}); ok {
					userMap["name"] = contactName
					uc.log.Debugf("LocalizeNotification: Updated user name in convertedMap to: %s", contactName)
				}
			}
		}

		// Determine chat type with fallback logic
		if chat := dto.GetChat(); chat != nil && chat.GetType() != "" {
			chatType = chat.GetType()
		} else {
			// Fallback: try to determine chat type from context
			if dto.GetChat() != nil {
				// If chat has eventId, it's likely an EVENT chat
				if dto.GetChat().EventId != nil && dto.GetChat().GetEventId() != 0 {
					chatType = "EVENT"
				} else if dto.GetChat().Title != nil && dto.GetChat().GetTitle() != "" {
					// If chat has a title, it's likely a GROUP chat
					chatType = "GROUP"
				} else {
					// Default to DIRECT chat
					chatType = "DIRECT"
				}
			} else {
				// Ultimate fallback: assume DIRECT chat
				chatType = "DIRECT"
			}
		}

		// Обрабатываем заголовок и изображение в зависимости от типа чата
		if chat := dto.GetChat(); chat != nil {
			if chatType == "GROUP" || chatType == "EVENT" {
				if chat.Title != nil && chat.GetTitle() != "" {
					localizedTitle = chat.GetTitle()
					uc.log.Debugf("LocalizeNotification: Using chat title: %s", localizedTitle)
				}

				if chat.Cover != nil && chat.GetCover() != "" {
					imageURL = chat.GetCover()
					uc.log.Debugf("LocalizeNotification: Using chat cover: %s", imageURL)
				}
			} else if chatType == "DIRECT" {
				if contactName != "" {
					localizedTitle = contactName
					uc.log.Debugf("LocalizeNotification: Using contact name as title: %s", localizedTitle)
				}

				if contactAvatar != "" {
					imageURL = contactAvatar
					uc.log.Debugf("LocalizeNotification: Using contact avatar: %s", imageURL)
				}
			}
		}

		if *dto.Type == "chat.update" || *dto.Type == "EVENT_UPDATED" {
			uc.log.Debugf("LocalizeNotification: Processing metadata for type: %s", *dto.Type)
			if metadataRaw, ok := msg.Data["metadata"]; ok {
				uc.log.Debugf("LocalizeNotification: Raw metadata: %s", metadataRaw)
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
						uc.log.Debugf(
							"LocalizeNotification: Failed to translate metadata part '%s', using original", part,
						)
					} else {
						uc.log.Debugf("LocalizeNotification: Translated metadata part '%s' -> '%s'", part, translated)
					}
					translatedParts = append(translatedParts, translated)
				}

				metadataString := strings.Join(translatedParts, ", ")
				dto.GetConvertedMap()["metadata"] = metadataString
				uc.log.Debugf("LocalizeNotification: Final metadata string: %s", metadataString)
			}
		}

		body, err := uc.localizer.GetLocalizedMessage(
			lang, *dto.Type, dto.GetConvertedMap(), dto.PluralCount,
		)
		if err == nil {
			localizedBody = body
			uc.log.Debugf("LocalizeNotification: Localized body: %s", localizedBody)
		}
		if err != nil {
			uc.log.Debugf("LocalizeNotification: Failed to localize message: %v", err)
		}

		if chatType == "GROUP" || chatType == "EVENT" {
			subType := msg.Data["type"]
			uc.log.Debugf(
				"LocalizeNotification: Processing group/event message with chatType='%s', subType='%s'", chatType,
				subType,
			)

			if subType == "message.new" || subType == "message.photo" {
				if contactName != "" {
					originalBody := localizedBody
					localizedBody = contactName + ": " + localizedBody
					uc.log.Debugf(
						"LocalizeNotification: Added contact name prefix: '%s' -> '%s'", originalBody, localizedBody,
					)
				} else {
					uc.log.Debugf("LocalizeNotification: Contact name is empty, cannot add prefix")
				}
			}
		} else {
			uc.log.Debugf("LocalizeNotification: Not a group/event message, chatType='%s', no prefix added", chatType)
		}
	} else {
		uc.log.Debugf("LocalizeNotification: Failed to parse notification data or type is nil. Error: %v", err)
	}

	// Log final result
	uc.log.Debugf(
		"LocalizeNotification: Final result - title='%s', body='%s', imageURL='%s'",
		localizedTitle, localizedBody, imageURL,
	)

	return localizedTitle, localizedBody, imageURL
}

func (dispatchCtx *PushDispatchContext) BuildPushMessage() *messaging.Message {
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
	ctx context.Context, dispatchCtx PushDispatchContext,
) []string {
	inactiveTokens := make([]string, 0)

	message := dispatchCtx.BuildPushMessage()

	for _, device := range dispatchCtx.Devices {
		deviceMessage := *message
		deviceMessage.Token = device.Token

		err := uc.fcmClient.Send(ctx, device.Token, &deviceMessage)
		if err != nil {
			uc.log.Errorf("DispatchPushNotifications: invalid token %s: %v", device.Token, err)
			inactiveTokens = append(inactiveTokens, device.Token)
		} else {
			if deviceMessage.Notification != nil {
				uc.log.Debugf(
					"DispatchPushNotifications: sent successfully userID=[%d] (%v)", dispatchCtx.UserID, deviceMessage,
				)
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
	ctx context.Context, user *users_v1.User, receiverID int64,
) string {
	if user == nil {
		return ""
	}

	authorName := ""
	if user.GetName() != "" {
		authorName = user.GetName()
	} else if user.GetUsername() != "" {
		authorName = user.GetUsername()
	}

	ctxWithUserID := auth.AppendAuthIds(ctx, receiverID, 0)
	contacts, err := uc.contactsRemote.GetContactsByUserId(ctxWithUserID, user.GetId())
	if err != nil {
		uc.log.Debugf("failed to get contacts for user %d: %v", receiverID, err)
		return authorName
	} else {
		uc.log.Debugf("ReceverContacts: [%d]: %v", receiverID, contacts)
	}

	for _, contact := range contacts {
		if contact.UserId != nil && contact.GetUserId() == user.GetId() && contact.GetLabel() != "" {
			return contact.GetLabel()
		}
	}

	return authorName
}

func (uc *FcmUsecase) ExtractContactNameAndAvatar(
	ctx context.Context, msg *u_struc.FirebaseNotification, receiverID int64,
) (string, string) {
	var user users_v1.User

	if userJSON, ok := msg.Data["user"]; ok {
		err := json.Unmarshal([]byte(userJSON), &user)
		if err != nil {
			uc.log.Debugf("failed to parse user json: %v", err)
			return "", ""
		}
	}

	authorAvatar := user.GetAvatar()
	contactName := uc.getAuthorNameFromUser(ctx, &user, receiverID)

	return contactName, authorAvatar
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

func (uc *FcmUsecase) handleSilentPushNotifications(ctx context.Context, m jetstream.Msg) bool {
	notification := u_struc.FirebaseNotification{}
	err := json.Unmarshal(m.Data(), &notification)
	if err != nil {
		uc.log.Errorf("handlePushNotifications: json.Unmarshal: %s", err.Error())
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
