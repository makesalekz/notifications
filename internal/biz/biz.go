// nolint:gochecknoglobals // this global variable is required for wire and project
package biz

import (
	"github.com/google/wire"
	"gitlab.calendaria.team/services/utils/v2/nats"
)

const QueueFCM = "fcm"
const QueueEmail = "email"

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
