//go:build integration

package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testTimeout = 60 * time.Second

func TestConsumer_ReadMessage(t *testing.T) {
	broker := startKafkaContainer(t)
	topic := "test-consumer-read"
	createTopic(t, broker, topic)

	cfg := newTestConfig(broker)

	// Send a message first
	p := newTestProducer(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	err := p.SendSync(ctx, topic, []byte("hello consumer"))
	require.NoError(t, err)

	// Read it
	c := newTestConsumer(t, cfg, topic)
	msg, err := c.ReadMessage(ctx)
	require.NoError(t, err)
	require.Equal(t, []byte("hello consumer"), msg.Value)
}

func TestConsumer_CommitMessages(t *testing.T) {
	broker := startKafkaContainer(t)
	topic := "test-consumer-commit"
	createTopic(t, broker, topic)

	cfg := newTestConfig(broker)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Send a message
	p := newTestProducer(t, cfg)
	err := p.SendSync(ctx, topic, []byte("commit-me"))
	require.NoError(t, err)

	// First consumer: read and commit
	c1 := newTestConsumer(t, cfg, topic)
	msg, err := c1.ReadMessage(ctx)
	require.NoError(t, err)
	require.Equal(t, []byte("commit-me"), msg.Value)

	err = c1.CommitMessages(ctx, msg)
	require.NoError(t, err)

	// Second consumer (same group): no more messages to consume
	// Close first consumer so partition is rebalanced
	c1.reader.Close()

	// Wait for rebalance
	time.Sleep(2 * time.Second)

	c2, cleanup2, err := NewConsumer(cfg, topic, newTestLogger())
	require.NoError(t, err)
	defer cleanup2()

	// Should get no new message within timeout
	readCtx, readCancel := context.WithTimeout(ctx, 5*time.Second)
	defer readCancel()

	_, err = c2.ReadMessage(readCtx)
	// We expect a timeout/cancelled error since no new messages
	require.Error(t, err)
}

func TestProducerConsumer_EndToEnd(t *testing.T) {
	broker := startKafkaContainer(t)
	topic := "test-e2e"
	createTopic(t, broker, topic)

	cfg := newTestConfig(broker)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Send N messages
	const n = 5
	p := newTestProducer(t, cfg)
	for i := 0; i < n; i++ {
		err := p.SendSync(ctx, topic, []byte("msg"))
		require.NoError(t, err)
	}

	// Consume all N messages
	c := newTestConsumer(t, cfg, topic)
	received := make([][]byte, 0, n)
	for i := 0; i < n; i++ {
		msg, err := c.ReadMessage(ctx)
		require.NoError(t, err)
		received = append(received, msg.Value)
		err = c.CommitMessages(ctx, msg)
		require.NoError(t, err)
	}

	require.Len(t, received, n)
	for _, v := range received {
		require.Equal(t, []byte("msg"), v)
	}
}
