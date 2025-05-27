package data

import (
	"context"

	notifications_v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/ent/lastreadnotification"
	"gitlab.calendaria.team/services/notifications/ent/notification"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
	u_struc "gitlab.calendaria.team/services/utils/v2/struc"

	"entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
)

// NotificationsRepo.
type NotificationsRepo interface {
	CreateNotifications(ctx context.Context, data []*NotificationDto) (int32, error)
	ListNotifications(
		ctx context.Context,
		filter *FilterNotificationsDto,
		paginate *utils_v1.PaginateRequest,
	) ([]*ent.Notification, error)
	CountNotifications(ctx context.Context, userID int64, notificationType string) (int32, error)
	ReadNotification(ctx context.Context, readDto ReadNotificationDto) error
	CountUnreadNotifications(ctx context.Context, userID int64) ([]Counter, error)
	CountUnreadNotificationsByType(ctx context.Context, userIDs []int64, notificationType string) (
		map[int64]int32,
		error,
	)
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
		notificationCreate := tx.Notification.Create().SetUserID(dto.UserID).SetTitle(dto.Title).SetText(dto.Text)

		switch {
		case dto.EventID != 0:
			notificationCreate.SetEventID(dto.EventID).SetType(u_struc.Event)
		case dto.ContactID != 0:
			notificationCreate.SetContactID(dto.ContactID).SetType(u_struc.Contact)
		case dto.TaskID != 0:
			notificationCreate.SetTaskID(dto.TaskID).SetType(u_struc.Tasks)
		case dto.ProjectID != 0:
			notificationCreate.SetProjectID(dto.ProjectID).SetType(u_struc.Projects)
		default:
			notificationCreate.SetType(u_struc.Common)
		}

		newNotification, err2 := notificationCreate.Save(ctx)
		if err2 != nil {
			return 0, err2
		}

		query := tx.NotificationData.Create().
			SetNillableChat(dto.ChatJSON).
			SetNillableContact(dto.ContactJSON).
			SetNillableEvent(dto.EventJSON).
			SetNillableMember(dto.MemberJSON).
			SetNillableMessage(dto.MessageJSON).
			SetNillableMetadata(dto.MetadataJSON).
			SetNillablePluralCount(dto.PluralCount).
			SetNillableTask(dto.TaskJSON).
			SetNillableType(dto.Type).
			SetNotificationID(newNotification.ID).
			SetTargetUserID(dto.TargetUserID).
			SetNillableProject(dto.ProjectJSON)

		if dto.TargetUserID != 0 {
			query.SetTargetUserID(dto.TargetUserID)
		}

		notificationData, err2 := query.Save(ctx)
		if err2 != nil {
			return 0, err2
		}

		newNotification.Edges.NotificationData = notificationData

		created++
	}
	// commit transaction
	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	//nolint:gosec // convertation to int32 is safe
	return int32(created), err
}

func (r *notificationsRepo) ReadNotification(ctx context.Context, readDto ReadNotificationDto) error {
	query := r.db.LastReadNotification.Create().
		SetUserID(readDto.UserID).
		SetLastReadID(readDto.NotificationID)

	if readDto.Type != "" {
		query.SetType(u_struc.NotificationType(readDto.Type))
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
	query := r.db.Notification.Query().Where(notification.UserID(filter.UserID))

	if u_struc.NotificationType(filter.Type).IsValid() {
		query.Where(notification.Type(u_struc.NotificationType(filter.Type)))
	}

	desc := true
	if paginate.GetFromId() != 0 {
		query.Where(notification.IDGT(paginate.GetFromId()))
		desc = false
	}

	if paginate.GetToId() != 0 {
		query.Where(notification.IDLT(paginate.GetToId()))
	}

	if paginate.GetLimit() == 0 {
		paginate.Limit = 100
	}

	if desc {
		query = query.Order(ent.Desc(notification.FieldID))
	} else {
		query = query.Order(ent.Asc(notification.FieldID))
	}

	notifications, err := query.Limit(int(paginate.GetLimit())).WithNotificationData().All(ctx)
	if err != nil {
		return nil, err
	}

	if desc != paginate.GetDescending() && len(notifications) > 1 {
		reverse(notifications)
	}

	return notifications, nil
}

func (r *notificationsRepo) CountNotifications(
	ctx context.Context,
	userID int64,
	notificationType string,
) (int32, error) {
	query := r.db.Notification.Query().
		Where(notification.UserID(userID))

	if u_struc.NotificationType(notificationType).IsValid() {
		query.Where(notification.Type(u_struc.NotificationType(notificationType)))
	}

	count, err := query.Count(ctx)

	return int32(count), err
}

func (r *notificationsRepo) CountUnreadNotifications(ctx context.Context, userID int64) ([]Counter, error) {
	var counters []Counter

	err := r.db.Notification.Query().
		Where(
			func(notificationTable *sql.Selector) {
				lastReadTable := sql.Table(lastreadnotification.Table)

				notificationTable.LeftJoin(lastReadTable).
					On(
						notificationTable.C(notification.FieldUserID),
						lastReadTable.C(lastreadnotification.FieldUserID),
					).
					On(
						notificationTable.C(notification.FieldType),
						lastReadTable.C(lastreadnotification.FieldType),
					).
					Where(
						sql.Or(
							sql.ColumnsGT(
								notificationTable.C(notification.FieldID),
								lastReadTable.C(lastreadnotification.FieldLastReadID),
							),
							sql.IsNull(lastReadTable.C(lastreadnotification.FieldLastReadID)),
						),
					)
			},
			notification.UserID(userID),
		).
		GroupBy(notification.FieldType).
		Aggregate(ent.Count()).
		Scan(ctx, &counters)

	return counters, err
}

func (r *notificationsRepo) CountUnreadNotificationsByType(
	ctx context.Context,
	userIDs []int64,
	notificationType string,
) (map[int64]int32, error) {
	result := make(map[int64]int32)

	if len(userIDs) == 0 {
		return result, nil
	}

	var countResults []struct {
		UserID int64 `sql:"user_id"`
		Count  int   `sql:"count"`
	}

	err := r.db.Notification.Query().
		Where(
			func(notificationTable *sql.Selector) {
				lastReadTable := sql.Table(lastreadnotification.Table)

				notificationTable.LeftJoin(lastReadTable).
					On(
						notificationTable.C(notification.FieldUserID),
						lastReadTable.C(lastreadnotification.FieldUserID),
					).
					On(
						notificationTable.C(notification.FieldType),
						lastReadTable.C(lastreadnotification.FieldType),
					).
					Where(
						sql.Or(
							sql.ColumnsGT(
								notificationTable.C(notification.FieldID),
								lastReadTable.C(lastreadnotification.FieldLastReadID),
							),
							sql.IsNull(lastReadTable.C(lastreadnotification.FieldLastReadID)),
						),
					)
			},
			notification.UserIDIn(userIDs...),
			notification.Type(u_struc.NotificationType(notificationType)),
		).
		GroupBy(notification.FieldUserID).
		Aggregate(
			func(s *sql.Selector) string {
				return sql.As(sql.Count("*"), "count")
			},
		).
		Scan(ctx, &countResults)

	if err != nil {
		return nil, err
	}

	for _, res := range countResults {
		result[res.UserID] = int32(res.Count)
	}

	for _, userID := range userIDs {
		if _, exists := result[userID]; !exists {
			result[userID] = 0
		}
	}

	return result, nil
}

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
