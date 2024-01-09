package service

import (
	"context"
	"time"

	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/internal/biz"
	"gitlab.calendaria.team/services/notifications/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type NotificationsService struct {
	v1.UnimplementedNotificationsServer

	sh *ServiceHelper
	nu *biz.NotificationsUsecase
}

func NewNotificationsService(
	nu *biz.NotificationsUsecase,
	sh *ServiceHelper,
) *NotificationsService {
	return &NotificationsService{
		nu: nu,
		sh: sh,
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
	actorId, err := s.sh.GetActorId(ctx, req.ActorId)
	if err != nil {
		return nil, err
	}

	list, err := s.nu.ListNotifications(
		ctx,
		actorId,
		&data.FilterNotificationsDto{
			Type: req.Type,
		},
		req.Paginate,
	)
	if err != nil {
		return nil, err
	}

	return &v1.ListNotificationsReply{
		Notifications: replyNotifications(list.Notifications),
		Paginate:      list.Paginate,
	}, nil
}

func (s *NotificationsService) GetNotificationsCounters(ctx context.Context, req *v1.NotificationRequest) (*v1.NotificationCountersReply, error) {
	actorId, err := s.sh.GetActorId(ctx, req.ActorId)
	if err != nil {
		return nil, err
	}

	reply, err := s.nu.GetNotificationCounters(ctx, actorId)
	if err != nil {
		return nil, err
	}

	return &v1.NotificationCountersReply{
		UnreadCount: *reply,
	}, err
}

func (s *NotificationsService) DoActionOnNotification(ctx context.Context, req *v1.DoActionOnNotificationRequest) (*utils_v1.EmptyReply, error) {
	actorId, err := s.sh.GetActorId(ctx, req.ActorId)
	if err != nil {
		return nil, err
	}

	switch req.Action {
	case "read":
		return &utils_v1.EmptyReply{}, s.nu.ReadNotification(ctx, actorId, req.NotificationId, req.Type)
	}

	return &utils_v1.EmptyReply{}, nil
}
