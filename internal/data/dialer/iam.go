package dialer

import (
	"context"

	users_v1 "gitlab.calendaria.team/services/iam/api/iam/v1"
	notifications_v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/internal/conf"
	u_dialer "gitlab.calendaria.team/services/utils/v4/dialer"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/exp/maps"
)

type IIamRemote interface {
	GetUsers(
		ctx context.Context,
		mapUsersIDs map[int64]struct{},
		withPrivacies bool,
	) (map[int64]*users_v1.UserShort, error)
	GetUsersSettings(ctx context.Context, userIDs []int64) (map[int64]map[string]string, error)
}

type IamRemote struct {
	log    *log.Helper
	dialer u_dialer.IDialer
}

func NewIamRemote(
	logger log.Logger,
	conf *conf.Bootstrap,
	dm u_dialer.IDialerManager,
) (IIamRemote, func(), error) {
	dialer, err := dm.NewServiceDialer("iam", conf.GetDiscovery().GetIam())
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		dialer.Close()
	}

	return &IamRemote{
		log:    log.NewHelper(log.With(logger, "module", "data/iam")),
		dialer: dialer,
	}, cleanup, nil
}

func (r *IamRemote) getUsersClient(ctx context.Context) (users_v1.UsersClient, error) {
	conn, err := r.dialer.Connect(ctx)
	if err != nil {
		return nil, notifications_v1.ErrorGrpcConnection("can't connect to iam: %s", err.Error())
	}

	return users_v1.NewUsersClient(conn), nil
}

func (r *IamRemote) getSettingsClient(ctx context.Context) (users_v1.SettingsClient, error) {
	conn, err := r.dialer.Connect(ctx)
	if err != nil {
		return nil, notifications_v1.ErrorGrpcConnection("can't connect to iam: %s", err.Error())
	}

	return users_v1.NewSettingsClient(conn), nil
}

// GetUsers returns userShorts map from iam service by mapUsersIDs.
func (r *IamRemote) GetUsers(
	ctx context.Context,
	mapUsersIDs map[int64]struct{},
	withPrivacies bool,
) (map[int64]*users_v1.UserShort, error) {
	if len(mapUsersIDs) == 0 {
		return nil, notifications_v1.ErrorInvalidRequest("empty mapUsersIDs")
	}
	usersIDs := maps.Keys(mapUsersIDs)

	client, err := r.getUsersClient(ctx)
	if err != nil {
		return nil, err
	}

	reply, err := client.GetUsers(
		ctx, &users_v1.GetUsersRequest{
			Ids:           usersIDs,
			WithPrivacies: withPrivacies,
		},
	)
	if err != nil {
		return nil, err
	}

	if len(reply.GetUsers()) != len(usersIDs) {
		r.log.Warnf(
			"iam.GetUsers: users found partially (%d/%d) %v",
			len(reply.GetUsers()),
			len(usersIDs),
			usersIDs,
		)
	}

	users := make(map[int64]*users_v1.UserShort)
	for _, user := range reply.GetUsers() {
		users[user.GetId()] = user
	}

	return users, nil
}

func (r *IamRemote) GetUsersSettings(ctx context.Context, userIDs []int64) (map[int64]map[string]string, error) {
	client, err := r.getSettingsClient(ctx)
	if err != nil {
		return nil, err
	}

	reply, err := client.GetUsersSettings(
		ctx, &users_v1.GetUsersSettingsRequest{
			UserIds: userIDs,
		},
	)
	if err != nil {
		return nil, err
	}

	usersSettings := make(map[int64]map[string]string)

	for userID, settings := range reply.GetUsersSettings() {
		usersSettings[userID] = settings.GetSettings()
	}

	return usersSettings, nil
}
