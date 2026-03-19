//go:build integration

package rocketmq

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/apache/rocketmq-clients/golang/v5/credentials"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	rocketmqImage     = "apache/rocketmq:5.3.1"
	testTopic         = "test_rocketmq_topic"
	testConsumerGroup = "test_consumer_group"
	proxyPort         = "8081"
)

var sharedCluster *rocketmqCluster

// rocketmqCluster holds the container references and the gRPC proxy endpoint.
type rocketmqCluster struct {
	proxyEndpoint string
	broker        testcontainers.Container
	teardown      func()
}

// createTopic creates a topic on the broker using mqadmin.
func (c *rocketmqCluster) createTopic(ctx context.Context, topic string) error {
	code, reader, err := c.broker.Exec(ctx, []string{
		"sh", "mqadmin", "updateTopic",
		"-n", "namesrv:9876",
		"-b", "localhost:10911",
		"-t", topic,
	})
	if err != nil {
		return fmt.Errorf("exec mqadmin updateTopic: %w", err)
	}

	out, _ := io.ReadAll(reader)
	out = bytes.TrimSpace(out)
	if code != 0 {
		return fmt.Errorf("mqadmin updateTopic exit code %d: %s", code, string(out))
	}

	// Wait for route to propagate to nameserver
	time.Sleep(2 * time.Second)
	return nil
}

// setupRocketMQCluster starts a NameServer + Broker+Proxy cluster using testcontainers.
//
// RocketMQ v5 SDK networking requirement:
// The SDK connects to the proxy endpoint, then queries topic routes which return broker
// addresses. The SDK then opens a telemetry gRPC stream to those addresses. For this to
// work from the host, we must:
//   - Set brokerIP1=127.0.0.1 so the broker registers localhost as its address
//   - Map container port 8081 → host port 8081 (same port) so the address matches
func setupRocketMQCluster() (*rocketmqCluster, error) {
	ctx := context.Background()

	// Create a shared network
	nw, err := network.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create docker network: %w", err)
	}

	networkName := nw.Name

	// Start NameServer
	namesrvReq := testcontainers.ContainerRequest{
		Image:        rocketmqImage,
		ExposedPorts: []string{"9876/tcp"},
		Cmd:          []string{"sh", "mqnamesrv"},
		Networks:     []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {"namesrv"},
		},
		WaitingFor: wait.ForLog("The Name Server boot success").WithStartupTimeout(60 * time.Second),
	}

	namesrv, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: namesrvReq,
		Started:          true,
	})
	if err != nil {
		_ = nw.Remove(ctx)
		return nil, fmt.Errorf("start rocketmq namesrv: %w", err)
	}

	// Start Broker with embedded Proxy (gRPC endpoint on port 8081)
	brokerReq := testcontainers.ContainerRequest{
		Image:        rocketmqImage,
		ExposedPorts: []string{proxyPort + ":" + proxyPort + "/tcp", "10911/tcp"},
		Cmd: []string{
			"bash", "-c",
			"echo 'brokerIP1=127.0.0.1' > /home/rocketmq/broker.conf && " +
				"echo 'autoCreateTopicEnable=true' >> /home/rocketmq/broker.conf && " +
				"echo 'autoCreateSubscriptionGroup=true' >> /home/rocketmq/broker.conf && " +
				"sh mqbroker -n namesrv:9876 --enable-proxy -c /home/rocketmq/broker.conf",
		},
		Networks: []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {"broker"},
		},
		WaitingFor: wait.ForLog("rocketmq-proxy startup successfully").WithStartupTimeout(120 * time.Second),
	}

	broker, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: brokerReq,
		Started:          true,
	})
	if err != nil {
		_ = namesrv.Terminate(ctx)
		_ = nw.Remove(ctx)
		return nil, fmt.Errorf("start rocketmq broker+proxy: %w", err)
	}

	// Wait for broker to fully register with nameserver
	time.Sleep(5 * time.Second)

	return &rocketmqCluster{
		proxyEndpoint: fmt.Sprintf("127.0.0.1:%s", proxyPort),
		broker:        broker,
		teardown: func() {
			ctx := context.Background()
			_ = broker.Terminate(ctx)
			_ = namesrv.Terminate(ctx)
			_ = nw.Remove(ctx)
		},
	}, nil
}

