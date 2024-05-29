package biz

import (
	"context"
	"os"

	"gitlab.calendaria.team/services/notifications/internal/data"
	"gitlab.calendaria.team/services/utils/v1/config"

	"github.com/go-kratos/kratos/v2/log"
)

type Sms struct {
	Phone   string
	Message string
}

// SmsUsecase is a Greeter usecase.
type SmsUsecase struct {
	config  *config.Config
	log     *log.Helper
	smsRepo data.SmsRepo
}

func NewSmsUsecase(c *config.Config, logger log.Logger, smsRepo data.SmsRepo) (*SmsUsecase, error) {
	return &SmsUsecase{
		config:  c,
		log:     log.NewHelper(log.With(logger, "module", "usecase/sms")),
		smsRepo: smsRepo,
	}, nil
}

func (uc *SmsUsecase) SendSms(_ context.Context, sms *Sms) error {
	uc.log.Infof("Sending sms to %s: %s", sms.Phone, sms.Message)

	debug := os.Getenv("DEBUG")
	if debug == "" {
		// Get the SMSC endpoint and credentials
		endpoint, err := uc.smsRepo.GetSmsEndpoint()
		if err != nil {
			return err
		}
		login, err := uc.smsRepo.GetSmsLogin()
		if err != nil {
			return err
		}
		password, err := uc.smsRepo.GetSmsPassword()
		if err != nil {
			return err
		}

		result, err := SendSms(SmsRequest{
			Endpoint: endpoint,
			Login:    login,
			Password: password,
			Message:  sms.Message,
			Phones:   []string{sms.Phone},
		})
		if err != nil {
			return err
		}

		uc.log.Infof("SMS sent with result: %s", result)
	} else {
		uc.log.Infof("[DEBUG] SMS sent with result: OK - 1 SMS, ID - TEST")
	}

	return nil
}
