package dialer

import (
	"context"

	contacts_v1 "gitlab.calendaria.team/services/contacts/api/contacts/v1"
	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/internal/conf"
	u_dialer "gitlab.calendaria.team/services/utils/v4/dialer"
)

type IContactsRemote interface {
	GetContactsByUserID(ctx context.Context, userID int64) ([]*contacts_v1.Contact, error)
	GetBatchContactLabels(ctx context.Context, ownerIDs, userIDs []int64) (map[int64]*contacts_v1.Contact, error)
}

type ContactsRemote struct {
	dialer u_dialer.IDialer
}

func NewContactsRemote(
	conf *conf.Bootstrap,
	dm u_dialer.IDialerManager,
) (IContactsRemote, func(), error) {
	dialer, err := dm.NewServiceDialer("contacts", conf.GetDiscovery().GetContacts())
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		dialer.Close()
	}

	return &ContactsRemote{
		dialer: dialer,
	}, cleanup, nil
}

func (c *ContactsRemote) getContactsClient(ctx context.Context) (contacts_v1.ContactsClient, error) {
	conn, err := c.dialer.Connect(ctx)
	if err != nil {
		return nil, v1.ErrorGrpcConnection("chats: %s", err.Error())
	}

	return contacts_v1.NewContactsClient(conn), nil
}

func (c *ContactsRemote) GetContactsByUserID(ctx context.Context, userID int64) ([]*contacts_v1.Contact, error) {
	client, err := c.getContactsClient(ctx)
	if err != nil {
		return nil, err
	}

	contactsReply, err := client.GetContactsByUserId(
		ctx, &contacts_v1.UserIdRequest{
			UserId: userID,
		},
	)

	if err != nil {
		return nil, v1.ErrorGrpcConnection("contacts: %s", err.Error())
	}

	return contactsReply.GetContacts(), nil
}

func (c *ContactsRemote) GetBatchContactLabels(
	ctx context.Context,
	ownerIDs, userIDs []int64,
) (map[int64]*contacts_v1.Contact, error) {
	client, err := c.getContactsClient(ctx)
	if err != nil {
		return nil, err
	}

	batchReply, err := client.GetBatchContactLabels(
		ctx, &contacts_v1.GetBatchContactLabelsRequest{
			OwnerIds: ownerIDs,
			UserIds:  userIDs,
		},
	)

	if err != nil {
		return nil, v1.ErrorGrpcConnection("contacts: %s", err.Error())
	}

	return batchReply.GetContactsByOwner(), nil
}
