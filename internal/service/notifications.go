package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/internal/biz"
	"gitlab.calendaria.team/services/notifications/internal/data"
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

func replyNotifications(notifications []*biz.NotificationItem) []*v1.Notification {
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
	list, err := s.nu.ListNotifications(
		ctx,
		&data.FilterNotificationsDto{
			Type: req.Type,
		},
		&v1.PaginateRequest{
			FromId:     req.GetPaginate().GetFromId(),
			ToId:       req.GetPaginate().GetToId(),
			Limit:      req.GetPaginate().GetLimit(),
			Descending: req.GetPaginate().GetDescending(),
		},
	)
	if err != nil {
		return nil, err
	}

	return &v1.ListNotificationsReply{
		Notifications: replyNotifications(list.Notifications),
		Paginate:      list.Paginate,
	}, nil
}

func (s *NotificationsService) GetNotificationsCounters(ctx context.Context, req *v1.EmptyRequest) (*v1.NotificationCountersReply, error) {
	reply, err := s.nu.GetNotificationCounters(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.NotificationCountersReply{
		UnreadCount: *reply,
	}, err
}

func (s *NotificationsService) DoActionOnNotification(ctx context.Context, req *v1.DoActionOnNotificationRequest) (*v1.EmptyReply, error) {
	switch req.Action {
	case "read":
		return &v1.EmptyReply{}, s.nu.ReadNotification(ctx, req.NotificationId)
	}

	return &v1.EmptyReply{}, nil
}
