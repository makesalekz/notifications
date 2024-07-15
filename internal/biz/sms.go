package biz

import (
	"context"

	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/internal/data"

	"github.com/go-kratos/kratos/v2/log"
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

func (uc *SmsUsecase) SendSms(ctx context.Context, sms data.Sms) error {
	_, err := uc.smscClient.SendSms(ctx, data.Sms{
		Message: sms.Message,
		Phones:  sms.Phones,
	})
	if err != nil {
		return v1.ErrorSmsFailed("failed to send sms: %v", err)
	}

	return nil
}
