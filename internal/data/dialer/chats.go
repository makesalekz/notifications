package dialer

import (
	"context"

	chats_v1 "gitlab.calendaria.team/services/chats/api/chats/v1"
	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/internal/conf"
	u_dialer "gitlab.calendaria.team/services/utils/v4/dialer"
)

type IChatsRemote interface {
	CountUnreadMessages(ctx context.Context, userIDs []int64) (map[int64]int32, error)
}

type ChatsRemote struct {
	dialer u_dialer.IDialer
}

func NewChatsRemote(
	conf *conf.Bootstrap,
	dm u_dialer.IDialerManager,
) (IChatsRemote, func(), error) {
	dialer, err := dm.NewServiceDialer("chats", conf.GetDiscovery().GetChats())
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		dialer.Close()
	}

	return &ChatsRemote{
		dialer: dialer,
	}, cleanup, nil
}

func (r *ChatsRemote) getMessagesClient(ctx context.Context) (chats_v1.MessagesClient, error) {
	conn, err := r.dialer.Connect(ctx)
	if err != nil {
		return nil, v1.ErrorGrpcConnection("chats: %s", err.Error())
	}

	return chats_v1.NewMessagesClient(conn), nil
}

func (r *ChatsRemote) CountUnreadMessages(ctx context.Context, userIDs []int64) (map[int64]int32, error) {
	client, err := r.getMessagesClient(ctx)
	if err != nil {
		return nil, err
	}

	unreadReply, err := client.CountUnreadMessages(
		ctx, &chats_v1.CountUnreadedRequest{
			UserIds: userIDs,
		},
	)
	if err != nil {
		return nil, err
	}

	return unreadReply.UnreadMessages, nil
}
