//nolint: gosec // convertation to int32 is safe
package biz

import (
	"context"

	iam_v1 "gitlab.calendaria.team/services/iam/api/iam/v1"
	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/ent/enum"
	"gitlab.calendaria.team/services/notifications/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type NotificationsList struct {
	Notifications []*ent.Notification
	Paginate      *utils_v1.PaginateReply
}

type NotificationsCounters map[string]int32

// NotificationsUsecase is a Greeter usecase.
type NotificationsUsecase struct {
	localizer         *data.Localizer
	notificationsRepo data.NotificationsRepo
	iam               data.IIamRemote
}

// NewGreeterUsecase new a Greeter usecase.
func NewNotificationsUsecase(
	localizer *data.Localizer,
	notificationsRepo data.NotificationsRepo,
	iam data.IIamRemote,
) (*NotificationsUsecase, error) {
	return &NotificationsUsecase{
		localizer:         localizer,
		notificationsRepo: notificationsRepo,
		iam:               iam,
	}, nil
}

func (uc *NotificationsUsecase) CreateNotifications(ctx context.Context, data []*v1.NotificationDto) (int32, error) {
	newRecords, err := uc.notificationsRepo.CreateNotifications(ctx, toDtos(data))
	if err != nil {
		if !ent.IsNotFound(err) {
			return 0, v1.ErrorDatabaseQuery("can't create notifactions: %v", err)
		}
	}

	return newRecords, nil
}

func (uc *NotificationsUsecase) ReadNotification(
	ctx context.Context,
	userID, notificationID int64,
	notificationType string,
) error {
	err := uc.notificationsRepo.ReadNotification(
		ctx,
		data.ReadNotificationDto{
			UserID:         userID,
			NotificationID: notificationID,
			Type:           notificationType,
		})
	if err != nil {
		if !ent.IsNotFound(err) {
			return v1.ErrorDatabaseQuery("can't read notifaction: %v", err)
		}
		return v1.ErrorNotificationNotFound("there is no such notification")
	}

	return nil
}

func (uc *NotificationsUsecase) GetNotificationCounters(
	ctx context.Context,
	userID int64,
) (*NotificationsCounters, error) {
	counters, err := uc.notificationsRepo.CountUnreadNotifications(ctx, userID)
	if err != nil {
		if !ent.IsNotFound(err) {
			return nil, v1.ErrorDatabaseQuery("can't count common type notifications: %v", err)
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

func (uc *NotificationsUsecase) ListNotifications(
	ctx context.Context,
	language string,
	filter *data.FilterNotificationsDto,
	paginate *utils_v1.PaginateRequest,
) (*NotificationsList, error) {
	var notificationType string

	if enum.NotificationType(filter.Type).IsValid() {
		notificationType = filter.Type
	}

	if paginate == nil {
		paginate = &utils_v1.PaginateRequest{}
	}

	notifications, err := uc.notificationsRepo.ListNotifications(ctx, filter, paginate)
	if err != nil {
		return nil, err
	}

	mapUsersIDs := make(map[int64]struct{})
	for _, notification := range notifications {
		if notification.Edges.NotificationData != nil && notification.Edges.NotificationData.TargetUserID != nil {
			mapUsersIDs[*notification.Edges.NotificationData.TargetUserID] = struct{}{}
		}
	}

	var mapUsers map[int64]*iam_v1.UserShort
	if len(mapUsersIDs) > 0 {
		mapUsers, err = uc.iam.GetUsers(ctx, mapUsersIDs, false)
		if err != nil {
			return nil, v1.ErrorGrpcConnection("can't get users: %v", err)
		}
	}

	for _, notification := range notifications {
		dto := data.FromEnt(notification, mapUsers)
		if dto.NotificationAddInfo.Type == nil {
			continue
		}

		localizedText, err2 := uc.localizer.GetLocalizedMessage(
			language,
			*dto.Type,
			dto.GetConvertedMap(),
			dto.PluralCount,
		)
		if err2 != nil {
			continue
		}

		notification.Text = localizedText
	}

	total, err := uc.notificationsRepo.CountNotifications(ctx, filter.UserID, notificationType)
	if err != nil {
		if !ent.IsNotFound(err) {
			return nil, v1.ErrorDatabaseQuery("can't get notification count")
		}
		return nil, v1.ErrorNotificationNotFound("this user has no notifications")
	}

	// set paginateReply
	var fromID, toID *int64
	if len(notifications) > 0 {
		fromID = &notifications[len(notifications)-1].ID
		toID = &notifications[0].ID
	}

	paginateReply := replyPaginate(paginate, len(notifications), total, fromID, toID)

	return &NotificationsList{
		Notifications: notifications,
		Paginate:      paginateReply,
	}, nil
}

func toDtos(createDtos []*v1.NotificationDto) []*data.NotificationDto {
	dtos := make([]*data.NotificationDto, len(createDtos))
	for i, dto := range createDtos {
		dtos[i] = &data.NotificationDto{
			UserID:    dto.GetUserId(),
			Title:     dto.GetTitle(),
			Text:      dto.GetText(),
			EventID:   dto.GetEventId(),
			ContactID: dto.GetContactId(),
			TaskID:    dto.GetTaskId(),
			ProjectID: dto.GetProjectId(),
		}
	}

	return dtos
}
