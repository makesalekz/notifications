package biz

import (
	"context"
	"encoding/json"
	"notifications/internal/data"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/nats-io/nats.go"
)

// SmsUsecase is a Greeter usecase.
type FcmUsecase struct {
	client      *messaging.Client
	log         *log.Helper
	devicesRepo data.DevicesRepo
	queue       *Queue
}

func NewFcmUsecase(c *data.Config, logger log.Logger, devicesRepo data.DevicesRepo, qm *QueueManager) (*FcmUsecase, error) {
	app, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	uc := &FcmUsecase{
		client:      client,
		log:         log.NewHelper(logger),
		devicesRepo: devicesRepo,
	}

	uc.queue = qm.Create("fcm", uc.sendNotifications)

	return uc, nil
}

func (uc *FcmUsecase) sendNotifications(ctx context.Context, m *nats.Msg) bool {
	notification := Notification{}
	err := json.Unmarshal(m.Data, &notification)
	if err != nil {
		uc.log.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		return true
	}

	return uc.sendMessage(ctx, notification)
}

func (uc *FcmUsecase) RegisterDevice(ctx context.Context, userId int64, token string) error {
	_, err := uc.devicesRepo.CreateDevice(ctx, userId, token)

	return err
}

func (uc *FcmUsecase) UnregisterDevice(ctx context.Context, userId int64, token string) error {
	_, err := uc.devicesRepo.DeleteDevice(ctx, userId, token)

	return err
}

func (uc *FcmUsecase) sendMessage(ctx context.Context, msg Notification) bool {
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

	_, err = uc.client.SendEachForMulticast(ctx, message)
	if err != nil {
		uc.log.Warnf("sendMessage: client.SendEachForMulticast: %s", err.Error())
	}

	return err == nil
}
