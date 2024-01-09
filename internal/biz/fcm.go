package biz

import (
	"context"
	"encoding/json"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/go-kratos/kratos/v2/log"
	nnats "github.com/nats-io/nats.go"
	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/internal/data"
	"gitlab.calendaria.team/services/utils/v1/nats"
)

type FirebaseNotification struct {
	UsersIds []int64
	Title    string
	Body     string
	Image    string
	Data     map[string]string
}

// SmsUsecase is a Greeter usecase.
type FcmUsecase struct {
	client            *messaging.Client
	log               *log.Helper
	devicesRepo       data.DevicesRepo
	notificationsRepo data.NotificationsRepo
	qm                *nats.QueueManager
}

func NewFcmUsecase(
	logger log.Logger,
	devicesRepo data.DevicesRepo,
	notificationsRepo data.NotificationsRepo,
	qm *nats.QueueManager,
) (*FcmUsecase, error) {
	uc := &FcmUsecase{
		log:               log.NewHelper(logger),
		devicesRepo:       devicesRepo,
		notificationsRepo: notificationsRepo,
		qm:                qm,
	}

	if os.Getenv("DEBUG") == "" {
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

func (uc *FcmUsecase) sendNotifications(ctx context.Context, m *nnats.Msg) bool {
	notification := FirebaseNotification{}
	err := json.Unmarshal(m.Data, &notification)
	if err != nil {
		uc.log.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		return true
	}

	uc.log.Debugf("sendNotifications: %v", notification)

	ok := uc.sendMessage(ctx, notification)
	if ok {
		listDto := make([]*v1.NotificationDto, len(notification.UsersIds))
		type NotificationData struct {
			Id int64 `json:"id"`
		}
		for i, userId := range notification.UsersIds {
			dto := &v1.NotificationDto{
				UserId: userId,
				Title:  notification.Title,
				Text:   notification.Body,
			}

			var notificationData NotificationData
			if notification.Data["event"] != "" {
				err := json.Unmarshal([]byte(notification.Data["event"]), &notificationData)
				if err != nil {
					uc.log.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
				}

				dto.EventId = notificationData.Id
			} else {
				dto.EventId = 0
			}

			if notification.Data["contact"] != "" {
				err := json.Unmarshal([]byte(notification.Data["contact"]), &notificationData)
				if err != nil {
					uc.log.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
				}

				dto.ContactId = notificationData.Id
			} else {
				dto.ContactId = 0
			}

			listDto[i] = dto
		}

		_, err := uc.notificationsRepo.CreateNotifications(ctx, listDto)
		if err != nil {
			uc.log.Errorf("sendNotifications: notificationsRepo.CreateNotifications: %s", err.Error())
		}
	}

	return ok
}

func (uc *FcmUsecase) RegisterDevice(ctx context.Context, userId int64, token string) error {
	return uc.devicesRepo.CreateDevice(ctx, userId, token)
}

func (uc *FcmUsecase) UnregisterDevice(ctx context.Context, userId int64, token string) error {
	_, err := uc.devicesRepo.DeleteDevice(ctx, userId, token)

	return err
}

func (uc *FcmUsecase) sendMessage(ctx context.Context, msg FirebaseNotification) bool {
	message := &messaging.MulticastMessage{}
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

	tokens := make([]string, len(devices))
	for i, device := range devices {
		tokens[i] = device.Token
	}

	message.Tokens = tokens

	if uc.client != nil {
		_, err = uc.client.SendEachForMulticast(ctx, message)
		if err != nil {
			uc.log.Warnf("sendMessage: client.SendEachForMulticast: %s", err.Error())
		}
	} else {
		uc.log.Debug("sendMessage (debug): ", message)
	}

	return err == nil
}
