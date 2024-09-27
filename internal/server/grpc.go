package server

import (
	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/internal/conf"
	"gitlab.calendaria.team/services/notifications/internal/service"
	u_metrics "gitlab.calendaria.team/services/utils/v1/middlewares/metrics"
	u_jwt "gitlab.calendaria.team/services/utils/v2/jwt"
	u_auth "gitlab.calendaria.team/services/utils/v2/middlewares/auth"
	u_tracing "gitlab.calendaria.team/services/utils/v2/tracing"

	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(
	c *conf.Bootstrap,
	jwtp u_jwt.IJwtProcessor,
	tracer *u_tracing.Tracer,
	senderService *service.SenderService,
	notificationsService *service.NotificationsService,
) *grpc.Server {
	err := tracer.Initialize()
	if err != nil {
		panic(err)
	}

	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			metadata.Server(),
			u_auth.Server(jwtp),
			validate.Validator(),
			tracing.Server(),
			u_metrics.Server(
				u_metrics.WithSeconds(prom.NewHistogram(_metricSeconds)),
				u_metrics.WithRequests(prom.NewCounter(_metricRequests)),
				u_metrics.WithGauge(prom.NewGauge(_activeRequests)),
			),
		),
	}
	if c.GetServer().GetGrpc().GetNetwork() != "" {
		opts = append(opts, grpc.Network(c.GetServer().GetGrpc().GetNetwork()))
	}
	if c.GetServer().GetGrpc().GetAddr() != "" {
		opts = append(opts, grpc.Address(c.GetServer().GetGrpc().GetAddr()))
	}
	if c.GetServer().GetGrpc().GetTimeout() != nil {
		opts = append(opts, grpc.Timeout(c.GetServer().GetGrpc().GetTimeout().AsDuration()))
	}
	srv := grpc.NewServer(opts...)

	v1.RegisterNotificationsServer(srv, notificationsService)
	v1.RegisterSenderServer(srv, senderService)

	return srv
}
