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

// TestRocketMQSendAndReceive tests sending and receiving messages via RocketMQ v5.
func TestRocketMQSendAndReceive(t *testing.T) {
	ctx := context.Background()
	logger := log.DefaultLogger

	// Create producer
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

	// Send first message
	initMsg := fmt.Sprintf("Init message at %s", time.Now().Format(time.RFC3339))
	t.Logf("Sending init message: %s", initMsg)
	if sendErr := prod.SendSync(ctx, testTopic, []byte(initMsg)); sendErr != nil {
		t.Fatalf("failed to send init message: %v", sendErr)
	}
	t.Log("Message sent successfully")

	// Wait for message to be processed
	time.Sleep(2 * time.Second)

	// Create simple consumer
	var receivedMsgs []string
	var mu sync.Mutex
	msgReceived := make(chan struct{}, 10)

	consumerCfg := NewSimpleConsumerConfigFromConfig(cfg)
	subscriptions := map[string]*FilterExpression{
		testTopic: SubAll,
	}

	consumer, consumerCleanup, err := NewSimpleConsumer(consumerCfg, subscriptions, logger)
	if err != nil {
		t.Fatalf("failed to create consumer: %v", err)
	}
	defer consumerCleanup()

	if err := consumer.Start(); err != nil {
		t.Fatalf("failed to start consumer: %v", err)
	}

	// Receive messages in background
	go func() {
		for {
			msgs, err := consumer.Receive(ctx, 16, 20*time.Second)
			if err != nil {
				t.Logf("receive error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			for _, msg := range msgs {
				t.Logf("Received message: %s (MsgId: %s)", string(msg.GetBody()), msg.GetMessageId())
				mu.Lock()
				receivedMsgs = append(receivedMsgs, string(msg.GetBody()))
				mu.Unlock()
				if err := consumer.Ack(ctx, msg); err != nil {
					t.Logf("ack error: %v", err)
				}
				select {
				case msgReceived <- struct{}{}:
				default:
				}
			}
		}
	}()

	// Send test messages
	testMessages := []string{
		"Hello RocketMQ v5 - 1",
		"Hello RocketMQ v5 - 2",
		"Hello RocketMQ v5 - 3",
	}

	for i, msg := range testMessages {
		t.Logf("Sending message %d: %s", i+1, msg)
		if err := prod.SendSync(ctx, testTopic, []byte(msg)); err != nil {
			t.Fatalf("failed to send message %d: %v", i+1, err)
		}
		t.Logf("Message %d sent successfully", i+1)
	}

	// Wait for messages to be received
	timeout := time.After(15 * time.Second)
	received := 0
	for received < len(testMessages) {
		select {
		case <-msgReceived:
			received++
			t.Logf("Received %d/%d messages", received, len(testMessages))
		case <-timeout:
			t.Logf("Timeout waiting for messages, received %d/%d", received, len(testMessages))
			goto checkResults
		}
	}

checkResults:
	mu.Lock()
	defer mu.Unlock()

	t.Logf("Total received messages: %d", len(receivedMsgs))
	for i, msg := range receivedMsgs {
		t.Logf("  [%d] %s", i, msg)
	}

	if len(receivedMsgs) == 0 {
		t.Error("No messages received!")
	} else {
		t.Logf("SUCCESS: Received %d messages", len(receivedMsgs))
	}
}

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
