package rocketmq

import (
	"context"
	"fmt"
	"time"

	rmq "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/go-kratos/kratos/v2/log"
)

// ConsumerResult represents the result of message consumption.
type ConsumerResult = rmq.ConsumerResult

// ConsumerResult constants.
const (
	ConsumeSuccess = rmq.SUCCESS
	ConsumeFailure = rmq.FAILURE
)

// MessageView represents a received message.
type MessageView = rmq.MessageView

// FilterExpression represents a message filter expression.
type FilterExpression = rmq.FilterExpression

// Filter expression type constants.
var (
	// SubAll subscribes to all messages (tag = "*").
	SubAll = rmq.SUB_ALL
	// NewFilterExpression creates a new TAG filter expression.
	NewFilterExpression = rmq.NewFilterExpression
	// NewFilterExpressionWithType creates a filter expression with specified type.
	NewFilterExpressionWithType = rmq.NewFilterExpressionWithType
)

// Filter expression types.
const (
	FilterTypeSQL92 = rmq.SQL92
	FilterTypeTag   = rmq.TAG
)

// MessageHandler is the callback function for processing messages.
// Return SUCCESS to acknowledge the message, FAILURE to retry.
type MessageHandler func(msg *MessageView) ConsumerResult

// PushConsumer wraps RocketMQ v5 push consumer for receiving messages.
type PushConsumer struct {
	client rmq.PushConsumer
	log    *log.Helper
	cfg    *Config
}

// PushConsumerConfig holds configuration for push consumer.
type PushConsumerConfig struct {
	*Config
	AwaitDuration              time.Duration
	MaxCacheMessageCount       int32
	MaxCacheMessageSizeInBytes int64
	ConsumptionThreadCount     int32
}

// NewPushConsumerConfigFromConfig creates a PushConsumerConfig from base Config.
func NewPushConsumerConfigFromConfig(cfg *Config) *PushConsumerConfig {
	return &PushConsumerConfig{
		Config:                     cfg,
		AwaitDuration:              5 * time.Second,
		MaxCacheMessageCount:       1024,
		MaxCacheMessageSizeInBytes: 64 * 1024 * 1024,
		ConsumptionThreadCount:     20,
	}
}

// NewPushConsumer creates a new RocketMQ v5 push consumer.
// subscriptions maps topic to filter expression.
// handler is called for each received message.
func NewPushConsumer(
	cfg *PushConsumerConfig,
	subscriptions map[string]*FilterExpression,
	handler MessageHandler,
	logger log.Logger,
) (*PushConsumer, func(), error) {
	logHelper := log.NewHelper(log.With(logger, "module", "pkg/rocketmq/consumer"))

	if len(subscriptions) == 0 {
		return nil, nil, fmt.Errorf("subscriptions cannot be empty")
	}
	if handler == nil {
		return nil, nil, fmt.Errorf("handler cannot be nil")
	}

	configureSSL(cfg.EnableSSL)

	opts := []rmq.PushConsumerOption{
		rmq.WithPushAwaitDuration(cfg.AwaitDuration),
		rmq.WithPushSubscriptionExpressions(subscriptions),
		rmq.WithPushMessageListener(&rmq.FuncMessageListener{
			Consume: handler,
		}),
		rmq.WithPushConsumptionThreadCount(cfg.ConsumptionThreadCount),
		rmq.WithPushMaxCacheMessageCount(cfg.MaxCacheMessageCount),
		rmq.WithPushMaxCacheMessageSizeInBytes(cfg.MaxCacheMessageSizeInBytes),
	}

	c, err := rmq.NewPushConsumer(cfg.ToRMQConfig(), opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("create rocketmq push consumer: %w", err)
	}

	logHelper.Infof("rocketmq push consumer created, endpoint=%s, group=%s",
		cfg.Endpoint, cfg.ConsumerGroup)

	cleanup := func() {
		logHelper.Info("shutting down rocketmq push consumer")
		if err := c.GracefulStop(); err != nil {
			logHelper.Errorf("shutdown rocketmq push consumer: %v", err)
		}
	}

	return &PushConsumer{
		client: c,
		log:    logHelper,
		cfg:    cfg.Config,
	}, cleanup, nil
}

// Start starts the push consumer.
func (c *PushConsumer) Start() error {
	if err := c.client.Start(); err != nil {
		return fmt.Errorf("start rocketmq push consumer: %w", err)
	}
	c.log.Info("rocketmq push consumer started")
	return nil
}

