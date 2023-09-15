package data

import (
	"context"
	"notifications/internal/conf"
	"time"

	"github.com/go-kratos/kratos/contrib/config/consul/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/vault-client-go"
)

type Config struct {
	config.Config

	Bootstrap *conf.Bootstrap
	Vault     *vault.Client
}

func NewConfig(consulClient *api.Client, bootstrap *conf.Bootstrap) (*Config, error) {
	globalSource, err := consul.New(consulClient, consul.WithPath("app/global/"))
	if err != nil {
		return nil, err
	}
	source, err := consul.New(consulClient, consul.WithPath(bootstrap.Consul.Path))
	if err != nil {
		return nil, err
	}
	cfg := config.New(config.WithSource(globalSource, source))
	if err := cfg.Load(); err != nil {
		return nil, err
	}

	// prepare a client with the given base address
	client, err := vault.New(
		vault.WithAddress(bootstrap.Vault.Address),
		vault.WithRequestTimeout(30*time.Second),
	)
	if err != nil {
		return nil, err
	}

	// authenticate with a token
	if err := client.SetToken(bootstrap.Vault.Token); err != nil {
		return nil, err
	}

	return &Config{
		Config:    cfg,
		Bootstrap: bootstrap,
		Vault:     client,
	}, nil
}

func (c *Config) ReadSecretsFor(ctx context.Context, subpath string) (map[string]interface{}, error) {
	response, err := c.Vault.Secrets.KvV2Read(ctx, c.Bootstrap.Consul.Path+"/"+subpath, vault.WithMountPath("secret"))
	if err != nil {
		return nil, err
	}

	return response.Data.Data, nil
}
