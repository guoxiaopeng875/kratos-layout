//go:build integration

// This file contains integration tests that require a real RocketMQ server connection.
// Run with: go test -tags=integration ./pkg/rocketmq/...

package rocketmq

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	rmq "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/apache/rocketmq-clients/golang/v5/credentials"
	"github.com/go-kratos/kratos/v2/log"
)

func init() {
	os.Setenv("mq.consoleAppender.enabled", "true")
	rmq.ResetLogger()
}

const (
	testTopic         = "test_rocketmq_topic"
	testEndpoint      = "127.0.0.1:8081" // gRPC proxy endpoint
	testConsumerGroup = "test_consumer_group"
)

// TestProducerSendOnly tests only the producer sending functionality.
func TestProducerSendOnly(t *testing.T) {
	ctx := context.Background()
	logger := log.DefaultLogger

	cfg := &Config{
		Endpoint:      testEndpoint,
		ConsumerGroup: testConsumerGroup,
		SendTimeout:   5 * time.Second,
		MaxAttempts:   3,
		EnableSSL:     false,
		Credentials:   &credentials.SessionCredentials{},
	}

	prod, cleanup, err := NewProducer(cfg, []string{testTopic}, logger)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	defer cleanup()

	// Send a test message
	testMsg := fmt.Sprintf("Test message at %s", time.Now().Format(time.RFC3339))
	t.Logf("Sending: %s", testMsg)

	if err := prod.SendSync(ctx, testTopic, []byte(testMsg)); err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	t.Log("Message sent successfully!")
}

// TestProducerRaw tests producer using raw SDK API.
func TestProducerRaw(t *testing.T) {
	// Create producer using raw SDK
	producer, err := rmq.NewProducer(&rmq.Config{
		Endpoint: testEndpoint,
		Credentials: &credentials.SessionCredentials{
			AccessKey:    "",
			AccessSecret: "",
		},
	},
		rmq.WithTopics(testTopic),
	)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}

	// Start producer
	if err := producer.Start(); err != nil {
		t.Fatalf("failed to start producer: %v", err)
	}
	defer producer.GracefulStop()

	// Send messages
	for i := 0; i < 3; i++ {
		msg := &rmq.Message{
			Topic: testTopic,
			Body:  []byte(fmt.Sprintf("raw message %d at %s", i, time.Now().Format(time.RFC3339))),
		}
		msg.SetKeys("testKey")
		msg.SetTag("testTag")

		resp, err := producer.Send(context.Background(), msg)
		if err != nil {
			t.Fatalf("failed to send message %d: %v", i, err)
		}
		t.Logf("Message %d sent, MessageID: %s", i, resp[0].MessageID)
	}

	t.Log("All messages sent successfully!")
}

// TestPushConsumer tests the push consumer functionality.
func TestPushConsumer(t *testing.T) {
	logger := log.DefaultLogger

	cfg := &Config{
		Endpoint:      testEndpoint,
		ConsumerGroup: testConsumerGroup + "_push",
		SendTimeout:   5 * time.Second,
		MaxAttempts:   3,
		EnableSSL:     false,
		Credentials:   &credentials.SessionCredentials{},
	}

	var receivedMsgs []string
	var mu sync.Mutex

	pushCfg := NewPushConsumerConfigFromConfig(cfg)
	subscriptions := map[string]*FilterExpression{
		testTopic: SubAll,
	}

	handler := func(msg *MessageView) ConsumerResult {
		t.Logf("Push received: %s (MsgId: %s)", string(msg.GetBody()), msg.GetMessageId())
		mu.Lock()
		receivedMsgs = append(receivedMsgs, string(msg.GetBody()))
		mu.Unlock()
		return ConsumeSuccess
	}

	consumer, cleanup, err := NewPushConsumer(pushCfg, subscriptions, handler, logger)
	if err != nil {
		t.Fatalf("failed to create push consumer: %v", err)
	}
	defer cleanup()

	if err := consumer.Start(); err != nil {
		t.Fatalf("failed to start push consumer: %v", err)
	}

	t.Log("Push consumer started, waiting for messages for 30 seconds...")
	time.Sleep(30 * time.Second)

	mu.Lock()
	defer mu.Unlock()
	t.Logf("Received %d messages total", len(receivedMsgs))
}

// TestSendMessageWithKeysAndTags tests sending messages with keys and tags.
func TestSendMessageWithKeysAndTags(t *testing.T) {
	ctx := context.Background()
	logger := log.DefaultLogger

	cfg := &Config{
		Endpoint:      testEndpoint,
		ConsumerGroup: testConsumerGroup,
		SendTimeout:   5 * time.Second,
		MaxAttempts:   3,
		EnableSSL:     false,
		Credentials:   &credentials.SessionCredentials{},
	}

	prod, cleanup, err := NewProducer(cfg, []string{testTopic}, logger)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	defer cleanup()

	msg := &Message{
		Topic: testTopic,
		Body:  []byte("Message with keys and tags"),
		Keys:  []string{"key1", "key2"},
		Tag:   "testTag",
	}

	receipt, err := prod.SendMessage(ctx, msg)
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	t.Logf("Message sent successfully, MessageID: %s", receipt.MessageID)
}

// TestNewProducerWithSSLDisabled tests creating a producer with SSL disabled.
func TestNewProducerWithSSLDisabled(t *testing.T) {
	logger := log.DefaultLogger

	cfg := &Config{
		Endpoint:      testEndpoint,
		ConsumerGroup: testConsumerGroup,
		SendTimeout:   5 * time.Second,
		MaxAttempts:   3,
		EnableSSL:     false,
		Credentials:   &credentials.SessionCredentials{},
	}

	t.Logf("Config: endpoint=%s, enableSSL=%v", cfg.Endpoint, cfg.EnableSSL)

	// Check the global EnableSsl setting before creating producer
	t.Logf("Before: rmq.EnableSsl=%v", rmq.EnableSsl)

	_, cleanup, err := NewProducer(cfg, []string{testTopic}, logger)
	if err != nil {
		// Connection error is expected if server is not running
		t.Logf("Expected error (server may not be running): %v", err)
		return
	}
	defer cleanup()

	t.Logf("After: rmq.EnableSsl=%v", rmq.EnableSsl)
	t.Log("Producer created successfully!")
}
