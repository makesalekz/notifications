//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"notifications/internal/biz"
	"notifications/internal/conf"
	"notifications/internal/data"
	"notifications/internal/server"
	"notifications/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/hashicorp/consul/api"
)

// wireApp init kratos application.
func wireApp(*conf.Bootstrap, *api.Client, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
