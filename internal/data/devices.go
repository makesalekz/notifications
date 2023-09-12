package data

import (
	"context"

	"notifications/ent"
	"notifications/ent/device"

	"github.com/go-kratos/kratos/v2/log"
	_ "github.com/lib/pq"
)

// DevicesRepo
type DevicesRepo interface {
	CreateDevice(ctx context.Context, userId int64, token string) (*ent.Device, error)
	DeleteDevice(ctx context.Context, userId int64, token string) (int, error)
	GetDevicesForUser(ctx context.Context, userId int64) ([]*ent.Device, error)
}

type devicesRepo struct {
	db *ent.Client
}

// NewUsersRepo .
func NewUsersRepo(d *Data, logger log.Logger) DevicesRepo {
	return &devicesRepo{
		db: d.db,
	}
}

func (r *devicesRepo) CreateDevice(ctx context.Context, userId int64, token string) (*ent.Device, error) {
	return r.db.Device.Create().
		SetUserID(userId).
		SetToken(token).
		Save(ctx)
}

func (r *devicesRepo) DeleteDevice(ctx context.Context, userId int64, token string) (int, error) {
	return r.db.Device.Delete().Where(device.And(device.UserID(userId), device.Token(token))).Exec(ctx)
}

func (r *devicesRepo) GetDevicesForUser(ctx context.Context, userId int64) ([]*ent.Device, error) {
	return r.db.Device.Query().Select("registration_id").Where(device.UserID(userId)).All(ctx)
}
