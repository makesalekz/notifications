package service

import (
	"context"
	"fmt"

	v1 "notifications/api/send/v1"
	"notifications/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type SenderService struct {
	v1.UnimplementedSenderServer

	uc  *biz.SmsUsecase
	log *log.Helper
}

func NewSenderService(logger log.Logger, uc *biz.SmsUsecase) *SenderService {
	return &SenderService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *SenderService) PersonalSmsSender(ctx context.Context, req *v1.PersonalSmsSenderRequest) (*v1.PersonalSmsSenderReply, error) {
	err := s.uc.SendSms(ctx, &biz.Sms{
		Phone:   req.Phone,
		Message: req.Message,
	})
	if err != nil {
		s.log.Errorf("uc.AuthUserByPhone: ", err)
		return &v1.PersonalSmsSenderReply{
			Result: fmt.Sprintf("error: %v", err),
		}, nil
	}

	return &v1.PersonalSmsSenderReply{
		Result: "success",
	}, nil
}
