// Package chats_v1 provides stub types for the chats service API.
// The chats service is not available as a GitHub dependency.
package chats_v1

import (
	"context"

	"google.golang.org/grpc"
)

type Chat struct {
	Id          int64   `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Type        *string `protobuf:"bytes,2,opt,name=type,proto3,oneof" json:"type,omitempty"`
	Title       *string `protobuf:"bytes,3,opt,name=title,proto3,oneof" json:"title,omitempty"`
	Description *string `protobuf:"bytes,4,opt,name=description,proto3,oneof" json:"description,omitempty"`
	Cover       *string `protobuf:"bytes,5,opt,name=cover,proto3,oneof" json:"cover,omitempty"`
	EventId     *int64  `protobuf:"varint,6,opt,name=eventId,proto3,oneof" json:"eventId,omitempty"`
}

func (c *Chat) GetId() int64 {
	if c != nil {
		return c.Id
	}
	return 0
}

func (c *Chat) GetType() string {
	if c != nil && c.Type != nil {
		return *c.Type
	}
	return ""
}

func (c *Chat) GetTitle() string {
	if c != nil && c.Title != nil {
		return *c.Title
	}
	return ""
}

func (c *Chat) GetDescription() string {
	if c != nil && c.Description != nil {
		return *c.Description
	}
	return ""
}

func (c *Chat) GetCover() string {
	if c != nil && c.Cover != nil {
		return *c.Cover
	}
	return ""
}

func (c *Chat) GetEventId() int64 {
	if c != nil && c.EventId != nil {
		return *c.EventId
	}
	return 0
}

type CountUnreadedRequest struct {
	UserIds []int64 `protobuf:"varint,1,rep,packed,name=user_ids,json=userIds,proto3" json:"user_ids,omitempty"`
}

type CountUnreadedReply struct {
	UnreadMessages map[int64]int32 `protobuf:"bytes,1,rep,name=unread_messages,json=unreadMessages,proto3" json:"unread_messages,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

func (r *CountUnreadedReply) GetUnreadMessages() map[int64]int32 {
	if r != nil {
		return r.UnreadMessages
	}
	return nil
}

type MessagesClient interface {
	CountUnreadMessages(ctx context.Context, in *CountUnreadedRequest, opts ...grpc.CallOption) (*CountUnreadedReply, error)
}

func NewMessagesClient(_ grpc.ClientConnInterface) MessagesClient {
	return &stubMessagesClient{}
}

type stubMessagesClient struct{}

func (s *stubMessagesClient) CountUnreadMessages(_ context.Context, _ *CountUnreadedRequest, _ ...grpc.CallOption) (*CountUnreadedReply, error) {
	return &CountUnreadedReply{}, nil
}
