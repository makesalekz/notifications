package biz

import (
	"context"
	"fmt"
	"notifications/internal/data"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/koorgoo/smsc"
)

type Sms struct {
	Phone   string
	Message string
}

// SmsUsecase is a Greeter usecase.
type SmsUsecase struct {
	client *smsc.Client
	conf   *data.Config
	log    *log.Helper
}

func NewSmsUsecase(c *data.Config, logger log.Logger) (*SmsUsecase, error) {
	return &SmsUsecase{
		conf: c,
		log:  log.NewHelper(logger),
	}, nil
}

func (uc *SmsUsecase) getClient(ctx context.Context) (*smsc.Client, error) {
	if uc.client != nil {
		return uc.client, nil
	}

	endpoint, err := uc.conf.Value("SMSC_ENDPOINT").String()
	if err != nil {
		return nil, err
	}
	smscCredentials, err := uc.conf.ReadSecretsFor(context.Background(), "smsc")
	if err != nil {
		return nil, err
	}

	login, ok := smscCredentials["login"].(string)
	if !ok {
		return nil, fmt.Errorf("SMSC Login is not set: %v", smscCredentials)
	}
	password, ok := smscCredentials["password"].(string)
	if !ok {
		return nil, fmt.Errorf("SMSC Password is not set: %v", smscCredentials)
	}

	client, err := smsc.New(smsc.Config{
		URL:         endpoint,
		Login:       login,
		PasswordMD5: password,
	})
	if err != nil {
		return nil, err
	}

	uc.client = client

	return uc.client, nil
}

func (uc *SmsUsecase) SendSms(ctx context.Context, sms *Sms) error {
	client, err := uc.getClient(ctx)
	if err != nil {
		return err
	}

	uc.log.WithContext(ctx).Infof("Send sms to %s: %s", sms.Phone, sms.Message)

	result, err := client.Send(sms.Message, []string{sms.Phone})

	uc.log.Debugf("SMS sent with result: %s", result)

	return err
}
