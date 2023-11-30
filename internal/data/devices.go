package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/ent/device"
)

// DevicesRepo
type DevicesRepo interface {
	CreateDevice(ctx context.Context, userId int64, token string) error
	DeleteDevice(ctx context.Context, userId int64, token string) (int, error)
	GetDevicesForUser(ctx context.Context, userId int64) ([]*ent.Device, error)
	GetDevicesForUsers(ctx context.Context, usersIds []int64) ([]*ent.Device, error)
}

type devicesRepo struct {
	db *ent.Client
}

// NewUsersRepo .
func NewDevicesRepo(d *Data, logger log.Logger) DevicesRepo {
	return &devicesRepo{
		db: d.db,
	}
}

func (r *devicesRepo) CreateDevice(ctx context.Context, userId int64, token string) error {
	return r.db.Device.Create().
		SetUserID(userId).
		SetToken(token).
		OnConflictColumns(device.FieldToken).
		UpdateNewValues().
		Exec(ctx)
}

func (r *devicesRepo) DeleteDevice(ctx context.Context, userId int64, token string) (int, error) {
	return r.db.Device.Delete().Where(device.And(device.UserID(userId), device.Token(token))).Exec(ctx)
}

func (r *devicesRepo) GetDevicesForUser(ctx context.Context, userId int64) ([]*ent.Device, error) {
	return r.db.Device.Query().Select(device.FieldToken).Where(device.UserID(userId)).All(ctx)
}

func (r *devicesRepo) GetDevicesForUsers(ctx context.Context, usersIds []int64) ([]*ent.Device, error) {
	return r.db.Device.Query().Select(device.FieldToken).Where(device.UserIDIn(usersIds...)).All(ctx)
}
