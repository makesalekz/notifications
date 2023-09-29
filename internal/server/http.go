package server

import (
	notifications_v1 "notifications/api/notifications/v1"
	send_v1 "notifications/api/send/v1"
	"notifications/internal/conf"
	"notifications/internal/data"
	"notifications/internal/service"

	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	jwtv4 "github.com/golang-jwt/jwt/v4"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Bootstrap, jwtp *data.JwtProcessor, senderService *service.SenderService, notificationsService *service.NotificationsService) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			jwt.Server(func(token *jwtv4.Token) (interface{}, error) {
				return jwtp.GetSecret(), nil
			}, jwt.WithSigningMethod(jwtv4.SigningMethodHS256), jwt.WithClaims(func() jwtv4.Claims { return &jwtv4.RegisteredClaims{} })),
		),
	}
	if c.Server.Http.Network != "" {
		opts = append(opts, http.Network(c.Server.Http.Network))
	}
	if c.Server.Http.Addr != "" {
		opts = append(opts, http.Address(c.Server.Http.Addr))
	}
	if c.Server.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Server.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)

	notifications_v1.RegisterNotificationsHTTPServer(srv, notificationsService)
	send_v1.RegisterSenderHTTPServer(srv, senderService)

	return srv
}
