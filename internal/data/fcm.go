package data

import (
	"context"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/go-kratos/kratos/v2/log"
)

type FcmClient interface {
	Send(ctx context.Context, token string, message *messaging.Message) error
	SendMulticast(ctx context.Context, message *messaging.MulticastMessage) error
}

type fcmClient struct {
	client *messaging.Client
	log    *log.Helper
}

func NewFcmClient(logger log.Logger) (FcmClient, error) {
	l := log.NewHelper(logger)

	if os.Getenv("FIREBASE_CONFIG") == "" {
		l.Info("FIREBASE_CONFIG not set, running in debug mode")
		return &fcmClient{
			log: l,
		}, nil
	}

	app, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		return nil, err
	}

	return &fcmClient{
		client: client,
		log:    l,
	}, nil
}

func (c *fcmClient) Send(ctx context.Context, token string, message *messaging.Message) error {
	if c.client == nil {
		c.log.Debug("Send (debug mode): ", message)
		return nil
	}

	_, err := c.client.Send(ctx, message)
	if err != nil {
		if messaging.IsInvalidArgument(err) || messaging.IsUnregistered(err) {
			c.log.Debugf("Send: invalid token %s: %v", token, err)
			return err
		}
		c.log.Errorf("Send: %v", err)
		return err
	}

	c.log.Debugf("Send: sent successfully (%s)", message.Notification.Body)
	return nil
}

func (c *fcmClient) SendMulticast(ctx context.Context, message *messaging.MulticastMessage) error {
	if c.client == nil {
		c.log.Debug("SendMulticast (debug mode): ", message)
		return nil
	}

	_, err := c.client.SendEachForMulticast(ctx, message)
	if err != nil {
		if messaging.IsInvalidArgument(err) || messaging.IsUnregistered(err) {
			c.log.Debugf("SendMulticast: invalid tokens: %v", err)
			return err
		}
		c.log.Errorf("SendMulticast: %v", err)
		return err
	}

	c.log.Debugf("SendMulticast: sent successfully (%s)", message.Notification.Body)
	return nil
}
