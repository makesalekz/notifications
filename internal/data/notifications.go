package data

import (
	"context"

	"entgo.io/ent/dialect/sql"
	notifications_v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/ent/enum"
	"gitlab.calendaria.team/services/notifications/ent/lastreadnotification"
	"gitlab.calendaria.team/services/notifications/ent/notification"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"

	_ "github.com/lib/pq"
)

// NotificationsRepo
type NotificationsRepo interface {
	CreateNotifications(ctx context.Context, data []*NotificationDto) (int32, error)
	ListNotifications(ctx context.Context, filter *FilterNotificationsDto, paginate *utils_v1.PaginateRequest) ([]*ent.Notification, error)
	CountNotifications(ctx context.Context, userId int64, notificationType string) (int32, error)
	ReadNotification(ctx context.Context, readDto ReadNotificationDto) error
	CountUnreadNotifications(ctx context.Context, userId int64) ([]Counter, error)
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

func (r *notificationsRepo) CreateNotifications(ctx context.Context, data []*NotificationDto) (int32, error) {
	// creating a transaction and rollback method
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return 0, notifications_v1.ErrorDatabaseQuery("transaction initialize failed")
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	created := 0
	for _, dto := range data {
		notificationCreate := tx.Notification.Create().SetUserID(dto.UserId).SetTitle(dto.Title).SetText(dto.Text)

		if dto.EventId != 0 {
			notificationCreate.SetEventID(dto.EventId).SetType(enum.Event)
		} else if dto.ContactId != 0 {
			notificationCreate.SetContactID(dto.ContactId).SetType(enum.Contact)
		} else if dto.TaskId != 0 {
			notificationCreate.SetTaskID(dto.TaskId).SetType(enum.Tasks)
		} else {
			notificationCreate.SetType(enum.Common)
		}

		newNotification, err := notificationCreate.Save(ctx)
		if err != nil {
			return 0, err
		}

		notificationData, err := tx.NotificationData.Create().
			SetNillableChat(dto.ChatJson).
			SetNillableContact(dto.ContactJson).
			SetNillableEvent(dto.EventJson).
			SetNillableMember(dto.MemberJson).
			SetNillableMessage(dto.MessageJson).
			SetNillableMetadata(dto.MetadataJson).
			SetNillablePluralCount(dto.PluralCount).
			SetNillableTask(dto.TaskJson).
			SetNillableType(dto.Type).
			SetNotificationID(newNotification.ID).
			Save(ctx)
		if err != nil {
			return 0, err
		}

		newNotification.Edges.NotificationData = notificationData

		created++
	}
	// commit transaction
	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return int32(created), err
}

func (r *notificationsRepo) ReadNotification(ctx context.Context, readDto ReadNotificationDto) error {
	query := r.db.LastReadNotification.Create().
		SetUserID(readDto.UserId).
		SetLastReadID(readDto.NotificationId)

	if readDto.Type != "" {
		query.SetType(enum.NotificationType(readDto.Type))
	}

	query.OnConflictColumns(lastreadnotification.FieldUserID, lastreadnotification.FieldType).
		UpdateNewValues()

	return query.Exec(ctx)
}

func (r *notificationsRepo) ListNotifications(
	ctx context.Context,
	filter *FilterNotificationsDto,
	paginate *utils_v1.PaginateRequest,
) ([]*ent.Notification, error) {
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

	notifications, err := query.Limit(int(paginate.Limit)).WithNotificationData().All(ctx)
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

func (r *notificationsRepo) CountUnreadNotifications(ctx context.Context, userId int64) ([]Counter, error) {
	var counters []Counter

	err := r.db.Notification.Query().
		Where(
			func(notificationTable *sql.Selector) {
				lastReadTable := sql.Table(lastreadnotification.Table)

				notificationTable.LeftJoin(lastReadTable).
					On(notificationTable.C(notification.FieldUserID), lastReadTable.C(lastreadnotification.FieldUserID)).
					On(notificationTable.C(notification.FieldType), lastReadTable.C(lastreadnotification.FieldType)).
					Where(
						sql.Or(
							sql.ColumnsGT(notificationTable.C(notification.FieldID), lastReadTable.C(lastreadnotification.FieldLastReadID)),
							sql.IsNull(lastReadTable.C(lastreadnotification.FieldLastReadID)),
						),
					)
			},
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
