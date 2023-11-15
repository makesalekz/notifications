package biz

import (
	"context"

	consul "github.com/go-kratos/consul/registry"
	"github.com/go-kratos/kratos/v2/log"
	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/ent/enum"
	"gitlab.calendaria.team/services/notifications/internal/conf"
	"gitlab.calendaria.team/services/notifications/internal/data"
)

type NotificationItem struct {
	*ent.Notification
}

type NotificationsList struct {
	Notifications []*NotificationItem
	Paginate      *v1.PaginateReply
}

type NotificationsCounters map[string]int32

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

func (uc *NotificationsUsecase) CreateNotifications(ctx context.Context, data []*v1.NotificationDto) (int32, error) {
	newRecords, err := uc.notificationsRepo.CreateNotifications(ctx, data)
	if err != nil {
		if !ent.IsNotFound(err) {
			return 0, v1.ErrorDatabaseQuery("can't create notifactions: %v", err)
		}
	}

	return newRecords, nil
}

func (uc *NotificationsUsecase) ReadNotification(ctx context.Context, Id int64) error {
	userId, ok := uc.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return v1.ErrorUnauthorized("Unauthorized")
	}

	err := uc.notificationsRepo.ReadNotification(ctx, data.ReadNotificationDto{UserId: userId, NotificationId: Id})
	if err != nil {
		if !ent.IsNotFound(err) {
			return v1.ErrorDatabaseQuery("can't read notifaction: %v", err)
		}
		return v1.ErrorNotificationNotFound("there is no such notification")
	}

	return nil
}

func (uc *NotificationsUsecase) GetNotificationCounters(ctx context.Context) (*NotificationsCounters, error) {
	userId, ok := uc.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, v1.ErrorUnauthorized("Unauthorized")
	}

	lastReadNotification, err := uc.notificationsRepo.GetLastReadNotification(ctx, userId)
	if err != nil {
		if !ent.IsNotFound(err) {
			return nil, v1.ErrorDatabaseQuery("can't get last read notification")
		}

		err = uc.notificationsRepo.ReadNotification(ctx, data.ReadNotificationDto{UserId: userId, NotificationId: 0})
		if err != nil {
			return nil, v1.ErrorDatabaseQuery("can't read notification: %v", err)
		}

		lastReadNotification = &ent.LastReadNotification{LastReadID: 0}
	}

	counters, err := uc.notificationsRepo.CountUnreadNotifications(ctx, userId, lastReadNotification.LastReadID)
	if err != nil {
		if !ent.IsNotFound(err) {
			return nil, v1.ErrorDatabaseQuery("can't count common type notifications")
		}
		return nil, v1.ErrorNotificationNotFound("notifications not found")
	}

	if len(counters) == 0 {
		return nil, v1.ErrorNotificationNotFound("notifications not found")
	}

	replyCounter := make(map[string]int32)
	var totalUnread int32
	for _, counter := range counters {
		replyCounter[counter.Type] = int32(counter.Count)
		totalUnread += int32(counter.Count)
	}

	return (*NotificationsCounters)(&replyCounter), nil
}

func (uc *NotificationsUsecase) ListNotifications(ctx context.Context, filter *data.FilterNotificationsDto, paginate *v1.PaginateRequest) (*NotificationsList, error) {
	userId, ok := uc.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, v1.ErrorUnauthorized("Unauthorized")
	}
	filter.UserId = userId

	var notificationType string

	if enum.NotificationType(filter.Type).IsValid() {
		notificationType = filter.Type
	}
	notifications, err := uc.notificationsRepo.ListNotifications(ctx, filter, paginate)
	if err != nil {
		return nil, err
	}

	notificationItems := uc.createNotifications(notifications)

	total, err := uc.notificationsRepo.CountNotifications(ctx, userId, notificationType)
	if err != nil {
		if !ent.IsNotFound(err) {
			return nil, v1.ErrorDatabaseQuery("can't get notification count")
		}
		return nil, v1.ErrorNotificationNotFound("this user has no notifications")
	}

	paginateReply := &v1.PaginateReply{Total: &total}

	if paginate.FromId == 0 && len(notifications) == int(paginate.Limit) {
		paginateReply.ToId = &notifications[0].ID
		if paginate.Descending {
			paginateReply.ToId = &notifications[len(notifications)-1].ID
		}
	}

	if paginate.ToId == 0 || paginate.FromId != 0 {
		paginateReply.FromId = &notifications[len(notifications)-1].ID
		if paginate.Descending {
			paginateReply.FromId = &notifications[0].ID
		}
	}

	return &NotificationsList{
		Notifications: notificationItems,
		Paginate:      paginateReply,
	}, nil
}

func (uc *NotificationsUsecase) createNotifications(notifications []*ent.Notification) []*NotificationItem {
	notificationsItems := make([]*NotificationItem, len(notifications))
	for i, notification := range notifications {
		notificationsItems[i] = &NotificationItem{}
		notificationsItems[i].Notification = notification
	}

	return notificationsItems
}
