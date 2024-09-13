// nolint: gochecknoglobals, stylecheck, promlinter // no need refactor http server
package server

import (
	"gitlab.calendaria.team/services/notifications/internal/conf"
	u_metrics "gitlab.calendaria.team/services/utils/v1/middlewares/metrics"
	u_jwt "gitlab.calendaria.team/services/utils/v2/jwt"
	u_auth "gitlab.calendaria.team/services/utils/v2/middlewares/auth"

	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var _metricSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "server",
	Subsystem: "requests",
	Name:      "duration_sec",
	Help:      "server requests duratio(sec).",
	Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1},
}, []string{"kind", "operation"})

var _metricRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "server",
	Subsystem: "requests",
	Name:      "code_total",
	Help:      "The total number of processed requests",
}, []string{"kind", "operation", "code", "reason"})

var _activeRequests = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "server",
	Subsystem: "requests",
	Name:      "active_requests",
	Help:      "The total number of active requests",
}, []string{"kind", "operation"})

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	c *conf.Bootstrap,
	jwtp u_jwt.IJwtProcessor,
) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			metadata.Server(),
			u_auth.Server(jwtp),
			u_metrics.Server(
				u_metrics.WithSeconds(prom.NewHistogram(_metricSeconds)),
				u_metrics.WithRequests(prom.NewCounter(_metricRequests)),
				u_metrics.WithGauge(prom.NewGauge(_activeRequests)),
			),
		),
	}
	if c.GetServer().GetHttp().GetNetwork() != "" {
		opts = append(opts, http.Network(c.GetServer().GetHttp().GetNetwork()))
	}
	if c.GetServer().GetHttp().GetAddr() != "" {
		opts = append(opts, http.Address(c.GetServer().GetHttp().GetAddr()))
	}
	if c.GetServer().GetHttp().GetTimeout() != nil {
		opts = append(opts, http.Timeout(c.GetServer().GetHttp().GetTimeout().AsDuration()))
	}
	srv := http.NewServer(opts...)

	registerTechRoutes(srv)

	return srv
}

func registerTechRoutes(s *khttp.Server) {
	prometheus.MustRegister(_metricSeconds, _metricRequests)

	s.Handle("/metrics", promhttp.Handler())
}
