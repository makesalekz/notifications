package biz

import (
	"context"
	"encoding/json"
	"strconv"
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

// FcmUsecase is a Greeter usecase.
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

	uc.log.Debugf("sendNotifications: %v", notification)

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

	devices, err := uc.devicesRepo.GetDevicesForUsers(ctx, msg.UsersIds)
	if err != nil {
		uc.log.Warnf("sendMessage: devicesRepo.GetDevicesForUsers: %s", err.Error())
		return false
	}

	if len(devices) == 0 {
		uc.log.Debug("sendMessage: No devices found")
		return true
	}

	userSettings, err := uc.iam.GetUsersSettings(ctx, msg.UsersIds)

	userDevicesMap := make(map[int64][]*ent.Device)
	for _, device := range devices {
		userDevicesMap[device.UserID] = append(userDevicesMap[device.UserID], device)
	}

	inactiveTokens := make([]string, 0)

	candidatesToReFetch := make([]int64, 0)

	for userID, userDevices := range userDevicesMap {
		badges, badgeErr := uc.badgeClient.GetBadges(ctx, userID)
		if badgeErr != nil && isFirst {
			candidatesToReFetch = append(candidatesToReFetch, userID)

			uc.log.Warnf("sendMessage: failed to get badges for user %d: %v", userID, badgeErr)
			continue
		}

		totalBadges := int64(1)
		for _, count := range badges {
			totalBadges += count
		}

		if isFirst {
			err = uc.badgeClient.IncrementBadge(ctx, userID, msg.Type)
			if err != nil {
				uc.log.Warnf("sendMessage: failed to increment badge for user %d: %v", userID, err)
			}
		}

		badgeCount := int(totalBadges)

		withSound, withVibration := uc.userSettings(userID, userSettings)

		userInactiveTokens := uc.sendFcmMessageToUserDevices(
			ctx, msg, badgeCount, userDevices, withSound, withVibration,
		)
		if len(userInactiveTokens) > 0 {
			inactiveTokens = append(inactiveTokens, userInactiveTokens...)
		}
	}

	go uc.deleteInactiveTokens(ctx, inactiveTokens)

	if len(candidatesToReFetch) > 0 {
		uc.log.Debugf("sendMessage: re-fetching badges for %v", candidatesToReFetch)
		uc.fetchBadges(ctx, candidatesToReFetch)
		newMsg := msg
		newMsg.UsersIds = candidatesToReFetch
		uc.sendMessage(ctx, newMsg, false)
	}

	return err == nil
}

func (uc *FcmUsecase) userSettings(userID int64, userSettings map[int64]map[string]string) (bool, bool) {
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

func (uc *FcmUsecase) sendFcmMessageToUserDevices(
	ctx context.Context, msg u_struc.FirebaseNotification, badgeCount int, userDevices []*ent.Device,
	withSound, withVibration bool,
) []string {
	inactiveTokens := make([]string, 0)
	baseMessage := &messaging.Message{
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					MutableContent: true,
					Badge:          &badgeCount,
				},
			},
		},
		Android: &messaging.AndroidConfig{
			Notification: &messaging.AndroidNotification{
				NotificationCount: &badgeCount,
			},
		},
	}

	if withSound {
		baseMessage.APNS.Payload.Aps.Sound = "default"
		baseMessage.Android.Notification.Sound = "default"
	} else {
		baseMessage.APNS.Payload.Aps.Sound = ""
		baseMessage.Android.Notification.Sound = ""
	}

	if withVibration {
		if baseMessage.Data == nil {
			baseMessage.Data = make(map[string]string)
		}
		baseMessage.Data["vibrate"] = "true"
	}

	if len(msg.Data) > 0 {
		if msg.Type == u_struc.Chat {
			uc.processChatNotification(ctx, &msg, userDevices)
		}
		baseMessage.Data = msg.Data
	}

	if msg.Title != "" || msg.Body != "" || msg.Image != "" {
		baseMessage.Notification = &messaging.Notification{
			Title:    msg.Title,
			Body:     msg.Body,
			ImageURL: msg.Image,
		}
	}

	langDevicesMap := make(map[string][]*ent.Device)
	for _, device := range userDevices {
		lang := "null"
		if device.Language != "" {
			lang = device.Language
		}
		langDevicesMap[lang] = append(langDevicesMap[lang], device)
	}

	for lang, langDevices := range langDevicesMap {
		message := *baseMessage

		if lang != "null" {
			dto := &data.NotificationDto{}
			if err := dto.ParseAndSetNotificationData(msg.Data); err == nil && dto.Type != nil {
				localizedBody, localizedErr := uc.localizer.GetLocalizedMessage(
					lang, *dto.Type, dto.GetConvertedMap(), dto.PluralCount,
				)
				if localizedErr == nil && message.Notification != nil {
					message.Notification.Body = localizedBody
				}
			}
		}

		for _, device := range langDevices {
			message.Token = device.Token
			err := uc.fcmClient.Send(ctx, device.Token, &message)
			if err != nil {
				uc.log.Debugf("sendMessage: invalid token %s: %v", device.Token, err)
				inactiveTokens = append(inactiveTokens, device.Token)
			} else {
				uc.log.Debugf("sendMessage: sent successfully (%s)", message.Notification.Body)
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

	receivedUserID := userDevices[0].UserID

	ctxWithUserID := auth.AppendAuthIds(ctx, receivedUserID, 0)
	contacts, err := uc.contactsRemote.GetContactsByUserId(ctxWithUserID, authorIDInt)
	if err != nil {
		uc.log.Debugf("failed to fetch contact name: %v", err)
		return authorName
	}

	for _, contact := range contacts {
		if contact.GetUserId() == authorIDInt {
			if contact.GetLabel() != "" {
				return contact.GetLabel()
			}
			break
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

func (uc *FcmUsecase) processChatNotification(
	ctx context.Context, msg *u_struc.FirebaseNotification, userDevices []*ent.Device,
) {
	var user map[string]interface{}
	if userJSON, ok := msg.Data["user"]; ok {
		err := json.Unmarshal([]byte(userJSON), &user)
		if err != nil {
			uc.log.Debugf("failed to parse user json: %v", err)
		}
	}

	authorName := msg.Title
	if len(user) > 0 {
		authorNameFromUser := uc.getAuthorNameFromUser(ctx, user, userDevices)
		if authorNameFromUser != "" {
			authorName = authorNameFromUser
		}
	}

	authorAvatar := uc.getAuthorAvatar(user)
	title := msg.Title
	imageURL := msg.Image

	if chatJSON, ok := msg.Data["chat"]; ok {
		var chat map[string]interface{}
		err := json.Unmarshal([]byte(chatJSON), &chat)
		if err != nil {
			uc.log.Debugf("failed to parse chat json: %v", err)
			return
		}

		chatType, _ := chat["type"].(string)

		if chatType == "GROUP" {
			groupName, _ := chat["title"].(string)
			if groupName != "" {
				title = groupName
				msg.Body = authorName + ": " + msg.Body
			}

			if cover, ok := chat["cover"].(string); ok && cover != "" {
				imageURL = cover
			}
		} else {
			title = authorName
			if authorAvatar != "" {
				imageURL = authorAvatar
			}
		}
	} else if _, ok := msg.Data["chatId"]; ok {
		title = authorName
		if authorAvatar != "" {
			imageURL = authorAvatar
		}
	}

	msg.Title = title
	if imageURL != "" {
		msg.Image = imageURL
	}
}

func (uc *FcmUsecase) deleteInactiveTokens(ctx context.Context, tokens []string) {
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
