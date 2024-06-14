package server

import (
	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/internal/conf"
	"gitlab.calendaria.team/services/notifications/internal/service"
	"gitlab.calendaria.team/services/utils/v1/jwt"
	"gitlab.calendaria.team/services/utils/v1/middlewares/auth"
	metrics "gitlab.calendaria.team/services/utils/v1/middlewares/metrics"
	u_tracing "gitlab.calendaria.team/services/utils/v2/tracing"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(
	c *conf.Bootstrap,
	jwtp *jwt.JwtProcessor,
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
			auth.Server(jwtp),
			validate.Validator(),
			tracing.Server(),
			metrics.Server(
				metrics.WithSeconds(prom.NewHistogram(_metricSeconds)),
				metrics.WithRequests(prom.NewCounter(_metricRequests)),
				metrics.WithGauge(prom.NewGauge(_activeRequests)),
			),
		),
	}
	if c.Server.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Server.Grpc.Network))
	}
	if c.Server.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Server.Grpc.Addr))
	}
	if c.Server.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Server.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)

	v1.RegisterNotificationsServer(srv, notificationsService)
	v1.RegisterSenderServer(srv, senderService)

	return srv
}
