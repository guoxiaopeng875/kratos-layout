package metrics

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/require"
)

func setupTestRegistry(t *testing.T) {
	t.Helper()
	mu.Lock()
	registry = nil
	mu.Unlock()
	_, err := Setup(context.Background(), "test-service", log.DefaultLogger)
	require.NoError(t, err)
}

func gatherNames(t *testing.T) map[string]struct{} {
	t.Helper()
	mu.RLock()
	reg := registry
	mu.RUnlock()
	families, err := reg.Gather()
	require.NoError(t, err)
	names := make(map[string]struct{}, len(families))
	for _, f := range families {
		names[f.GetName()] = struct{}{}
	}
	return names
}

func TestRequestMetrics_Server(t *testing.T) {
	setupTestRegistry(t)

	require.NotNil(t, serverRequests)
	require.NotNil(t, serverDuration)

	serverRequests.WithLabelValues("HTTP", "/test", "200", "").Inc()
	serverDuration.WithLabelValues("HTTP", "/test").Observe(0.1)

	names := gatherNames(t)
	require.Contains(t, names, "server_requests_code_total")
	require.Contains(t, names, "server_requests_seconds")
}

func TestInfrastructureMetrics_Defined(t *testing.T) {
	setupTestRegistry(t)

	require.NotNil(t, RedisCommandsTotal)
	require.NotNil(t, RedisCommandDuration)
	require.NotNil(t, KafkaMessagesSentTotal)
	require.NotNil(t, KafkaSendDuration)

	RedisCommandsTotal.WithLabelValues("GET", "success").Inc()
	RedisCommandDuration.WithLabelValues("GET").Observe(0.001)
	KafkaMessagesSentTotal.WithLabelValues("demo", "success").Inc()
	KafkaSendDuration.WithLabelValues("demo").Observe(0.01)

	names := gatherNames(t)
	require.Contains(t, names, "app_redis_commands_total")
	require.Contains(t, names, "app_kafka_messages_sent_total")
}

func TestCronJobMetrics_Defined(t *testing.T) {
	setupTestRegistry(t)

	require.NotNil(t, CronJobExecutionsTotal)
	require.NotNil(t, CronJobDuration)
	require.NotNil(t, CronJobLastExecutionTimestamp)
	require.NotNil(t, CronJobSkippedTotal)

	CronJobExecutionsTotal.WithLabelValues("demo", "success").Inc()
	CronJobDuration.WithLabelValues("demo").Observe(5.0)
	CronJobLastExecutionTimestamp.WithLabelValues("demo").SetToCurrentTime()
	CronJobSkippedTotal.WithLabelValues("demo").Inc()

	names := gatherNames(t)
	require.Contains(t, names, "app_cronjob_executions_total")
	require.Contains(t, names, "app_cronjob_duration_seconds")
}

func TestHandler_Nil(t *testing.T) {
	mu.Lock()
	registry = nil
	mu.Unlock()

	h := Handler()
	require.NotNil(t, h)
}

func TestSetup_Idempotent(t *testing.T) {
	mu.Lock()
	registry = nil
	mu.Unlock()

	_, err := Setup(context.Background(), "test-service", log.DefaultLogger)
	require.NoError(t, err)

	_, err = Setup(context.Background(), "test-service", log.DefaultLogger)
	require.NoError(t, err)
}
