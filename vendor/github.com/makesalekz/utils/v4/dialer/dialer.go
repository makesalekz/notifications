package dialer

import (
	"context"
	"time"

	u_auth "github.com/makesalekz/utils/v4/middlewares/auth"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/golang-jwt/jwt/v5"
	ggrpc "google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	gmetadata "google.golang.org/grpc/metadata"
)

const CONNECTING_TIMEOUT = 3 * time.Second
const TRANSIENT_FAILURE_TIMEOUT = 5 * time.Second

type IDialer interface {
	SetEndpoint(endpointName, endpoint string) IDialer
	SetTimeout(timeout time.Duration) IDialer
	Connect(ctx context.Context) (*ggrpc.ClientConn, error)
	Close() error
}

// Dialer is a service dialer, implements IDialer
type Dialer struct {
	conn        *ggrpc.ClientConn
	dm          *DialerManager
	jwtAudience jwt.ClaimStrings
	endpoint    string
	timeout     time.Duration
}

func (d *Dialer) SetEndpoint(endpointName, endpoint string) IDialer {
	d.endpoint = endpoint
	d.jwtAudience = jwt.ClaimStrings{endpointName}

	return d
}

func (d *Dialer) SetTimeout(timeout time.Duration) IDialer {
	d.timeout = timeout

	return d
}

func (d *Dialer) Connect(ctx context.Context) (*ggrpc.ClientConn, error) {
	if d.conn != nil {
		s := d.conn.GetState()
		switch s {
		case connectivity.Idle:
			// we can use this connection
			return d.conn, nil
		case connectivity.Ready:
			// we can use this connection
			return d.conn, nil
		case connectivity.Connecting:
			// we should wait for connection
			waitCtx, cancelWait := context.WithTimeout(ctx, CONNECTING_TIMEOUT)
			defer cancelWait()
			if d.conn.WaitForStateChange(waitCtx, s) {
				return d.conn, nil
			}
		case connectivity.TransientFailure:
			// we should wait for connection
			waitCtx, cancelWait := context.WithTimeout(ctx, TRANSIENT_FAILURE_TIMEOUT)
			defer cancelWait()
			if d.conn.WaitForStateChange(waitCtx, s) {
				return d.conn, nil
			}
		case connectivity.Shutdown:
			// we should reconnect
		}
	}

	middlewares := []middleware.Middleware{
		u_auth.Client(d.dm.jwt, d.dm.jwtIssuer, d.jwtAudience),
		metadata.Client(),
	}

	if d.dm.tracer.IsInitialized() {
		middlewares = append(middlewares, tracing.Client())
	}

	conn, err := grpc.DialInsecure(
		ctx,
		grpc.WithEndpoint(d.endpoint),
		grpc.WithDiscovery(d.dm.discovery),
		grpc.WithTimeout(d.timeout),
		grpc.WithMiddleware(middlewares...),
		grpc.WithOptions(
			ggrpc.WithDefaultCallOptions(
				ggrpc.MaxCallRecvMsgSize(20*1024*1024),
				ggrpc.MaxCallSendMsgSize(20*1024*1024),
			),
		),
	)
	if err != nil {
		return nil, err
	}

	d.conn = conn

	return conn, nil
}

func (d *Dialer) Close() error {
	if d.conn == nil {
		return nil
	}

	return d.conn.Close()
}

func InvokeMetadata[Request any, Reply any](ctx context.Context, req *Request, caller func(ctx context.Context, in *Request, opts ...ggrpc.CallOption) (*Reply, error)) (*Reply, error) {
	md := gmetadata.MD{}

	reply, err := caller(ctx, req, ggrpc.Trailer(&md))
	if err != nil {
		e := errors.FromError(err)
		e.Metadata = make(map[string]string)
		for k, v := range md {
			e.Metadata[k] = v[0]
		}
		return nil, e
	}

	return reply, nil
}
