//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/makesalekz/notifications/internal/biz"
	"github.com/makesalekz/notifications/internal/conf"
	"github.com/makesalekz/notifications/internal/data"
	"github.com/makesalekz/notifications/internal/server"
	"github.com/makesalekz/notifications/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Bootstrap, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
