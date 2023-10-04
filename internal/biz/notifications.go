package biz

import (
	"context"
	_ "embed"

	notifications_v1 "notifications/api/notifications/v1"
	send_v1 "notifications/api/send/v1"
	"notifications/ent"
	"notifications/internal/conf"
	"notifications/internal/data"

	consul "github.com/go-kratos/consul/registry"
	"github.com/go-kratos/kratos/v2/log"
)

type NotificationsList struct {
	Notifications []*ent.Notification
	NextFromId    *int64
	NextToId      *int64
}

// NotificationsUsecase is a Greeter usecase.
type NotificationsUsecase struct {
	conf              *conf.Bootstrap
	log               *log.Helper
	discovery         *consul.Registry
	jwt               *data.JwtProcessor
	notificationsRepo data.NotificationsRepo
}

// NewGreeterUsecase new a Greeter usecase.
func NewNotificationsUsecase(
	logger log.Logger,
	c *data.Config,
	jwt *data.JwtProcessor,
	notificationsRepo data.NotificationsRepo,
) (*NotificationsUsecase, error) {
	return &NotificationsUsecase{
		conf:              c.Bootstrap,
		log:               log.NewHelper(logger),
		discovery:         c.GetRegistry(),
		jwt:               jwt,
		notificationsRepo: notificationsRepo,
	}, nil
}

func (uc *NotificationsUsecase) CreateNotifications(ctx context.Context, data []*notifications_v1.NotificationDto) (int32, error) {
	return uc.notificationsRepo.CreateNotifications(ctx, data)
}

func (uc *NotificationsUsecase) ListNotifications(ctx context.Context, filter *data.FilterNotificationsDto) (*NotificationsList, error) {
	userId, ok := uc.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, send_v1.ErrorUnauthorized("Unauthorized")
	}
	filter.UserId = userId

	if filter.FromId != 0 {
		filter.Ascending = true
	}

	notifications, err := uc.notificationsRepo.ListNotifications(ctx, filter)
	if err != nil {
		return nil, err
	}

	var nextFromId, nextToId *int64
	if len(notifications) == int(filter.Limit) {
		if filter.Ascending {
			nextFromId = &notifications[0].ID
		} else {
			nextToId = &notifications[len(notifications)-1].ID
		}
	}

	return &NotificationsList{
		Notifications: notifications,
		NextFromId:    nextFromId,
		NextToId:      nextToId,
	}, nil
}
