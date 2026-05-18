package service

import (
	"context"

	v1 "github.com/makesalekz/notifications/api/notifications/v1"
	"github.com/makesalekz/notifications/internal/biz"
	"github.com/makesalekz/notifications/internal/data"
	utils_v1 "github.com/makesalekz/utils/api/utils/v1"
	"github.com/makesalekz/utils/v2/auth"
	u_struc "github.com/makesalekz/utils/v2/struc"
)

type SenderService struct {
	v1.UnimplementedSenderServer

	sms   *biz.SmsUsecase
	fcm   *biz.FcmUsecase
	email *biz.EmailUsecase
}

func NewSenderService(
	sms *biz.SmsUsecase,
	fcm *biz.FcmUsecase,
	email *biz.EmailUsecase,
) *SenderService {
	return &SenderService{
		sms:   sms,
		fcm:   fcm,
		email: email,
	}
}

func (s *SenderService) CreateFcmDevice(ctx context.Context, req *v1.FcmDataRequest) (*utils_v1.EmptyReply, error) {
	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}

	err := s.fcm.RegisterDevice(
		ctx, data.DeviceDto{
			DeviceKey: data.DeviceKey{
				UserID:   actorID,
				Token:    req.GetToken(),
				OldToken: req.GetOldToken(),
			},
			DeviceData: data.DeviceData{
				Language: req.GetLanguage(),
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return &utils_v1.EmptyReply{}, nil
}

func (s *SenderService) DeleteFcmDevice(ctx context.Context, req *v1.FcmDeviceRequest) (*utils_v1.EmptyReply, error) {
	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}

	err := s.fcm.UnregisterDevice(
		ctx, data.DeviceKey{
			UserID: actorID,
			Token:  req.GetToken(),
		},
	)
	if err != nil {
		return nil, err
	}

	return &utils_v1.EmptyReply{}, nil
}

func (s *SenderService) PersonalSmsSender(
	ctx context.Context,
	req *v1.PersonalSmsSenderRequest,
) (*utils_v1.EmptyReply, error) {
	err := s.sms.SendSms(
		ctx, data.Sms{
			Sender:  req.GetSender(),
			Message: req.GetMessage(),
			Phones:  []string{req.GetPhone()},
		},
	)
	if err != nil {
		return nil, err
	}

	return &utils_v1.EmptyReply{}, nil
}

func (s *SenderService) EmailSender(ctx context.Context, req *v1.EmailSenderRequest) (*utils_v1.EmptyReply, error) {
	language := req.GetLanguage()
	if language == "" {
		language = biz.DefaultLanguage
	}

	err := s.email.SendEmail(
		ctx, &u_struc.EmailDetails{
			Language: language,
			Type:     req.GetType(),
			Emails:   req.GetEmails(),
			Data:     req.GetData(),
		},
	)

	if err != nil {
		return nil, err
	}

	return &utils_v1.EmptyReply{}, nil
}
