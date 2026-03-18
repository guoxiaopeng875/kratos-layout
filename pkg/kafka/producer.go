package kafka

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/segmentio/kafka-go"
)

// Producer wraps kafka.Writer for sending messages to Kafka.
type Producer struct {
	writer *kafka.Writer
	log    *log.Helper
}

// NewProducer creates a new Kafka producer.
// The returned cleanup function closes the writer and flushes pending messages.
func NewProducer(cfg *Config, logger log.Logger) (*Producer, func(), error) {
	logHelper := log.NewHelper(log.With(logger, "module", "pkg/kafka/producer"))

	w := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		WriteTimeout:           cfg.WriteTimeout,
		RequiredAcks:           kafka.RequireOne,
		AllowAutoTopicCreation: true,
	}

	logHelper.Infof("kafka producer created, brokers=%v", cfg.Brokers)

	cleanup := func() {
		logHelper.Info("closing kafka producer")
		if err := w.Close(); err != nil {
			logHelper.Errorf("close kafka producer: %v", err)
		}
	}

	return &Producer{writer: w, log: logHelper}, cleanup, nil
}

// SendSync sends a single message synchronously to the given topic.
func (p *Producer) SendSync(ctx context.Context, topic string, value []byte) error {
	if err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Value: value,
	}); err != nil {
		return fmt.Errorf("kafka: write message: %w", err)
	}
	p.log.WithContext(ctx).Debugf("sent to topic %s, size=%d", topic, len(value))
	return nil
}

// WriteMessages sends one or more messages with full control over Key and Headers.
// Each kafka.Message must have Topic set.
func (p *Producer) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	if err := p.writer.WriteMessages(ctx, msgs...); err != nil {
		return fmt.Errorf("kafka: write message: %w", err)
	}
	p.log.WithContext(ctx).Debugf("sent %d message(s)", len(msgs))
	return nil
}
