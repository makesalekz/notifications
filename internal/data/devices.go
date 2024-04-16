package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/ent/device"
)

type DeviceDto struct {
	DeviceKey
	DeviceData
}

type DeviceKey struct {
	UserId int64
	Token  string
}

type DeviceData struct {
	Language *string
}

// DevicesRepo
type DevicesRepo interface {
	CreateDevice(ctx context.Context, deviceDto DeviceDto) error
	UpdateDevice(ctx context.Context, deviceEnt *ent.Device, deviceData DeviceData) (*ent.Device, error)
	GetDevice(ctx context.Context, deviceKey DeviceKey) (*ent.Device, error)
	DeleteDevice(ctx context.Context, deviceKey DeviceKey) (int, error)
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

func (r *devicesRepo) CreateDevice(ctx context.Context, deviceDto DeviceDto) error {
	query := r.db.Device.Create().
		SetUserID(deviceDto.UserId).
		SetToken(deviceDto.Token).
		OnConflictColumns(device.FieldToken).
		UpdateNewValues()

	if deviceDto.Language != nil && *deviceDto.Language != "" {
		query.SetLanguage(*deviceDto.Language)
	}

	return query.Exec(ctx)
}

func (r *devicesRepo) DeleteDevice(ctx context.Context, deviceKey DeviceKey) (int, error) {
	return r.db.Device.Delete().
		Where(
			device.UserID(deviceKey.UserId),
			device.Token(deviceKey.Token),
		).
		Exec(ctx)
}

func (r *devicesRepo) GetDevice(ctx context.Context, deviceKey DeviceKey) (*ent.Device, error) {
	return r.db.Device.Query().
		Where(
			device.UserID(deviceKey.UserId),
			device.Token(deviceKey.Token),
		).
		First(ctx)
}

func (r *devicesRepo) UpdateDevice(ctx context.Context, deviceEnt *ent.Device, deviceData DeviceData) (*ent.Device, error) {
	query := deviceEnt.Update()
	mustUpdate := false

	if deviceData.Language != nil && *deviceData.Language != "" &&
		(deviceEnt.Language == nil || *deviceEnt.Language != *deviceData.Language) {
		mustUpdate = true
		query.SetLanguage(*deviceData.Language)
	}

	if mustUpdate {
		return query.Save(ctx)
	}
	return deviceEnt, nil
}

func (r *devicesRepo) GetDevicesForUser(ctx context.Context, userId int64) ([]*ent.Device, error) {
	return r.db.Device.Query().Where(device.UserID(userId)).All(ctx)
}

func (r *devicesRepo) GetDevicesForUsers(ctx context.Context, usersIds []int64) ([]*ent.Device, error) {
	return r.db.Device.Query().Where(device.UserIDIn(usersIds...)).All(ctx)
}
