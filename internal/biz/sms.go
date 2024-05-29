package biz

import (
	"context"
	"os"

	"github.com/go-kratos/kratos/v2/log"

	"gitlab.calendaria.team/services/notifications/internal/data"
)

// SmsUsecase is a Greeter usecase.
type SmsUsecase struct {
	log        *log.Helper
	smscClient data.SmscClient
}

func NewSmsUsecase(logger log.Logger, smscClient data.SmscClient) (*SmsUsecase, error) {
	return &SmsUsecase{
		log:        log.NewHelper(log.With(logger, "module", "usecase/sms")),
		smscClient: smscClient,
	}, nil
}

func (uc *SmsUsecase) SendSms(_ context.Context, sms data.Sms) error {
	debug := os.Getenv("DEBUG")
	if debug == "" {
		uc.log.Infof("Sending sms to %s: <message>", sms.Phones)

		result, err := uc.smscClient.SendSms(data.Sms{
			Message: sms.Message,
			Phones:  sms.Phones,
		})
		if err != nil {
			return err
		}

		uc.log.Infof("SMS sent with result: %s", result)
	} else {
		uc.log.Infof("Sending sms to %s: %s", sms.Phones, sms.Message)
		uc.log.Infof("[DEBUG] SMS sent with result: OK - 1 SMS, ID - TEST")
	}

	return nil
}
