package data

import (
	"context"
	"os"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"

	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/internal/conf"
	"gitlab.calendaria.team/services/notifications/internal/data/dialer"
	u_badge "gitlab.calendaria.team/services/utils/v4/badge"
	u_config "gitlab.calendaria.team/services/utils/v4/config"
	u_dialer "gitlab.calendaria.team/services/utils/v4/dialer"
	u_jwtp "gitlab.calendaria.team/services/utils/v4/jwt"
	u_tracing "gitlab.calendaria.team/services/utils/v4/tracing"

	_ "github.com/lib/pq"
)

// ProviderSet is data providers.
//
//nolint:gochecknoglobals // global variables, used in wire
var ProviderSet = wire.NewSet(
	NewData,
	NewRedisClient,
	u_config.NewConfig,
	u_jwtp.NewJwtProcessor,
	u_dialer.NewServiceDialerManager,
	u_tracing.NewTracer,
	NewNatsClient,
	NewSmscClient,
	NewIamRemote,
	NewDevicesRepo,
	NewNotificationsRepo,
	NewLocalizer,
	NewFcmClient,
	dialer.NewChatsRemote,
	dialer.NewEventsRemote,
	NewBadgeClient,
)

// Data .
type Data struct {
	log   *log.Helper
	db    *ent.Client
	redis *redis.Client
	badge u_badge.IBadgeClient
}

// GetBadgeClient возвращает клиент для работы с бейджами.
func (d *Data) GetBadgeClient() u_badge.IBadgeClient {
	return d.badge
}

// NewData .
func NewData(bc *conf.Bootstrap, c u_config.IConfig, logger log.Logger, redisClient *redis.Client) (
	*Data, func(), error,
) {
	l := log.NewHelper(logger)

	dbDsn := bc.GetDb() // read from local config
	if dbDsn == "" {
		// read from vault
		secret, err := c.ReadSecretsFor(context.Background(), "db-dsn")
		if err != nil {
			l.Fatalf("db dsn not found: %v", err)
			return nil, nil, err
		}

		var ok bool

		dbDsn, ok = secret["data"].(string)
		if !ok {
			l.Fatalf("db dsn not found: %v", err)

			return nil, nil, err
		}
	}

	autoMigrate := os.Getenv("AUTOMIGRATE")
	entLogging := os.Getenv("ENT_LOGGING")
	var options []ent.Option
	if entLogging == "true" {
		options = append(options, ent.Debug(), ent.Log(l.Debug))
	}

	client, err := ent.Open("postgres", dbDsn, options...)
	if err != nil {
		l.Fatalf("failed opening connection to postgres: %v", err)
		return nil, nil, err
	}

	if autoMigrate != "" {
		if err = client.Schema.Create(context.Background()); err != nil {
			l.Errorf("failed creating schema resources: %v", err)
			return nil, nil, err
		}
	}

	l.Info("Connected to postgres")

	// Создаем клиент для работы с бейджами
	badgeClient := u_badge.NewRedisBadgeClient(redisClient, logger, 8*time.Hour)

	cleanup := func() {
		if err = client.Close(); err != nil {
			l.Error(err)
		}
	}

	return &Data{
		log:   log.NewHelper(logger),
		db:    client,
		redis: redisClient,
		badge: badgeClient,
	}, cleanup, nil
}

// NewRedisClient create new client for dragonfly.
func NewRedisClient(conf *conf.Bootstrap, logger log.Logger) (*redis.Client, func(), error) {
	l := log.NewHelper(logger)

	client := redis.NewClient(
		&redis.Options{
			Addr:     conf.GetDragonfly(),
			Password: "",
			DB:       0, // use default DB
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		l.Fatalf("failed opening connection to dragonfly: %v", err)
		return nil, nil, err
	}

	l.Info("Connected to dragonfly")

	cleanup := func() {
		if err = client.Close(); err != nil {
			l.Error(err)
		}
	}

	return client, cleanup, nil
}

// NewBadgeClient .
func NewBadgeClient(redis *redis.Client, logger log.Logger) u_badge.IBadgeClient {
	return u_badge.NewRedisBadgeClient(redis, logger, 8*time.Hour)
}
