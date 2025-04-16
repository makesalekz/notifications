package dialer

import (
	"context"

	events_v1 "gitlab.calendaria.team/services/events/api/events/v1"
	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/internal/conf"
	u_dialer "gitlab.calendaria.team/services/utils/v4/dialer"
)

type IEventsRemote interface {
	GetEventsCount(ctx context.Context, userIDs []int64) (map[int64]int32, error)
}

type EventsRemote struct {
	dialer u_dialer.IDialer
}

func NewEventsRemote(
	conf *conf.Bootstrap,
	dm u_dialer.IDialerManager,
) (IEventsRemote, func(), error) {
	dialer, err := dm.NewServiceDialer("events", conf.GetDiscovery().GetEvents())
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		dialer.Close()
	}

	return &EventsRemote{
		dialer: dialer,
	}, cleanup, nil
}

func (r *EventsRemote) getEventsClient(ctx context.Context) (events_v1.EventsClient, error) {
	conn, err := r.dialer.Connect(ctx)
	if err != nil {
		return nil, v1.ErrorGrpcConnection("events: %s", err.Error())
	}

	return events_v1.NewEventsClient(conn), nil
}

func (r *EventsRemote) GetEventsCount(ctx context.Context, userIDs []int64) (map[int64]int32, error) {
	client, err := r.getEventsClient(ctx)
	if err != nil {
		return nil, err
	}

	eventCountReply, err := client.GetEventsCount(
		ctx, &events_v1.EventsCountRequest{
			UserIds: userIDs,
			MemberStatuses: []string{
				"WAITING",
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return eventCountReply.GetEventsCount(), nil
}
