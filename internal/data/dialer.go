package data

import (
	"context"

	consul "github.com/go-kratos/consul/registry"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	jwtv4 "github.com/golang-jwt/jwt/v4"
	contacts_v1 "gitlab.calendaria.team/services/contacts/api/contacts/v1"
	events_v1 "gitlab.calendaria.team/services/events/api/events/v1"
	"gitlab.calendaria.team/services/notifications/internal/conf"
)

type Dialer struct {
	conf      *conf.Bootstrap
	discovery *consul.Registry
	jwt       *JwtProcessor
}

func NewDialer(c *Config, jwt *JwtProcessor) (*Dialer, error) {
	return &Dialer{
		conf:      c.Bootstrap,
		discovery: c.GetRegistry(),
		jwt:       jwt,
	}, nil
}

func (d *Dialer) Contacts(ctx context.Context) (contacts_v1.ContactsClient, error) {
	conn, err := grpc.DialInsecure(
		ctx,
		grpc.WithEndpoint(d.conf.Discovery.Contacts),
		grpc.WithDiscovery(d.discovery),
		grpc.WithTimeout(d.conf.Discovery.ContactsTimeout.AsDuration()),
		grpc.WithMiddleware(
			jwt.Client(func(token *jwtv4.Token) (interface{}, error) {
				return d.jwt.GetSecret(), nil
			}, jwt.WithSigningMethod(jwtv4.SigningMethodHS256), jwt.WithClaims(func() jwtv4.Claims {
				return d.jwt.GetClaimsFromContext(ctx)
			})),
		),
	)

	if err != nil {
		return nil, err
	}

	return contacts_v1.NewContactsClient(conn), nil
}

func (d *Dialer) Events(ctx context.Context) (events_v1.EventsClient, error) {
	conn, err := grpc.DialInsecure(
		ctx,
		grpc.WithEndpoint(d.conf.Discovery.Events),
		grpc.WithDiscovery(d.discovery),
		grpc.WithTimeout(d.conf.Discovery.EventsTimeout.AsDuration()),
		grpc.WithMiddleware(
			jwt.Client(func(token *jwtv4.Token) (interface{}, error) {
				return d.jwt.GetSecret(), nil
			}, jwt.WithSigningMethod(jwtv4.SigningMethodHS256), jwt.WithClaims(func() jwtv4.Claims {
				return d.jwt.GetClaimsFromContext(ctx)
			})),
		),
	)
	if err != nil {
		return nil, err
	}
	return events_v1.NewEventsClient(conn), nil
}
