package biz

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/nats-io/nats.go/jetstream"

	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/internal/data"
	"gitlab.calendaria.team/services/notifications/internal/data/dialer"
	u_struc "gitlab.calendaria.team/services/utils/v2/struc"
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
	badgeClient       data.DragonflyClient
	chatsRemote       dialer.IChatsRemote
	eventsRemote      dialer.IEventsRemote
}

func NewFcmUsecase(
	logger log.Logger,
	devicesRepo data.DevicesRepo,
	notificationsRepo data.NotificationsRepo,
	localizer *data.Localizer,
	qm u_nats.IQueueManager,
	badgeClient data.DragonflyClient,
	fcmClient data.FcmClient,
	chatsRemote dialer.IChatsRemote,
	eventsRemote dialer.IEventsRemote,
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
	}

	qm.AddConsumer(QueueFCM, uc.sendNotifications)

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

	userDevicesMap := make(map[int64][]*ent.Device)
	for _, device := range devices {
		userDevicesMap[device.UserID] = append(userDevicesMap[device.UserID], device)
	}

	inactiveTokens := make([]string, 0)

	var reFetchMu sync.Mutex
	candidatesToReFetch := make([]int64, 0)

	for userID, userDevices := range userDevicesMap {
		badges, badgeErr := uc.badgeClient.GetBadges(ctx, userID)
		if badgeErr != nil && isFirst {
			reFetchMu.Lock()
			candidatesToReFetch = append(candidatesToReFetch, userID)
			reFetchMu.Unlock()

			uc.log.Warnf("sendMessage: failed to get badges for user %d: %v", userID, badgeErr)
			continue
		}

		totalBadges := int64(1)
		for _, count := range badges {
			totalBadges += count
		}

		err = uc.badgeClient.IncrementBadge(ctx, userID, msg.Type)
		if err != nil {
			uc.log.Warnf("sendMessage: failed to increment badge for user %d: %v", userID, err)
		}

		badgeCount := int(totalBadges)

		userInactiveTokens := uc.sendFcmMessageToUserDevices(ctx, msg, badgeCount, userDevices)
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

func (uc *FcmUsecase) sendFcmMessageToUserDevices(
	ctx context.Context, msg u_struc.FirebaseNotification, badgeCount int, userDevices []*ent.Device,
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

	if len(msg.Data) > 0 {
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

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var (
		eventsCountMap = make(map[int64]int32)
		chatsCountMap  = make(map[int64]int32)
		mu             sync.Mutex
		wg             sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		events, err := uc.eventsRemote.GetEventsCount(ctxWithTimeout, userIDs)
		if err != nil {
			uc.log.Errorf("fetchBadges: failed to get events count: %v", err)
			return
		}

		mu.Lock()
		for k, v := range events {
			eventsCountMap[k] = v
		}
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		chats, err := uc.chatsRemote.CountUnreadMessages(ctxWithTimeout, userIDs)
		if err != nil {
			uc.log.Errorf("fetchBadges: failed to get chats count: %v", err)
			return
		}

		mu.Lock()
		for k, v := range chats {
			chatsCountMap[k] = v
		}
		mu.Unlock()
	}()

	wg.Wait()

	for _, userID := range userIDs {
		contactsCount, err := uc.notificationsRepo.CountUnreadNotificationsByType(ctx, userID, u_struc.Contact.Value())
		if err != nil {
			uc.log.Errorf("fetchBadges: failed to get contacts count: %v", err)
			contactsCount = 0
		}

		badges := make(map[u_struc.NotificationType]int64)

		mu.Lock()
		if count, ok := eventsCountMap[userID]; ok {
			badges[u_struc.Event] = int64(count)
		}

		if count, ok := chatsCountMap[userID]; ok {
			badges[u_struc.Chat] = int64(count)
		}
		mu.Unlock()

		if contactsCount > 0 {
			badges[u_struc.Contact] = int64(contactsCount)
		}

		err = uc.badgeClient.SetBadges(ctx, userID, badges)
		if err != nil {
			uc.log.Errorf("fetchBadges: failed to set badges for user %d: %v", userID, err)
		} else {
			uc.log.Infof("fetchBadges: updated badges for user %d: %v", userID, badges)
		}
	}
}