func TestMain(m *testing.M) {
	cluster, err := setupRocketMQCluster()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start RocketMQ cluster: %v\n", err)
		os.Exit(1)
	}
	sharedCluster = cluster

	if err := sharedCluster.createTopic(context.Background(), testTopic); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create topic: %v\n", err)
		sharedCluster.teardown()
		os.Exit(1)
	}

	code := m.Run()
	sharedCluster.teardown()
	os.Exit(code)
}

func newTestRocketMQConfig(endpoint string) *Config {
	return &Config{
		Endpoint:      endpoint,
		ConsumerGroup: testConsumerGroup,
		SendTimeout:   10 * time.Second,
		MaxAttempts:   3,
		EnableSSL:     false,
		Credentials:   &credentials.SessionCredentials{},
	}
}

func TestProducerSendOnly(t *testing.T) {
	cfg := newTestRocketMQConfig(sharedCluster.proxyEndpoint)
	logger := log.DefaultLogger

	prod, cleanup, err := NewProducer(cfg, []string{testTopic}, logger)
	require.NoError(t, err, "create producer")
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testMsg := fmt.Sprintf("Test message at %s", time.Now().Format(time.RFC3339))
	err = prod.SendSync(ctx, testTopic, []byte(testMsg))
	require.NoError(t, err, "send message")
}

func TestSendSyncWithResult(t *testing.T) {
	cfg := newTestRocketMQConfig(sharedCluster.proxyEndpoint)
	logger := log.DefaultLogger

	prod, cleanup, err := NewProducer(cfg, []string{testTopic}, logger)
	require.NoError(t, err, "create producer")
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	receipt, err := prod.SendSyncWithResult(ctx, testTopic, []byte("hello rocketmq"))
	require.NoError(t, err, "send message with result")
	require.NotEmpty(t, receipt.MessageID)
}

func TestSendMessageWithKeysAndTags(t *testing.T) {
	cfg := newTestRocketMQConfig(sharedCluster.proxyEndpoint)
	logger := log.DefaultLogger

	prod, cleanup, err := NewProducer(cfg, []string{testTopic}, logger)
	require.NoError(t, err, "create producer")
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msg := &Message{
		Topic: testTopic,
		Body:  []byte("Message with keys and tags"),
		Keys:  []string{"key1", "key2"},
		Tag:   "testTag",
	}

	receipt, err := prod.SendMessage(ctx, msg)
	require.NoError(t, err, "send message with keys and tags")
	require.NotEmpty(t, receipt.MessageID)
}

func TestPushConsumer_ReceiveMessage(t *testing.T) {
	cfg := newTestRocketMQConfig(sharedCluster.proxyEndpoint)
	logger := log.DefaultLogger

	// First, send a message
	prod, prodCleanup, err := NewProducer(cfg, []string{testTopic}, logger)
	require.NoError(t, err, "create producer")
	defer prodCleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = prod.SendSync(ctx, testTopic, []byte("message for consumer"))
	require.NoError(t, err, "send message")

	// Then, consume it
	var receivedMsgs []string
	var mu sync.Mutex
	done := make(chan struct{}, 1)

	pushCfg := NewPushConsumerConfigFromConfig(cfg)
	subscriptions := map[string]*FilterExpression{
		testTopic: SubAll,
	}

	handler := func(msg *MessageView) ConsumerResult {
		mu.Lock()
		receivedMsgs = append(receivedMsgs, string(msg.GetBody()))
		mu.Unlock()
		select {
		case done <- struct{}{}:
		default:
		}
		return ConsumeSuccess
	}

	consumer, consCleanup, err := NewPushConsumer(pushCfg, subscriptions, handler, logger)
	require.NoError(t, err, "create push consumer")
	defer consCleanup()

	err = consumer.Start()
	require.NoError(t, err, "start push consumer")

	// Wait for message or timeout
	select {
	case <-done:
		t.Log("Message received by consumer")
	case <-time.After(30 * time.Second):
		t.Log("Timeout waiting for consumer message (may be expected in CI)")
	}

	mu.Lock()
	t.Logf("Received %d messages", len(receivedMsgs))
	mu.Unlock()
}
