package data

import (
	"context"

	notifications_v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/ent/enum"
	"gitlab.calendaria.team/services/notifications/ent/lastreadnotification"
	"gitlab.calendaria.team/services/notifications/ent/notification"

	_ "github.com/lib/pq"
)

type FilterNotificationsDto struct {
	UserId int64
	Type   string
}

type ReadNotificationDto struct {
	UserId         int64
	NotificationId int64
}

type Counter struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// NotificationsRepo
type NotificationsRepo interface {
	CreateNotifications(ctx context.Context, data []*notifications_v1.NotificationDto) (int32, error)
	ListNotifications(ctx context.Context, filter *FilterNotificationsDto, paginate *v1.PaginateRequest) ([]*ent.Notification, error)
	CountNotifications(ctx context.Context, userId int64, notificationType string) (int32, error)
	ReadNotification(ctx context.Context, readDto ReadNotificationDto) error
	GetLastReadNotification(ctx context.Context, userId int64) (*ent.LastReadNotification, error)
	CountUnreadNotifications(ctx context.Context, userId, lastReadId int64) ([]Counter, error)
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

func (r *notificationsRepo) ReadNotification(ctx context.Context, readDto ReadNotificationDto) error {
	query := r.db.LastReadNotification.Create().
		SetUserID(readDto.UserId).
		SetLastReadID(readDto.NotificationId).
		OnConflictColumns(lastreadnotification.FieldUserID).
		UpdateNewValues()

	return query.Exec(ctx)
}

func (r *notificationsRepo) GetLastReadNotification(ctx context.Context, userId int64) (*ent.LastReadNotification, error) {
	return r.db.LastReadNotification.Query().
		Where(lastreadnotification.UserID(userId)).
		First(ctx)
}

func (r *notificationsRepo) ListNotifications(ctx context.Context, filter *FilterNotificationsDto, paginate *v1.PaginateRequest) ([]*ent.Notification, error) {
	query := r.db.Notification.Query().Where(notification.UserID(filter.UserId))

	if enum.NotificationType(filter.Type).IsValid() {
		query.Where(notification.Type(enum.NotificationType(filter.Type)))
	}

	desc := true
	if paginate.FromId != 0 {
		query.Where(notification.IDGT(paginate.FromId))
		desc = false
	}

	if paginate.ToId != 0 {
		query.Where(notification.IDLT(paginate.ToId))
	}

	if paginate.Limit == 0 {
		paginate.Limit = 100
	}

	if desc {
		query = query.Order(ent.Desc(notification.FieldID))
	} else {
		query = query.Order(ent.Asc(notification.FieldID))
	}

	notifications, err := query.Limit(int(paginate.Limit)).All(ctx)
	if err != nil {
		return nil, err
	}

	if desc != paginate.Descending && len(notifications) > 1 {
		reverse(notifications)
	}

	return notifications, nil
}

func (r *notificationsRepo) CountNotifications(ctx context.Context, userId int64, notificationType string) (int32, error) {
	query := r.db.Notification.Query().
		Where(notification.UserID(userId))

	if enum.NotificationType(notificationType).IsValid() {
		query.Where(notification.Type(enum.NotificationType(notificationType)))
	}

	count, err := query.Count(ctx)

	return int32(count), err
}

func (r *notificationsRepo) CountUnreadNotifications(ctx context.Context, userId, lastReadId int64) ([]Counter, error) {
	var counters []Counter

	err := r.db.Notification.Query().
		Where(
			notification.IDGT(lastReadId),
			notification.UserID(userId),
		).
		GroupBy(notification.FieldType).
		Aggregate(ent.Count()).
		Scan(ctx, &counters)

	return counters, err
}

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
