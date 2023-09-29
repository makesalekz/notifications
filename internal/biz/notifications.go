package biz

import (
	"context"
	_ "embed"

	notifications_v1 "notifications/api/notifications/v1"
	"notifications/internal/conf"
	"notifications/internal/data"

	consul "github.com/go-kratos/consul/registry"
	"github.com/go-kratos/kratos/v2/log"
)

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

func (d *NotificationsUsecase) CreateNotifications(ctx context.Context, data []*notifications_v1.NotificationDto) (int32, error) {
	return d.notificationsRepo.CreateNotifications(ctx, data)
}
