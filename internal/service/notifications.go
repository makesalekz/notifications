package service

import (
	"context"
	"time"

	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/internal/biz"
	"gitlab.calendaria.team/services/notifications/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
	"gitlab.calendaria.team/services/utils/v2/auth"
)

type NotificationsService struct {
	v1.UnimplementedNotificationsServer

	nu *biz.NotificationsUsecase
}

func NewNotificationsService(
	nu *biz.NotificationsUsecase,
) *NotificationsService {
	return &NotificationsService{
		nu: nu,
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
		if notification.TaskID != nil {
			n.TaskId = *notification.TaskID
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
	actorId := auth.GetActorIdFromContext(ctx)
	if actorId == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}

	list, err := s.nu.ListNotifications(
		ctx,
		req.Language,
		&data.FilterNotificationsDto{
			UserId: actorId,
			Type:   req.Type,
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

func (s *NotificationsService) GetNotificationsCounters(ctx context.Context, req *utils_v1.EmptyRequest) (*v1.NotificationCountersReply, error) {
	actorId := auth.GetActorIdFromContext(ctx)
	if actorId == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
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
	actorId := auth.GetActorIdFromContext(ctx)
	if actorId == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}

	switch req.Action {
	case "read":
		return &utils_v1.EmptyReply{}, s.nu.ReadNotification(ctx, actorId, req.NotificationId, req.Type)
	}

	return &utils_v1.EmptyReply{}, nil
}
