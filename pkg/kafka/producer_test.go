package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func TestProducer_SendSync(t *testing.T) {
	broker := startKafkaContainer(t)
	topic := "test-producer-sendsync"
	createTopic(t, broker, topic)

	cfg := newTestConfig(broker)
	p := newTestProducer(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := p.SendSync(ctx, topic, []byte("hello kafka"))
	require.NoError(t, err)
}

func TestProducer_WriteMessages(t *testing.T) {
	broker := startKafkaContainer(t)
	topic := "test-producer-write"
	createTopic(t, broker, topic)

	cfg := newTestConfig(broker)
	p := newTestProducer(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msgs := []kafka.Message{
		{Topic: topic, Key: []byte("k1"), Value: []byte("v1")},
		{Topic: topic, Key: []byte("k2"), Value: []byte("v2")},
		{Topic: topic, Key: []byte("k3"), Value: []byte("v3")},
	}

	err := p.WriteMessages(ctx, msgs...)
	require.NoError(t, err)
}

func TestProducer_SendSync_ContextCancelled(t *testing.T) {
	broker := startKafkaContainer(t)
	cfg := newTestConfig(broker)
	p := newTestProducer(t, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := p.SendSync(ctx, "test-cancelled", []byte("should fail"))
	require.Error(t, err)
}
