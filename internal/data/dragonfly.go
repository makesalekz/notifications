package data

import (
	"context"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"

	"gitlab.calendaria.team/services/notifications/ent/enum"
	"gitlab.calendaria.team/services/notifications/internal/conf"
)

// DragonflyClient .
type DragonflyClient interface {
	GetBadges(ctx context.Context, userID int64) (map[enum.NotificationType]int64, error)
	IncrementBadge(ctx context.Context, userID int64, badgeType enum.NotificationType) error
}

type dragonflyClient struct {
	client *redis.Client
	log    *log.Helper
}

func NewDragonflyClient(conf *conf.Bootstrap, logger log.Logger) (DragonflyClient, func(), error) {
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

	return &dragonflyClient{
		client: client,
		log:    l,
	}, cleanup, nil
}

func (c *dragonflyClient) GetBadges(ctx context.Context, userID int64) (map[enum.NotificationType]int64, error) {
	key := "badges:" + strconv.FormatInt(userID, 10)

	badges := make(map[enum.NotificationType]int64)

	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return badges, err
	}

	if exists == 0 {
		fields := make(map[string]interface{})
		for _, badgeType := range []enum.NotificationType{
			enum.Common,
			enum.Event,
			enum.Contact,
			enum.Tasks,
			enum.Projects,
		} {
			fields[badgeType.Value()] = 0
		}

		err = c.client.HSet(ctx, key, fields).Err()
		if err != nil {
			return badges, err
		}

		// todo: increment or decrement expiration time
		err = c.client.Expire(ctx, key, 5*time.Minute).Err()
		if err != nil {
			return badges, err
		}
		return badges, nil
	}

	result, err := c.client.HGetAll(ctx, key).Result()
	if err != nil {
		return badges, err
	}

	for k, v := range result {
		badgeType := enum.NotificationType(k)
		if !badgeType.IsValid() {
			c.log.Warnf("invalid badge type: %s", k)
			continue
		}
		count, parseErr := strconv.ParseInt(v, 10, 64)
		if parseErr != nil {
			c.log.Warnf("failed to parse badge count for %s: %v", k, err)
			badges[badgeType] = 0
			continue
		}
		badges[badgeType] = count
	}

	return badges, nil
}

func (c *dragonflyClient) IncrementBadge(ctx context.Context, userID int64, badgeType enum.NotificationType) error {
	if !badgeType.IsValid() {
		return nil
	}
	key := "badges:" + strconv.FormatInt(userID, 10)
	_, err := c.client.HIncrBy(ctx, key, badgeType.Value(), 1).Result()
	return err
}
