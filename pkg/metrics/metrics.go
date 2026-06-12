package metrics

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http/status"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/codes"
)

const (
	MetricsPath = "/metrics"

	labelKind      = "kind"
	labelOperation = "operation"
)

var defaultLatencyBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

var (
	mu       sync.RWMutex
	registry *prometheus.Registry

	serverRequests *prometheus.CounterVec
	serverDuration *prometheus.HistogramVec
	clientRequests *prometheus.CounterVec
	clientDuration *prometheus.HistogramVec
)

// Registry returns the dedicated Prometheus registry. Returns nil if Setup
// hasn't been called.
func Registry() *prometheus.Registry {
	mu.RLock()
	defer mu.RUnlock()
	return registry
}

// Setup creates the dedicated Prometheus registry, registers Go runtime /
// process collectors, builds the server/client request meters, and initializes
// the infrastructure counters. Idempotent: repeated calls return early.
func Setup(_ context.Context, serviceName string, logger log.Logger) (func(), error) {
	mu.Lock()
	defer mu.Unlock()
	if registry != nil {
		return func() {}, nil
	}

	helper := log.NewHelper(log.With(logger, "module", "metrics"))

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	constLabels := prometheus.Labels{"service": serviceName}

	serverRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        "server_requests_code_total",
		Help:        "Total server-side requests by kind, operation, code, and reason.",
		ConstLabels: constLabels,
	}, []string{labelKind, labelOperation, "code", labelReason})
	serverDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "server_requests_seconds",
		Help:        "Server-side request duration in seconds.",
		Buckets:     defaultLatencyBuckets,
		ConstLabels: constLabels,
	}, []string{labelKind, labelOperation})
	clientRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        "client_requests_code_total",
		Help:        "Total outgoing client calls by kind, operation, code, and reason.",
		ConstLabels: constLabels,
	}, []string{labelKind, labelOperation, "code", labelReason})
	clientDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "client_requests_seconds",
		Help:        "Outgoing client call duration in seconds.",
		Buckets:     defaultLatencyBuckets,
		ConstLabels: constLabels,
	}, []string{labelKind, labelOperation})

	reg.MustRegister(serverRequests, serverDuration, clientRequests, clientDuration)

	if err := initInfraMetrics(reg, constLabels); err != nil {
		return nil, err
	}

	registry = reg
	helper.Info("prometheus registry initialized")
	return func() {}, nil
}

// Server returns a Kratos middleware that counts every server-side request
// and observes its latency.
func Server() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			mu.RLock()
			counter, hist := serverRequests, serverDuration
			mu.RUnlock()
			if counter == nil {
				return handler(ctx, req)
			}
			kind, op := operationLabels(transport.FromServerContext(ctx))
			start := time.Now()
			reply, err = handler(ctx, req)
			recordRequest(counter, hist, kind, op, err, time.Since(start))
			return reply, err
		}
	}
}

// Client mirrors Server for outgoing gRPC client calls.
func Client() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			mu.RLock()
			counter, hist := clientRequests, clientDuration
			mu.RUnlock()
			if counter == nil {
				return handler(ctx, req)
			}
			kind, op := operationLabels(transport.FromClientContext(ctx))
			start := time.Now()
			reply, err = handler(ctx, req)
			recordRequest(counter, hist, kind, op, err, time.Since(start))
			return reply, err
		}
	}
}

// Handler returns the HTTP handler serving the Prometheus scrape endpoint.
func Handler() http.Handler {
	mu.RLock()
	reg := registry
	mu.RUnlock()
	if reg == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "metrics not initialized", http.StatusServiceUnavailable)
		})
	}
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg})
}

// RegisterCollector registers a custom prometheus.Collector on the dedicated registry.
func RegisterCollector(c prometheus.Collector) {
	mu.RLock()
	reg := registry
	mu.RUnlock()
	if reg == nil {
		return
	}
	_ = reg.Register(c) //nolint:errcheck
}

func operationLabels(info transport.Transporter, ok bool) (kind, op string) {
	if !ok || info == nil {
		return "", ""
	}
	return info.Kind().String(), info.Operation()
}

func recordRequest(counter *prometheus.CounterVec, hist *prometheus.HistogramVec, kind, op string, err error, dur time.Duration) {
	code := status.FromGRPCCode(codes.OK)
	reason := ""
	if se := errors.FromError(err); se != nil {
		code = int(se.Code)
		reason = se.Reason
	}
	counter.WithLabelValues(kind, op, strconv.Itoa(code), reason).Inc()
	hist.WithLabelValues(kind, op).Observe(dur.Seconds())
}
