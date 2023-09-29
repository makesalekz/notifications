package data

import (
	"context"

	notifications_v1 "notifications/api/notifications/v1"
	"notifications/ent"
	"notifications/ent/enum"

	_ "github.com/lib/pq"
)

type FilterNotificationsDto struct {
	UserId int64
	FromId int64
	ToId   int64
	Limit  int32
}

// NotificationsRepo
type NotificationsRepo interface {
	CreateNotifications(ctx context.Context, data []*notifications_v1.NotificationDto) (int32, error)
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
