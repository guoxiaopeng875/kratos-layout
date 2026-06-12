# Grafana Dashboards

The reference project (`yield-service-go`) ships a full Grafana dashboard
(`yield-overview.json`) covering RED, downstream gRPC, business counters,
Outbox, Cron, DB and Redis. The template intentionally does **not** copy that
JSON — its panels reference business metrics (`yield_subscribe_*`,
`yield_settlement_*`, etc.) that don't exist here.

Build your dashboard against the infrastructure metrics this template
exposes; once your service has its own business counters, fork the reference
JSON or import it and rewire the panels.

## Metrics this template exposes

All metrics carry a `service` const label set to the process name (the
`SERVICE_NAME` env var, defaulting to `xxx-service` / `xxx-job`). Registration
happens in `pkg/metrics/{metrics.go,infra.go,redis_hook.go,db_collector.go}`.

**Server / Client RED (Kratos middleware, `pkg/metrics/metrics.go`)**
- `server_requests_code_total` (counter, labels: `kind, operation, code, reason`)
- `server_requests_seconds_bucket` (histogram, labels: `kind, operation`)
- `client_requests_code_total` (counter, labels: `kind, operation, code, reason`)
- `client_requests_seconds_bucket` (histogram, labels: `kind, operation`)

Success is `code="0"` (Kratos stringifies the gRPC status code). Errors are
`code!="0"`. HTTP and gRPC are recorded into the same series; distinguish
with the `kind` label.

**Infrastructure (`pkg/metrics/infra.go`)**
- `app_redis_commands_total` (labels: `command, result`)
- `app_redis_command_duration_seconds_bucket` (histogram, labels: `command`)
- `app_kafka_messages_sent_total` (labels: `topic, result`)
- `app_kafka_send_duration_seconds_bucket` (histogram, labels: `topic`)
- `app_cronjob_executions_total` (labels: `job, result`)
- `app_cronjob_duration_seconds_bucket` (histogram, labels: `job`)
- `app_cronjob_last_execution_timestamp` (gauge, labels: `job`) — use
  `time() - max(...)` to display time since last execution
- `app_cronjob_skipped_total` (labels: `job`) — re-entrancy guard hits

**DB connection pool (`pkg/metrics/db_collector.go`, scraped from
`sql.DB.Stats()`)**
- `app_db_open_connections` (gauge)
- `app_db_idle_connections` (gauge)
- `app_db_in_use_connections` (gauge)
- `app_db_wait_count_total` (counter)
- `app_db_wait_duration_seconds_total` (counter)

**Runtime**: `go_*` (Go collector) and `process_*` (Process collector) are
registered by `metrics.Setup`.

Pod annotations `prometheus.io/scrape=true`, `prometheus.io/port=8000` and
`prometheus.io/path=/metrics` (see `deploy/k8s/base/deployment*.yaml`) drive
auto-discovery for Prometheus operators / ARMS.

## Adding business metrics

Define them in your own package (e.g. `internal/biz/metrics.go`) and register
them on `metrics.Registry()` once `metrics.Setup` has run. Pattern:

```go
var SubscribeTotal *prometheus.CounterVec

func init() {
    SubscribeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "myapp_subscribe_total",
        Help: "Subscribe requests by result.",
    }, []string{"product_id", "result"})
    metrics.RegisterCollector(SubscribeTotal)
}
```

`metrics.RegisterCollector` is a no-op until `metrics.Setup` runs, which is
fine — call it from `init()` and it will register once Setup completes if you
re-register, or call `RegisterCollector` lazily inside a wire provider.
