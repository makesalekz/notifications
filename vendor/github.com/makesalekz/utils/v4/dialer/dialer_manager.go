package dialer

import (
	"time"

	u_config "github.com/makesalekz/utils/v4/config"
	u_jwt "github.com/makesalekz/utils/v4/jwt"
	u_tracing "github.com/makesalekz/utils/v4/tracing"

	consul "github.com/go-kratos/consul/registry"
	"github.com/golang-jwt/jwt/v5"
)

type IDialerManager interface {
	NewServiceDialer(endpointName string, endpoint string) (IDialer, error)
}

// DialerManager is a service dialer manager
type DialerManager struct {
	discovery *consul.Registry
	tracer    u_tracing.ITracer
	jwt       u_jwt.IJwtProcessor
	jwtIssuer string
}

func NewServiceDialerManager(
	c u_config.IConfig,
	tracer u_tracing.ITracer,
	jwt u_jwt.IJwtProcessor,
) (IDialerManager, error) {
	return &DialerManager{
		discovery: c.GetRegistry(),
		tracer:    tracer,
		jwt:       jwt,
		jwtIssuer: c.GetAppName(),
	}, nil
}

func (dm *DialerManager) NewServiceDialer(
	endpointName string,
	endpoint string,
) (IDialer, error) {
	return &Dialer{
		dm:          dm,
		jwtAudience: jwt.ClaimStrings{endpointName},
		endpoint:    endpoint,
		timeout:     30 * time.Second,
	}, nil
}
