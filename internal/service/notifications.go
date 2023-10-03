package service

import (
	"context"
	"time"

	v1 "notifications/api/notifications/v1"
	"notifications/ent"
	"notifications/internal/biz"
	"notifications/internal/data"

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

func replyNotifications(notifications []*ent.Notification) []*v1.Notification {
	reply := make([]*v1.Notification, len(notifications))

	for i, notification := range notifications {
		n := &v1.Notification{
			Id:        notification.ID,
			Type:      string(notification.Type),
			Title:     notification.Title,
			Text:      notification.Text,
			CreatedAt: notification.CreatedAt.Format(time.RFC3339),
		}
		if notification.EventID != nil {
			n.EventId = *notification.EventID
		}
		if notification.ContactID != nil {
			n.ContactId = *notification.ContactID
		}
		reply[i] = n
	}

	return reply
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
	list, err := s.nu.ListNotifications(ctx, &data.FilterNotificationsDto{
		FromId: req.FromId,
		ToId:   req.ToId,
		Limit:  req.Limit,
	})
	if err != nil {
		return nil, err
	}

	return &v1.ListNotificationsReply{
		Notifications: replyNotifications(list.Notifications),
		NextFromId:    list.NextFromId,
		NextToId:      list.NextToId,
	}, nil
}
