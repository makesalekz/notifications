// Package events_v1 provides stub types for the events service API.
// The events service is not available as a GitHub dependency.
package events_v1

import (
	"context"

	"google.golang.org/grpc"
)

type LocationDto struct {
	Address     *string `json:"address,omitempty"`
	Information *string `json:"information,omitempty"`
	Latitude    float32 `json:"latitude,omitempty"`
	Longitude   float32 `json:"longitude,omitempty"`
}

type Event struct {
	Id             int64        `json:"id,omitempty"`
	Title          string       `json:"title,omitempty"`
	Description    string       `json:"description,omitempty"`
	CoverUrl       *string      `json:"coverUrl,omitempty"`
	StartDateTime  string       `json:"startDateTime,omitempty"`
	EndDateTime    string       `json:"endDateTime,omitempty"`
	IsAllDay       bool         `json:"isAllDay,omitempty"`
	NoticeBefore   int32        `json:"noticeBefore,omitempty"`
	ChatId         *int64       `json:"chatId,omitempty"`
	Type           string       `json:"type,omitempty"`
	RecurrenceRule *string      `json:"recurrenceRule,omitempty"`
	Avatars        []string     `json:"avatars,omitempty"`
	MembersCount   int32        `json:"membersCount,omitempty"`
	OwnerId        int64        `json:"ownerId,omitempty"`
	PublishedAt    string       `json:"publishedAt,omitempty"`
	Location       *LocationDto `json:"location,omitempty"`
}

func (x *Event) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Event) GetTitle() string {
	if x != nil {
		return x.Title
	}
	return ""
}

// Stub gRPC client types

type EventsCountRequest struct {
	UserIds        []int64
	MemberStatuses []string
	IsPublished    bool
}

type EventsCountReply struct {
	EventsCount map[int64]int32
}

func (r *EventsCountReply) GetEventsCount() map[int64]int32 {
	if r != nil {
		return r.EventsCount
	}
	return nil
}

type EventsClient interface {
	GetEventsCount(ctx context.Context, in *EventsCountRequest, opts ...grpc.CallOption) (*EventsCountReply, error)
}

func NewEventsClient(_ grpc.ClientConnInterface) EventsClient {
	return &stubEventsClient{}
}

type stubEventsClient struct{}

func (s *stubEventsClient) GetEventsCount(_ context.Context, _ *EventsCountRequest, _ ...grpc.CallOption) (*EventsCountReply, error) {
	return &EventsCountReply{}, nil
}
