package service

import (
	"context"

	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/internal/biz"
	"gitlab.calendaria.team/services/notifications/internal/biz/reply"
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

func (s *NotificationsService) CreateNotifications(
	ctx context.Context,
	req *v1.CreateNotificationsRequest,
) (*v1.CreateNotificationsReply, error) {
	created, err := s.nu.CreateNotifications(ctx, req.GetNotifications())
	if err != nil {
		return nil, err
	}

	return &v1.CreateNotificationsReply{
		Created: created,
	}, nil
}

func (s *NotificationsService) ListNotifications(
	ctx context.Context,
	req *v1.ListNotificationsRequest,
) (*v1.ListNotificationsReply, error) {
	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}

	language := req.GetLanguage()
	if language == "" {
		language = biz.DefaultLanguage
	}

	list, err := s.nu.ListNotifications(
		ctx,
		language,
		&data.FilterNotificationsDto{
			UserID: actorID,
			Type:   req.GetType(),
		},
		req.GetPaginate(),
	)
	if err != nil {
		return nil, err
	}

	return &v1.ListNotificationsReply{
		Notifications: reply.MapNotifications(list.Notifications),
		Paginate:      list.Paginate,
	}, nil
}

func (s *NotificationsService) GetNotificationsCounters(
	ctx context.Context,
	_ *utils_v1.EmptyRequest,
) (*v1.NotificationCountersReply, error) {
	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}

	reply, err := s.nu.GetNotificationCounters(ctx, actorID)
	if err != nil {
		return nil, err
	}

	return &v1.NotificationCountersReply{
		UnreadCount: *reply,
	}, err
}

func (s *NotificationsService) DoActionOnNotification(
	ctx context.Context,
	req *v1.DoActionOnNotificationRequest,
) (*utils_v1.EmptyReply, error) {
	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}

	if req.GetAction() == "read" {
		notificationType := req.GetType()
		return &utils_v1.EmptyReply{}, s.nu.ReadNotification(ctx, actorID, req.GetNotificationId(), notificationType)
	}

	return &utils_v1.EmptyReply{}, nil
}
