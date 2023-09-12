package service

import (
	"context"
	"fmt"
	"strconv"

	v1 "notifications/api/send/v1"
	"notifications/internal/biz"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

type SenderService struct {
	v1.UnimplementedSenderServer

	log *log.Helper
	jwt *biz.JwtProcessor
	sms *biz.SmsUsecase
	fcm *biz.FcmUsecase
}

func NewSenderService(logger log.Logger, jwt *biz.JwtProcessor, sms *biz.SmsUsecase, fcm *biz.FcmUsecase) *SenderService {
	return &SenderService{
		log: log.NewHelper(logger),
		jwt: jwt,
		sms: sms,
		fcm: fcm,
	}
}

func (s *SenderService) CreateFcmDevice(ctx context.Context, req *v1.FcmDeviceRequest) (*v1.FcmDeviceReply, error) {
	userId, ok := s.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, v1.ErrorUnauthorized("Unauthorized")
	}

	err := s.fcm.RegisterDevice(ctx, userId, req.Token)

	if err != nil {
		s.log.Errorf("fcm.CreateDevice: ", err)
		return nil, errors.InternalServer("internal", "internal error")
	}

	return &v1.FcmDeviceReply{Result: "success"}, nil
}

func (s *SenderService) DeleteFcmDevice(ctx context.Context, req *v1.FcmDeviceRequest) (*v1.FcmDeviceReply, error) {
	userId, ok := s.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, v1.ErrorUnauthorized("Unauthorized")
	}

	err := s.fcm.UnregisterDevice(ctx, userId, req.Token)

	if err != nil {
		s.log.Errorf("fcm.CreateDevice: ", err)
		return nil, errors.InternalServer("internal", "internal error")
	}

	return &v1.FcmDeviceReply{Result: "success"}, nil
}

func (s *SenderService) PersonalSmsSender(ctx context.Context, req *v1.PersonalSmsSenderRequest) (*v1.PersonalSmsSenderReply, error) {
	err := s.sms.SendSms(ctx, &biz.Sms{
		Phone:   req.Phone,
		Message: req.Message,
	})
	if err != nil {
		s.log.Errorf("sms.AuthUserByPhone: ", err)
		return &v1.PersonalSmsSenderReply{
			Result: fmt.Sprintf("error: %v", err),
		}, nil
	}

	return &v1.PersonalSmsSenderReply{
		Result: "success",
	}, nil
}

func (s *SenderService) PersonalFcmSender(ctx context.Context, req *v1.PersonalFcmSenderRequest) (*v1.PersonalFcmSenderReply, error) {
	userId, ok := strconv.ParseInt(req.UserId, 10, 64)
	if ok != nil {
		return nil, v1.ErrorUnauthorized("Unauthorized")
	}

	err := s.fcm.SendMessage(ctx, userId, biz.FCMMessage{
		Title: req.Title,
		Body:  req.Body,
		Image: req.Image,
		Data:  req.Data,
	})
	if err != nil {
		s.log.Errorf("sms.SendMessage: ", err)
		return &v1.PersonalFcmSenderReply{
			Result: fmt.Sprintf("error: %v", err),
		}, nil
	}

	return &v1.PersonalFcmSenderReply{
		Result: "success",
	}, nil
}
