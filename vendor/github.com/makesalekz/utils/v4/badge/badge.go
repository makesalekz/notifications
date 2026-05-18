package badge

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"

	u_struc "github.com/makesalekz/utils/v2/struc"
)

type IBadgeClient interface {
	GetBadges(ctx context.Context, userID int64) (map[u_struc.NotificationType]int64, error)
	IncrementBadge(ctx context.Context, userID int64, badgeType u_struc.NotificationType) error
	DecrementBadge(ctx context.Context, userID int64, badgeType u_struc.NotificationType, count int32) error
	SetBadges(ctx context.Context, userID int64, badges map[u_struc.NotificationType]int64) error
	GetUsers(ctx context.Context) ([]int64, error)
}

type redisBadgeClient struct {
	client *redis.Client
	log    *log.Helper
	ttl    time.Duration
}

func NewRedisBadgeClient(redisClient *redis.Client, logger log.Logger, ttl time.Duration) IBadgeClient {
	l := log.NewHelper(log.With(logger, "module", "redisBadgeClient"))

	if ttl == 0 {
		ttl = 8 * time.Hour
	}

	return &redisBadgeClient{
		client: redisClient,
		log:    l,
		ttl:    ttl,
	}
}

func (c *redisBadgeClient) GetBadges(ctx context.Context, userID int64) (map[u_struc.NotificationType]int64, error) {
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

func (c *redisBadgeClient) IncrementBadge(ctx context.Context, userID int64, badgeType u_struc.NotificationType) error {
	if !badgeType.IsValid() {
		return nil
	}
	key := "badges:" + strconv.FormatInt(userID, 10)
	_, err := c.client.HIncrBy(ctx, key, badgeType.Value(), 1).Result()
	if err != nil {
		return err
	}

	return nil
}

func (c *redisBadgeClient) DecrementBadge(
	ctx context.Context, userID int64, badgeType u_struc.NotificationType, count int32,
) error {
	if !badgeType.IsValid() {
		return nil
	}

	if count <= 0 {
		return nil
	}

	key := "badges:" + strconv.FormatInt(userID, 10)

	currentCount, err := c.client.HGet(ctx, key, badgeType.Value()).Int64()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}

	if currentCount > 0 {
		decrement := int64(count)
		if currentCount < decrement {
			decrement = currentCount
		}

		_, err = c.client.HIncrBy(ctx, key, badgeType.Value(), -decrement).Result()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *redisBadgeClient) SetBadges(
	ctx context.Context, userID int64, badges map[u_struc.NotificationType]int64,
) error {
	key := "badges:" + strconv.FormatInt(userID, 10)

	data := map[string]interface{}{
		u_struc.Event.Value():   0,
		u_struc.Chat.Value():    0,
		u_struc.Contact.Value(): 0,
	}

	for badgeType, count := range badges {
		if !badgeType.IsValid() {
			c.log.Warnf("invalid badge type: %s", badgeType)
			continue
		}
		data[badgeType.Value()] = count
	}

	_, err := c.client.HSet(ctx, key, data).Result()
	if err != nil {
		return err
	}

	return nil
}

func (c *redisBadgeClient) GetUsers(ctx context.Context) ([]int64, error) {
	keys, err := c.client.Keys(ctx, "badges:*").Result()
	if err != nil {
		return nil, err
	}

	users := make([]int64, 0, len(keys))
	for _, key := range keys {
		userIDStr := key[len("badges:"):]
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			c.log.Warnf("failed to parse user ID from key %s: %v", key, err)
			continue
		}
		users = append(users, userID)
	}

	return users, nil
}
