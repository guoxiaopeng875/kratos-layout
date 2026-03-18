package kafka

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	kafkacontainer "github.com/testcontainers/testcontainers-go/modules/kafka"
)

// startKafkaContainer starts a real Kafka container and returns the broker address.
// The container is automatically terminated when the test finishes.
func startKafkaContainer(t *testing.T) string {
	t.Helper()

	ctx := context.Background()
	container, err := kafkacontainer.Run(ctx, "confluentinc/confluent-local:7.5.0")
	require.NoError(t, err, "start kafka container")

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("terminate kafka container: %v", err)
		}
	})

	brokers, err := container.Brokers(ctx)
	require.NoError(t, err, "get kafka brokers")
	require.NotEmpty(t, brokers)

	return brokers[0]
}

// createTopic creates a topic on the broker before test use.
// confluentinc/confluent-local may have auto-topic creation disabled,
// so we create topics explicitly via the Kafka admin API.
func createTopic(t *testing.T, broker, topic string) {
	t.Helper()

	conn, err := kafka.Dial("tcp", broker)
	require.NoError(t, err, "dial kafka for topic creation")
	defer conn.Close()

	ctrl, err := conn.Controller()
	require.NoError(t, err, "get kafka controller")

	ctrlConn, err := kafka.Dial("tcp", net.JoinHostPort(ctrl.Host, strconv.Itoa(ctrl.Port)))
	require.NoError(t, err, "dial kafka controller")
	defer ctrlConn.Close()

	err = ctrlConn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil && err.Error() != fmt.Sprintf("[36] Topic Already Exists: topic '%s' already exists", topic) {
		require.NoError(t, err, "create topic %s", topic)
	}
}

// newTestConfig creates a Config pointing to the given broker.
func newTestConfig(broker string) *Config {
	return &Config{
		Brokers:      []string{broker},
		GroupID:      "test-group",
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
}

// newTestLogger returns a nop logger for tests.
func newTestLogger() log.Logger {
	return log.DefaultLogger
}

// newTestProducer creates a test producer and registers cleanup.
func newTestProducer(t *testing.T, cfg *Config) *Producer {
	t.Helper()
	p, cleanup, err := NewProducer(cfg, newTestLogger())
	require.NoError(t, err)
	t.Cleanup(cleanup)
	return p
}

// newTestConsumer creates a test consumer and registers cleanup.
func newTestConsumer(t *testing.T, cfg *Config, topic string) *Consumer {
	t.Helper()
	c, cleanup, err := NewConsumer(cfg, topic, newTestLogger())
	require.NoError(t, err)
	t.Cleanup(cleanup)
	return c
}
