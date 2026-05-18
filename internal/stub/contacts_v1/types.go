// Package contacts_v1 provides stub types for the contacts service API.
// The contacts service is not available as a GitHub dependency.
package contacts_v1

import (
	"context"

	"google.golang.org/grpc"
)

type Contact struct {
	Id     int64  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	UserId *int64 `protobuf:"varint,2,opt,name=user_id,json=userId,proto3,oneof" json:"user_id,omitempty"`
	Label  string `protobuf:"bytes,3,opt,name=label,proto3" json:"label,omitempty"`
}

func (c *Contact) GetId() int64 {
	if c != nil {
		return c.Id
	}
	return 0
}

func (c *Contact) GetUserId() int64 {
	if c != nil && c.UserId != nil {
		return *c.UserId
	}
	return 0
}

func (c *Contact) GetLabel() string {
	if c != nil {
		return c.Label
	}
	return ""
}

type UserIdRequest struct {
	UserId int64 `protobuf:"varint,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
}

type GetBatchContactLabelsRequest struct {
	OwnerIds []int64 `protobuf:"varint,1,rep,packed,name=owner_ids,json=ownerIds,proto3" json:"owner_ids,omitempty"`
	UserIds  []int64 `protobuf:"varint,2,rep,packed,name=user_ids,json=userIds,proto3" json:"user_ids,omitempty"`
}

type ContactsReply struct {
	Contacts []*Contact `protobuf:"bytes,1,rep,name=contacts,proto3" json:"contacts,omitempty"`
}

func (r *ContactsReply) GetContacts() []*Contact {
	if r != nil {
		return r.Contacts
	}
	return nil
}

type BatchContactLabelsReply struct {
	ContactsByOwner map[int64]*Contact `protobuf:"bytes,1,rep,name=contacts_by_owner,json=contactsByOwner,proto3" json:"contacts_by_owner,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (r *BatchContactLabelsReply) GetContactsByOwner() map[int64]*Contact {
	if r != nil {
		return r.ContactsByOwner
	}
	return nil
}

type ContactsClient interface {
	GetContactsByUserId(ctx context.Context, in *UserIdRequest, opts ...grpc.CallOption) (*ContactsReply, error)
	GetBatchContactLabels(ctx context.Context, in *GetBatchContactLabelsRequest, opts ...grpc.CallOption) (*BatchContactLabelsReply, error)
}

func NewContactsClient(_ grpc.ClientConnInterface) ContactsClient {
	return &stubContactsClient{}
}

type stubContactsClient struct{}

func (s *stubContactsClient) GetContactsByUserId(_ context.Context, _ *UserIdRequest, _ ...grpc.CallOption) (*ContactsReply, error) {
	return &ContactsReply{}, nil
}

func (s *stubContactsClient) GetBatchContactLabels(_ context.Context, _ *GetBatchContactLabelsRequest, _ ...grpc.CallOption) (*BatchContactLabelsReply, error) {
	return &BatchContactLabelsReply{}, nil
}
