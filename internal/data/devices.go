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
	UserID   int64
	Token    string
	OldToken string
}

type DeviceData struct {
	Language string
}

// DevicesRepo.
type DevicesRepo interface {
	CreateDevice(ctx context.Context, deviceDto DeviceDto) error
	GetDevice(ctx context.Context, deviceKey DeviceKey) (*ent.Device, error)
	DeleteDevice(ctx context.Context, deviceKey DeviceKey) (int, error)
	GetDevicesForUser(ctx context.Context, userID int64) ([]*ent.Device, error)
	GetDevicesForUsers(ctx context.Context, usersIDs []int64) ([]*ent.Device, error)
	DeleteDevicesByTokens(ctx context.Context, tokens []string) (int, error)
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
		SetUserID(deviceDto.UserID).
		SetToken(deviceDto.Token).
		OnConflictColumns(device.FieldToken).
		UpdateNewValues()

	if deviceDto.Language != "" {
		query.SetLanguage(deviceDto.Language)
	}

	return query.Exec(ctx)
}

func (r *devicesRepo) DeleteDevice(ctx context.Context, deviceKey DeviceKey) (int, error) {
	return r.db.Device.Delete().
		Where(
			device.UserID(deviceKey.UserID),
			device.Token(deviceKey.Token),
		).
		Exec(ctx)
}

func (r *devicesRepo) GetDevice(ctx context.Context, deviceKey DeviceKey) (*ent.Device, error) {
	return r.db.Device.Query().
		Where(
			device.UserID(deviceKey.UserID),
			device.Token(deviceKey.Token),
		).
		First(ctx)
}

func (r *devicesRepo) GetDevicesForUser(ctx context.Context, userID int64) ([]*ent.Device, error) {
	return r.db.Device.Query().Where(device.UserID(userID)).All(ctx)
}

func (r *devicesRepo) GetDevicesForUsers(ctx context.Context, usersIDs []int64) ([]*ent.Device, error) {
	return r.db.Device.Query().Where(device.UserIDIn(usersIDs...)).All(ctx)
}

func (r *devicesRepo) DeleteDevicesByTokens(ctx context.Context, tokens []string) (int, error) {
	return r.db.Device.Delete().
		Where(device.TokenIn(tokens...)).
		Exec(ctx)
}
