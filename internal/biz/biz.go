//nolint:gochecknoglobals // this global variable is required for wire and project
package biz

import (
	"github.com/google/wire"

	"github.com/makesalekz/utils/v4/nats"
)

const (
	QueueFCM                = "fcm"
	QueueFCMSilent          = "fcm_silent"
	QueueEmail              = "email"
	QueueDeleteDeviceTokens = "delete_tokens"

	AccountDeletion = "ACCOUNT_DELETION"
)

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
