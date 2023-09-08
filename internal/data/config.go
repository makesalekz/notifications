package data

import (
	"notifications/internal/conf"

	"github.com/go-kratos/kratos/contrib/config/consul/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/hashicorp/consul/api"
)

func NewConfig(consulClient *api.Client, cfg *conf.Bootstrap) (config.Config, error) {
	globalSource, err := consul.New(consulClient, consul.WithPath("app/global/"))
	if err != nil {
		return nil, err
	}
	source, err := consul.New(consulClient, consul.WithPath(cfg.Consul.Path))
	if err != nil {
		return nil, err
	}
	c := config.New(config.WithSource(globalSource, source))
	if err := c.Load(); err != nil {
		return nil, err
	}

	return c, nil
}
