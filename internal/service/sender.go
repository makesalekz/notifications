package service

import (
	"context"

	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/internal/biz"
	"gitlab.calendaria.team/services/notifications/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
	"gitlab.calendaria.team/services/utils/v2/auth"
)

type SenderService struct {
	v1.UnimplementedSenderServer

	sms *biz.SmsUsecase
	fcm *biz.FcmUsecase
}

func NewSenderService(
	sms *biz.SmsUsecase,
	fcm *biz.FcmUsecase,
) *SenderService {
	return &SenderService{
		sms: sms,
		fcm: fcm,
	}
}

func (s *SenderService) CreateFcmDevice(ctx context.Context, req *v1.FcmDataRequest) (*utils_v1.EmptyReply, error) {
	actorId := auth.GetActorIdFromContext(ctx)
	if actorId == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}

	err := s.fcm.RegisterDevice(ctx, data.DeviceDto{
		DeviceKey: data.DeviceKey{
			UserId: actorId,
			Token:  req.Token,
		},
		DeviceData: data.DeviceData{
			Language: req.Language,
		},
	})
	if err != nil {
		return nil, err
	}

	return &utils_v1.EmptyReply{}, nil
}

func (s *SenderService) DeleteFcmDevice(ctx context.Context, req *v1.FcmDeviceRequest) (*utils_v1.EmptyReply, error) {
	actorId := auth.GetActorIdFromContext(ctx)
	if actorId == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}

	err := s.fcm.UnregisterDevice(ctx, data.DeviceKey{
		UserId: actorId,
		Token:  req.Token,
	})
	if err != nil {
		return nil, err
	}

	return &utils_v1.EmptyReply{}, nil
}

func (s *SenderService) PersonalSmsSender(ctx context.Context, req *v1.PersonalSmsSenderRequest) (*utils_v1.EmptyReply, error) {
	err := s.sms.SendSms(ctx, data.Sms{
		Message: req.Message,
		Phones:  []string{req.Phone},
	})
	if err != nil {
		return nil, err
	}

	return &utils_v1.EmptyReply{}, nil
}
