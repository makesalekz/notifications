//nolint:gochecknoglobals // this global variable is required for wire and project
package biz

import (
	"github.com/google/wire"

	"gitlab.calendaria.team/services/utils/v4/nats"
)

const QueueFCM = "fcm"
const QueueFCMSilent = "fcm_silent"
const QueueEmail = "email"
const QueueDeleteDeviceTokens = "delete_tokens"

var DefaultLanguage = "en"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewSmsUsecase,
	NewFcmUsecase,
	NewNotificationsUsecase,
	NewEmailUsecase,
	NewLocalizedEmailTemplates,
	nats.NewQueueManager,
)
