package biz

import (
	"github.com/google/wire"
	"gitlab.calendaria.team/services/utils/v1/nats"
)

const QueueFCM = "fcm"

var DefaultLanguage = "en"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewSmsUsecase,
	NewFcmUsecase,
	NewNotificationsUsecase,
	nats.NewQueueManager,
)
