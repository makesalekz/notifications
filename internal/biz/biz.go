package biz

import "github.com/google/wire"

const QueueFCM = "fcm"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewSmsUsecase, NewFcmUsecase, NewNotificationsUsecase, NewQueueManager)