// Subscribe subscribes to a topic with filter expression.
// Can be called after Start to dynamically add subscriptions.
func (c *PushConsumer) Subscribe(topic string, filter *FilterExpression) error {
	if err := c.client.Subscribe(topic, filter); err != nil {
		return fmt.Errorf("subscribe to %s: %w", topic, err)
	}
	c.log.Infof("subscribed to topic: %s", topic)
	return nil
}

// Unsubscribe unsubscribes from a topic.
func (c *PushConsumer) Unsubscribe(topic string) error {
	return c.client.Unsubscribe(topic)
}

// SimpleConsumer wraps RocketMQ v5 simple consumer for pull-based message receiving.
type SimpleConsumer struct {
	client rmq.SimpleConsumer
	log    *log.Helper
	cfg    *Config
}

// SimpleConsumerConfig holds configuration for simple consumer.
type SimpleConsumerConfig struct {
	*Config
	AwaitDuration time.Duration
}

// NewSimpleConsumerConfigFromConfig creates a SimpleConsumerConfig from base Config.
func NewSimpleConsumerConfigFromConfig(cfg *Config) *SimpleConsumerConfig {
	return &SimpleConsumerConfig{
		Config:        cfg,
		AwaitDuration: 5 * time.Second,
	}
}

// NewSimpleConsumer creates a new RocketMQ v5 simple consumer.
func NewSimpleConsumer(
	cfg *SimpleConsumerConfig,
	subscriptions map[string]*FilterExpression,
	logger log.Logger,
) (*SimpleConsumer, func(), error) {
	logHelper := log.NewHelper(log.With(logger, "module", "pkg/rocketmq/consumer"))

	configureSSL(cfg.EnableSSL)

	opts := []rmq.SimpleConsumerOption{
		rmq.WithSimpleAwaitDuration(cfg.AwaitDuration),
	}

	if len(subscriptions) > 0 {
		opts = append(opts, rmq.WithSimpleSubscriptionExpressions(subscriptions))
	}

	c, err := rmq.NewSimpleConsumer(cfg.ToRMQConfig(), opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("create rocketmq simple consumer: %w", err)
	}

	logHelper.Infof("rocketmq simple consumer created, endpoint=%s, group=%s",
		cfg.Endpoint, cfg.ConsumerGroup)

	cleanup := func() {
		logHelper.Info("shutting down rocketmq simple consumer")
		if err := c.GracefulStop(); err != nil {
			logHelper.Errorf("shutdown rocketmq simple consumer: %v", err)
		}
	}

	return &SimpleConsumer{
		client: c,
		log:    logHelper,
		cfg:    cfg.Config,
	}, cleanup, nil
}

// Start starts the simple consumer.
func (c *SimpleConsumer) Start() error {
	if err := c.client.Start(); err != nil {
		return fmt.Errorf("start rocketmq simple consumer: %w", err)
	}
	c.log.Info("rocketmq simple consumer started")
	return nil
}

// Subscribe subscribes to a topic with filter expression.
func (c *SimpleConsumer) Subscribe(topic string, filter *FilterExpression) error {
	if err := c.client.Subscribe(topic, filter); err != nil {
		return fmt.Errorf("subscribe to %s: %w", topic, err)
	}
	c.log.Infof("subscribed to topic: %s", topic)
	return nil
}

// Unsubscribe unsubscribes from a topic.
func (c *SimpleConsumer) Unsubscribe(topic string) error {
	return c.client.Unsubscribe(topic)
}

// Receive receives messages from subscribed topics.
// maxMessageNum specifies the maximum number of messages to receive.
// invisibleDuration specifies how long the message is invisible to other consumers.
func (c *SimpleConsumer) Receive(ctx context.Context, maxMessageNum int32, invisibleDuration time.Duration) ([]*MessageView, error) {
	msgs, err := c.client.Receive(ctx, maxMessageNum, invisibleDuration)
	if err != nil {
		c.log.WithContext(ctx).Errorf("receive messages failed: %v", err)
		return nil, fmt.Errorf("receive messages: %w", err)
	}
	return msgs, nil
}

// Ack acknowledges a message.
func (c *SimpleConsumer) Ack(ctx context.Context, msg *MessageView) error {
	if err := c.client.Ack(ctx, msg); err != nil {
		c.log.WithContext(ctx).Errorf("ack message %s failed: %v", msg.GetMessageId(), err)
		return fmt.Errorf("ack message: %w", err)
	}
	return nil
}

// ChangeInvisibleDuration changes the invisible duration of a message.
func (c *SimpleConsumer) ChangeInvisibleDuration(msg *MessageView, invisibleDuration time.Duration) error {
	return c.client.ChangeInvisibleDuration(msg, invisibleDuration)
}
