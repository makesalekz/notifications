package data

import (
	"context"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"

	"gitlab.calendaria.team/services/notifications/internal/conf"
	u_struc "gitlab.calendaria.team/services/utils/v2/struc"
)

// DragonflyClient .
type DragonflyClient interface {
	GetBadges(ctx context.Context, userID int64) (map[u_struc.NotificationType]int64, error)
	IncrementBadge(ctx context.Context, userID int64, badgeType u_struc.NotificationType) error
	SetBadges(ctx context.Context, userID int64, badges map[u_struc.NotificationType]int64) error
}

type dragonflyClient struct {
	client *redis.Client
	log    *log.Helper
	ttl    time.Duration
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
		ttl:    8 * time.Hour,
	}, cleanup, nil
}

func (c *dragonflyClient) GetBadges(ctx context.Context, userID int64) (map[u_struc.NotificationType]int64, error) {
	key := "badges:" + strconv.FormatInt(userID, 10)

	exists, err := c.client.Exists(ctx, key).Result()
	if exists == 0 || err != nil {
		return nil, err
	}

	badges := make(map[u_struc.NotificationType]int64)

	result, err := c.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	for k, v := range result {
		badgeType := u_struc.NotificationType(k)
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

func (c *dragonflyClient) IncrementBadge(ctx context.Context, userID int64, badgeType u_struc.NotificationType) error {
	if !badgeType.IsValid() {
		return nil
	}
	key := "badges:" + strconv.FormatInt(userID, 10)
	_, err := c.client.HIncrBy(ctx, key, badgeType.Value(), 1).Result()
	if err != nil {
		return err
	}

	_, err = c.client.Expire(ctx, key, c.ttl).Result()
	if err != nil {
		return err
	}

	return nil
}

func (c *dragonflyClient) SetBadges(
	ctx context.Context, userID int64, badges map[u_struc.NotificationType]int64,
) error {
	key := "badges:" + strconv.FormatInt(userID, 10)

	data := make(map[string]interface{})
	for badgeType, count := range badges {
		if !badgeType.IsValid() {
			c.log.Warnf("invalid badge type: %s", badgeType)
			continue
		}
		data[badgeType.Value()] = count
	}

	if len(data) == 0 {
		return nil
	}

	_, err := c.client.HSet(ctx, key, data).Result()
	if err != nil {
		return err
	}

	_, err = c.client.Expire(ctx, key, c.ttl).Result()
	return err
}
