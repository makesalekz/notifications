package reply

import (
	"encoding/json"
	"time"

	events_v1 "github.com/makesalekz/notifications/internal/stub/events_v1"
	v1 "github.com/makesalekz/notifications/api/notifications/v1"
	"github.com/makesalekz/notifications/ent"

	"github.com/go-kratos/kratos/v2/log"
)

type Event struct {
	Id             int64     `json:"id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	StartDateTime  time.Time `json:"startDateTime"`
	EndDateTime    time.Time `json:"endDateTime"`
	NoticeBefore   int32     `json:"noticeBefore"`
	Type           string    `json:"type"`
	RecurrenceRule string    `json:"recurrenceRule"`
	Avatars        []string  `json:"avatars"`
	MembersCount   int32     `json:"membersCount"`
	OwnerId        int64     `json:"ownerId"`
	PublishedAt    time.Time `json:"publishedAt"`
	Location       struct {
		Address     string  `json:"address"`
		Information string  `json:"information"`
		Latitude    float32 `json:"latitude"`
		Longitude   float32 `json:"longitude"`
	} `json:"location"`
}

func MapEvent(event Event) *events_v1.Event {
	return &events_v1.Event{
		Id:             event.Id,
		Title:          event.Title,
		Description:    event.Description,
		StartDateTime:  event.StartDateTime.Format(time.RFC3339),
		EndDateTime:    event.EndDateTime.Format(time.RFC3339),
		NoticeBefore:   event.NoticeBefore,
		Type:           event.Type,
		RecurrenceRule: &event.RecurrenceRule,
		Avatars:        event.Avatars,
		MembersCount:   event.MembersCount,
		OwnerId:        event.OwnerId,
		PublishedAt:    event.PublishedAt.Format(time.RFC3339),
		Location: &events_v1.LocationDto{
			Address:     &event.Location.Address,
			Information: &event.Location.Information,
			Latitude:    event.Location.Latitude,
			Longitude:   event.Location.Longitude,
		},
	}
}

func MapNotification(notification *ent.Notification) *v1.Notification {
	reply := &v1.Notification{
		Id:        notification.ID,
		Type:      string(notification.Type),
		Title:     notification.Title,
		Text:      notification.Text,
		CreatedAt: notification.CreatedAt.Format(time.RFC3339),
	}
	if notification.EventID != nil {
		reply.EventId = *notification.EventID

		if notification.Edges.NotificationData != nil && notification.Edges.NotificationData.Event != nil {
			event := Event{}
			err := json.Unmarshal([]byte(*notification.Edges.NotificationData.Event), &event)
			if err != nil {
				log.Errorf("can't unmarshal notification data event: %s", err.Error())
			} else {
				reply.Event = MapEvent(event)
			}
		}
	}
	if notification.ContactID != nil {
		reply.ContactId = *notification.ContactID
	}
	if notification.TaskID != nil {
		reply.TaskId = *notification.TaskID
	}
	if notification.ProjectID != nil {
		reply.ProjectId = *notification.ProjectID
	}

	return reply
}

func MapNotifications(notifications []*ent.Notification) []*v1.Notification {
	reply := make([]*v1.Notification, len(notifications))

	for i, notification := range notifications {
		n := MapNotification(notification)
		reply[i] = n
	}

	return reply
}
