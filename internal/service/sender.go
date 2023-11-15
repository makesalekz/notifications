package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/internal/biz"
	"gitlab.calendaria.team/services/notifications/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type SenderService struct {
	v1.UnimplementedSenderServer

	log *log.Helper
	jwt *data.JwtProcessor
	sms *biz.SmsUsecase
	fcm *biz.FcmUsecase
}

func NewSenderService(logger log.Logger, jwt *data.JwtProcessor, sms *biz.SmsUsecase, fcm *biz.FcmUsecase) *SenderService {
	return &SenderService{
		log: log.NewHelper(logger),
		jwt: jwt,
		sms: sms,
		fcm: fcm,
	}
}

func (s *SenderService) CreateFcmDevice(ctx context.Context, req *v1.FcmDeviceRequest) (*utils_v1.EmptyReply, error) {
	userId, ok := s.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, v1.ErrorUnauthorized("Unauthorized")
	}

	err := s.fcm.RegisterDevice(ctx, userId, req.Token)

	if err != nil {
		s.log.Errorf("fcm.RegisterDevice: %v", err)
		return nil, errors.InternalServer("internal", "Internal error")
	}

	return &utils_v1.EmptyReply{}, nil
}

func (s *SenderService) DeleteFcmDevice(ctx context.Context, req *v1.FcmDeviceRequest) (*utils_v1.EmptyReply, error) {
	userId, ok := s.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, v1.ErrorUnauthorized("Unauthorized")
	}

	err := s.fcm.UnregisterDevice(ctx, userId, req.Token)
	if err != nil {
		s.log.Errorf("fcm.UnregisterDevice: %v", err)
		return nil, errors.InternalServer("internal", "Internal error")
	}

	return &utils_v1.EmptyReply{}, nil
}

func (s *SenderService) PersonalSmsSender(ctx context.Context, req *v1.PersonalSmsSenderRequest) (*utils_v1.EmptyReply, error) {
	err := s.sms.SendSms(ctx, &biz.Sms{
		Phone:   req.Phone,
		Message: req.Message,
	})
	if err != nil {
		s.log.Errorf("sms.SendSms: %v", err)
		return nil, v1.ErrorSmsFailed("Internal error")
	}

	return &utils_v1.EmptyReply{}, nil
}
