package biz

import (
	"context"
	"notifications/internal/data"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/go-kratos/kratos/v2/log"
)

type FCMMessage struct {
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Image string            `json:"image"`
	Data  map[string]string `json:"data"`
}

// SmsUsecase is a Greeter usecase.
type FcmUsecase struct {
	client      *messaging.Client
	log         *log.Helper
	devicesRepo data.DevicesRepo
}

func NewFcmUsecase(c *data.Config, logger log.Logger, devicesRepo data.DevicesRepo) (*FcmUsecase, error) {
	app, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	return &FcmUsecase{
		client:      client,
		log:         log.NewHelper(logger),
		devicesRepo: devicesRepo,
	}, nil
}

func (uc *FcmUsecase) RegisterDevice(ctx context.Context, userId int64, token string) error {
	_, err := uc.devicesRepo.CreateDevice(ctx, userId, token)

	return err
}

func (uc *FcmUsecase) UnregisterDevice(ctx context.Context, userId int64, token string) error {
	_, err := uc.devicesRepo.DeleteDevice(ctx, userId, token)

	return err
}

func (uc *FcmUsecase) SendMessage(ctx context.Context, userId int64, msg FCMMessage) error {
	devices, err := uc.devicesRepo.GetDevicesForUser(ctx, userId)
	if err != nil {
		return err
	}

	tokens := make([]string, len(devices))
	for i, device := range devices {
		tokens[i] = device.Token
	}

	uc.log.Infof("Send push to %v", devices)

	result, err := uc.client.SendEachForMulticast(ctx, &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title:    msg.Title,
			Body:     msg.Body,
			ImageURL: msg.Image,
		},
		Data:   msg.Data,
		Tokens: tokens,
	})

	uc.log.Infof("Push sent with result: %s, %s", result, err)

	return err
}
