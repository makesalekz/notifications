package biz

import (
	"github.com/google/wire"
	"gitlab.calendaria.team/services/utils/v1/nats"
)

const QueueFCM = "fcm"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewSmsUsecase,
	NewFcmUsecase,
	NewNotificationsUsecase,
	nats.NewQueueManager,
)
