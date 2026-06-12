package server

import (
	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/go-kratos/kratos-layout/pkg/metrics"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewJobHTTPServer creates a lightweight Kratos HTTP server for the job
// process, exposing only infrastructure endpoints: /metrics and /health.
//
// The scrape endpoint must serve the dedicated registry created by
// metrics.Setup — promhttp.Handler() exposes the global default registry,
// which has none of the app_* instruments registered on it.
func NewJobHTTPServer(c *conf.Server, logger log.Logger) *http.Server {
	_ = logger
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	srv.Handle(metrics.MetricsPath, metrics.Handler())
	srv.HandlePrefix("/health", healthHandler())
	return srv
}
