package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/config"
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
	log    *log.Helper
}

func NewSmsUsecase(c config.Config, logger log.Logger) (*SmsUsecase, error) {
	endpoint, err := c.Value("SMSC_ENDPOINT").String()
	if err != nil {
		return nil, err
	}
	login, err := c.Value("SMSC_LOGIN").String()
	if err != nil {
		return nil, err
	}
	password, err := c.Value("SMSC_PASSWORD").String()
	if err != nil {
		return nil, err
	}

	client, err := smsc.New(smsc.Config{
		URL:         endpoint,
		Login:       login,
		PasswordMD5: password,
	})
	if err != nil {
		log.Fatal(err)
	}

	return &SmsUsecase{client: client, log: log.NewHelper(logger)}, nil
}

func (uc *SmsUsecase) SendSms(ctx context.Context, sms *Sms) error {
	uc.log.WithContext(ctx).Infof("Send sms to %s: %s", sms.Phone, sms.Message)

	result, err := uc.client.Send(sms.Message, []string{sms.Phone})

	uc.log.Infof("SMS sent with result: %s, %s", result, err)

	return err
}
