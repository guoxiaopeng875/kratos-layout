package rocketmq

import (
	"context"
	"fmt"
	"os"

	rmq "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/go-kratos/kratos/v2/log"
)

func init() {
	if err := os.Setenv("mq.consoleAppender.enabled", "true"); err != nil {
		panic(err)
	}
	rmq.ResetLogger()
}

// SendReceipt contains the result of a message send operation.
type SendReceipt struct {
	MessageID     string
	TransactionID string
	Offset        int64
}

// Message represents a message to be sent to RocketMQ.
type Message struct {
	Topic string
	Body  []byte
	Keys  []string // Message keys for filtering/lookup
	Tag   string   // Message tag for filtering
}

// Producer wraps RocketMQ v5 producer for sending messages.
type Producer struct {
	client rmq.Producer
	log    *log.Helper
	cfg    *Config
}

// NewProducer creates a new RocketMQ v5 producer.
func NewProducer(cfg *Config, topics []string, logger log.Logger) (*Producer, func(), error) {
	logHelper := log.NewHelper(log.With(logger, "module", "pkg/rocketmq"))

	configureSSL(cfg.EnableSSL)

	opts := []rmq.ProducerOption{
		rmq.WithMaxAttempts(cfg.MaxAttempts),
	}

	if len(topics) > 0 {
		opts = append(opts, rmq.WithTopics(topics...))
	}

	p, err := rmq.NewProducer(cfg.ToRMQConfig(), opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("create rocketmq producer: %w", err)
	}

	if err := p.Start(); err != nil {
		return nil, nil, fmt.Errorf("start rocketmq producer: %w", err)
	}

	logHelper.Infof("rocketmq producer started, endpoint=%s", cfg.Endpoint)

	cleanup := func() {
		logHelper.Info("shutting down rocketmq producer")
		if err := p.GracefulStop(); err != nil {
			logHelper.Errorf("shutdown rocketmq producer: %v", err)
		}
	}

	return &Producer{
		client: p,
		log:    logHelper,
		cfg:    cfg,
	}, cleanup, nil
}

// SendSync sends a message synchronously and returns only error.
// For simple use cases where message ID is not needed.
func (p *Producer) SendSync(ctx context.Context, topic string, body []byte) error {
	_, err := p.SendSyncWithResult(ctx, topic, body)
	return err
}

// SendSyncWithResult sends a message synchronously and returns the send result.
// Use this when you need the message ID for tracking or correlation.
func (p *Producer) SendSyncWithResult(ctx context.Context, topic string, body []byte) (*SendReceipt, error) {
	msg := &rmq.Message{
		Topic: topic,
		Body:  body,
	}
	return p.sendMessage(ctx, msg)
}

// SendMessage sends a message with custom keys and tags.
// Keys are used for message lookup and filtering.
// Tags are used for message filtering on consumer side.
func (p *Producer) SendMessage(ctx context.Context, msg *Message) (*SendReceipt, error) {
	m := &rmq.Message{
		Topic: msg.Topic,
		Body:  msg.Body,
	}
	if len(msg.Keys) > 0 {
		m.SetKeys(msg.Keys...)
	}
	if msg.Tag != "" {
		m.SetTag(msg.Tag)
	}
	return p.sendMessage(ctx, m)
}

// sendMessage is the internal method that sends a rmq.Message.
func (p *Producer) sendMessage(ctx context.Context, msg *rmq.Message) (*SendReceipt, error) {
	receipts, err := p.client.Send(ctx, msg)
	if err != nil {
		p.log.WithContext(ctx).Errorf("send to %s failed: %v", msg.Topic, err)
		return nil, fmt.Errorf("send message: %w", err)
	}

	if len(receipts) == 0 {
		return nil, fmt.Errorf("send message: no receipt returned")
	}

	result := receipts[0]
	p.log.WithContext(ctx).Debugf("sent to %s, msgId=%s", msg.Topic, result.MessageID)

	return &SendReceipt{
		MessageID:     result.MessageID,
		TransactionID: result.TransactionId,
		Offset:        result.Offset,
	}, nil
}

// SendAsync sends a message asynchronously.
func (p *Producer) SendAsync(ctx context.Context, msg *Message, callback func(context.Context, *SendReceipt, error)) {
	m := &rmq.Message{
		Topic: msg.Topic,
		Body:  msg.Body,
	}
	if len(msg.Keys) > 0 {
		m.SetKeys(msg.Keys...)
	}
	if msg.Tag != "" {
		m.SetTag(msg.Tag)
	}

	p.client.SendAsync(ctx, m, func(ctx context.Context, receipts []*rmq.SendReceipt, err error) {
		if err != nil {
			p.log.WithContext(ctx).Errorf("send async to %s failed: %v", msg.Topic, err)
			callback(ctx, nil, err)
			return
		}
		if len(receipts) == 0 {
			callback(ctx, nil, fmt.Errorf("send async: no receipt returned"))
			return
		}
		result := receipts[0]
		callback(ctx, &SendReceipt{
			MessageID:     result.MessageID,
			TransactionID: result.TransactionId,
			Offset:        result.Offset,
		}, nil)
	})
}
