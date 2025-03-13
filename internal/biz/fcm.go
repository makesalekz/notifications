package biz

import (
	"context"
	"encoding/json"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/go-kratos/kratos/v2/log"
	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/internal/data"
	"gitlab.calendaria.team/services/notifications/messages"
	u_nats "gitlab.calendaria.team/services/utils/v2/nats"

	"github.com/nats-io/nats.go/jetstream"
)

// SmsUsecase is a Greeter usecase.
type FcmUsecase struct {
	client            *messaging.Client
	log               *log.Helper
	devicesRepo       data.DevicesRepo
	localizer         *data.Localizer
	notificationsRepo data.NotificationsRepo
	qm                u_nats.IQueueManager
}

func NewFcmUsecase(
	logger log.Logger,
	devicesRepo data.DevicesRepo,
	notificationsRepo data.NotificationsRepo,
	localizer *data.Localizer,
	qm u_nats.IQueueManager,
) (*FcmUsecase, error) {
	uc := &FcmUsecase{
		log:               log.NewHelper(logger),
		devicesRepo:       devicesRepo,
		notificationsRepo: notificationsRepo,
		qm:                qm,
		localizer:         localizer,
	}

	if os.Getenv("FIREBASE_CONFIG") != "" {
		app, err := firebase.NewApp(context.Background(), nil)
		if err != nil {
			return nil, err
		}

		ctx := context.Background()
		client, err := app.Messaging(ctx)
		if err != nil {
			return nil, err
		}
		uc.client = client
	}

	qm.AddConsumer(QueueFCM, uc.sendNotifications)

	return uc, nil
}

func (uc *FcmUsecase) sendNotifications(ctx context.Context, m jetstream.Msg) bool {
	notification := messages.FirebaseNotification{}
	err := json.Unmarshal(m.Data(), &notification)
	if err != nil {
		uc.log.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		return true
	}

	uc.log.Debugf("sendNotifications: %v", notification)

	ok := uc.sendMessage(ctx, notification)
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

		if notification.Type.IsValid() && notification.Title != "" {
			_, err2 := uc.notificationsRepo.CreateNotifications(ctx, listDto)
			if err2 != nil {
				uc.log.Errorf("sendNotifications: notificationsRepo.CreateNotifications: %s", err2.Error())
			}
		}
	}

	return ok
}

func (uc *FcmUsecase) sendMessage(ctx context.Context, msg messages.FirebaseNotification) bool {
	defaultBadge := 1
	message := &messaging.MulticastMessage{
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					MutableContent: true,
					Badge:          &defaultBadge,
				},
			},
		},
	}

	if msg.Badge != nil || *msg.Badge != 1 {
		message.APNS.Payload.Aps.Badge = msg.Badge
	}

	empty := true

	if len(msg.Data) > 0 {
		message.Data = msg.Data
		empty = false
	}

	if msg.Title != "" || msg.Body != "" || msg.Image != "" {
		message.Notification = &messaging.Notification{
			Title:    msg.Title,
			Body:     msg.Body,
			ImageURL: msg.Image,
		}
		empty = false
	}

	if empty {
		uc.log.Error("sendMessage: Empty message")
		return true
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

	multicastMessages := uc.splitMessageToLanguages(devices, message)

	if uc.client != nil {
		for _, multicastMessage := range multicastMessages {
			uc.log.Debugf("sendMessage: send %d messages", len(multicastMessages))
			_, err = uc.client.SendEachForMulticast(ctx, multicastMessage)
			if err != nil {
				uc.log.Warnf("sendMessage: client.SendEachForMulticast: %s", err.Error())
			} else {
				uc.log.Debug("sendMessage: sent successfully")
			}
		}
	} else {
		uc.log.Debug("sendMessage (debug): ", message)
	}

	return err == nil
}

func (uc *FcmUsecase) splitMessageToLanguages(
	devices []*ent.Device,
	msg *messaging.MulticastMessage,
) []*messaging.MulticastMessage {
	langs := map[string][]string{}
	for _, device := range devices {
		lang := "null"
		if device.Language != "" {
			lang = device.Language
		}

		_, ok := langs[lang]
		if !ok {
			langs[lang] = []string{device.Token}
			continue
		}
		langs[lang] = append(langs[lang], device.Token)
	}

	localizedMsgs := make([]*messaging.MulticastMessage, 0, len(langs))
	for lang, tokens := range langs {
		localizedMessage := &messaging.MulticastMessage{
			Tokens: tokens,
		}

		if len(msg.Data) > 0 {
			localizedMessage.Data = msg.Data
		}

		if msg.Notification != nil {
			localizedMessage.Notification = &messaging.Notification{
				Title:    msg.Notification.Title,
				Body:     msg.Notification.Body,
				ImageURL: msg.Notification.ImageURL,
			}
		}

		localizedMsgs = append(localizedMsgs, localizedMessage)

		if lang == "null" {
			continue
		}

		dto := &data.NotificationDto{}
		err := dto.ParseAndSetNotificationData(msg.Data)
		if err != nil {
			uc.log.Errorf("splitMessageToLanguages: dto.ParseAndSetNotificationData: %s", err.Error())
			continue
		}

		if dto.Type == nil {
			continue
		}

		localizedBody, err := uc.localizer.GetLocalizedMessage(lang, *dto.Type, dto.GetConvertedMap(), dto.PluralCount)
		if err != nil {
			uc.log.Errorf("splitMessageToLanguages: localizer.GetLocalizedMessage: %s", err.Error())
			continue
		}

		// by pointer, also changes value in array
		localizedMessage.Notification.Body = localizedBody
	}

	return localizedMsgs
}

func (uc *FcmUsecase) RegisterDevice(ctx context.Context, device data.DeviceDto) error {
	err := uc.devicesRepo.CreateDevice(ctx, device)
	if err != nil {
		return err
	}

	if device.OldToken != "" {
		_, err = uc.devicesRepo.DeleteDevice(ctx, data.DeviceKey{
			UserID: device.UserID,
			Token:  device.OldToken,
		})
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
