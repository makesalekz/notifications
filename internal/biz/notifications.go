package biz

import (
	"context"

	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/ent/enum"
	"gitlab.calendaria.team/services/notifications/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
	"gitlab.calendaria.team/services/utils/v1/jwt"
)

type NotificationsList struct {
	Notifications []*ent.Notification
	Paginate      *utils_v1.PaginateReply
}

type NotificationsCounters map[string]int32

// NotificationsUsecase is a Greeter usecase.
type NotificationsUsecase struct {
	jwt               *jwt.JwtProcessor
	notificationsRepo data.NotificationsRepo
}

// NewGreeterUsecase new a Greeter usecase.
func NewNotificationsUsecase(
	jwt *jwt.JwtProcessor,
	notificationsRepo data.NotificationsRepo,
) (*NotificationsUsecase, error) {
	return &NotificationsUsecase{
		jwt:               jwt,
		notificationsRepo: notificationsRepo,
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

func (uc *NotificationsUsecase) ReadNotification(ctx context.Context, userId, notificationId int64, notificationType string) error {
	err := uc.notificationsRepo.ReadNotification(
		ctx,
		data.ReadNotificationDto{
			UserId:         userId,
			NotificationId: notificationId,
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

func (uc *NotificationsUsecase) GetNotificationCounters(ctx context.Context, userId int64) (*NotificationsCounters, error) {
	counters, err := uc.notificationsRepo.CountUnreadNotifications(ctx, userId)
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
	language *string,
	filter *data.FilterNotificationsDto,
	paginate *utils_v1.PaginateRequest,
) (*NotificationsList, error) {
	if language == nil || *language == "" {
		language = &DefaultLanguage
	}

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

	total, err := uc.notificationsRepo.CountNotifications(ctx, filter.UserId, notificationType)
	if err != nil {
		if !ent.IsNotFound(err) {
			return nil, v1.ErrorDatabaseQuery("can't get notification count")
		}
		return nil, v1.ErrorNotificationNotFound("this user has no notifications")
	}

	// set paginateReply
	var fromId, toId *int64
	if len(notifications) > 0 {
		fromId = &notifications[len(notifications)-1].ID
		toId = &notifications[0].ID
	}

	paginateReply := replyPaginate(paginate, len(notifications), total, fromId, toId)

	return &NotificationsList{
		Notifications: notifications,
		Paginate:      paginateReply,
	}, nil
}

func toDtos(createDtos []*v1.NotificationDto) []*data.NotificationDto {
	dtos := make([]*data.NotificationDto, len(createDtos))
	for i, dto := range createDtos {
		dtos[i] = &data.NotificationDto{
			UserId:    dto.UserId,
			Title:     dto.Title,
			Text:      dto.Text,
			EventId:   dto.EventId,
			ContactId: dto.ContactId,
			TaskId:    dto.TaskId,
		}
	}

	return dtos
}
