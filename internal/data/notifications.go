package data

import (
	"context"

	notifications_v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/ent/enum"
	"gitlab.calendaria.team/services/notifications/ent/notification"

	_ "github.com/lib/pq"
)

type FilterNotificationsDto struct {
	UserId    int64
	FromId    int64
	ToId      int64
	Limit     int32
	Ascending bool
}

// NotificationsRepo
type NotificationsRepo interface {
	CreateNotifications(ctx context.Context, data []*notifications_v1.NotificationDto) (int32, error)
	ListNotifications(ctx context.Context, filter *FilterNotificationsDto) ([]*ent.Notification, error)
}

type notificationsRepo struct {
	db *ent.Client
}

// NewNotificationsRepo .
func NewNotificationsRepo(d *Data) NotificationsRepo {
	return &notificationsRepo{
		db: d.db,
	}
}

func (r *notificationsRepo) CreateNotifications(ctx context.Context, data []*notifications_v1.NotificationDto) (int32, error) {
	notificationsCreate := make([]*ent.NotificationCreate, len(data))

	for i, dto := range data {
		notificationCreate := r.db.Notification.Create().SetUserID(dto.UserId).SetTitle(dto.Title).SetText(dto.Text)

		if dto.EventId != 0 {
			notificationCreate.SetEventID(dto.EventId).SetType(enum.Event)
		} else if dto.ContactId != 0 {
			notificationCreate.SetContactID(dto.ContactId).SetType(enum.Contact)
		} else {
			notificationCreate.SetType(enum.Common)
		}

		notificationsCreate[i] = notificationCreate
	}

	notifications, err := r.db.Notification.CreateBulk(notificationsCreate...).Save(ctx)

	return int32(len(notifications)), err
}

func (r *notificationsRepo) ListNotifications(ctx context.Context, filter *FilterNotificationsDto) ([]*ent.Notification, error) {
	query := r.db.Notification.Query().Where(notification.UserID(filter.UserId))

	if filter.FromId != 0 {
		query.Where(notification.IDGT(filter.FromId))
	}

	if filter.ToId != 0 {
		query.Where(notification.IDLT(filter.ToId))
	}

	if filter.Limit == 0 {
		filter.Limit = 100
	}

	if filter.Ascending {
		query = query.Order(ent.Asc(notification.FieldID))
	} else {
		query = query.Order(ent.Desc(notification.FieldID))
	}

	notifications, err := query.Limit(int(filter.Limit)).All(ctx)
	if err != nil {
		return nil, err
	}

	if filter.Ascending && len(notifications) > 1 {
		reverse(notifications)
	}

	return notifications, nil
}

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
