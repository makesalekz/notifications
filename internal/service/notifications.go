package service

import (
	"context"

	v1 "notifications/api/notifications/v1"
	"notifications/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type NotificationsService struct {
	v1.UnimplementedNotificationsServer

	log *log.Helper
	nu  *biz.NotificationsUsecase
}

func NewNotificationsService(logger log.Logger, nu *biz.NotificationsUsecase) *NotificationsService {
	return &NotificationsService{
		log: log.NewHelper(logger),
		nu:  nu,
	}
}

func (s *NotificationsService) CreateNotifications(ctx context.Context, req *v1.CreateNotificationsRequest) (*v1.CreateNotificationsReply, error) {
	created, err := s.nu.CreateNotifications(ctx, req.Notifications)
	if err != nil {
		return nil, err
	}

	return &v1.CreateNotificationsReply{
		Created: created,
	}, nil
}
func (s *NotificationsService) ListNotifications(ctx context.Context, req *v1.ListNotificationsRequest) (*v1.ListNotificationsReply, error) {
	return &v1.ListNotificationsReply{}, nil
}
