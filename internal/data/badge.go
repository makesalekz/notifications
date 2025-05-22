package data

import (
	"context"

	u_struc "gitlab.calendaria.team/services/utils/v2/struc"
)

// todo: move to utils
type IBadgeClient interface {
	GetBadges(ctx context.Context, userID int64) (map[u_struc.NotificationType]int64, error)
	IncrementBadge(ctx context.Context, userID int64, badgeType u_struc.NotificationType) error
	DecrementBadge(ctx context.Context, userID int64, badgeType u_struc.NotificationType, count int32) error
	SetBadges(ctx context.Context, userID int64, badges map[u_struc.NotificationType]int64) error
}
