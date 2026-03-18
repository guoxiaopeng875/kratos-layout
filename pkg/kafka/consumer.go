package kafka

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/segmentio/kafka-go"
)

// Consumer wraps kafka.Reader for receiving messages from a single Kafka topic.
type Consumer struct {
	reader *kafka.Reader
	log    *log.Helper
}

// NewConsumer creates a new Kafka consumer bound to a single topic.
// CommitInterval is set to 0 to disable automatic offset commits (manual commit only).
// The returned cleanup function closes the reader.
func NewConsumer(cfg *Config, topic string, logger log.Logger) (*Consumer, func(), error) {
	logHelper := log.NewHelper(log.With(logger, "module", "pkg/kafka/consumer"))

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.GroupID,
		Topic:          topic,
		MaxWait:        cfg.ReadTimeout,
		CommitInterval: 0, // disable auto-commit; caller must call CommitMessages
	})

	logHelper.Infof("kafka consumer created, brokers=%v, topic=%s, group=%s",
		cfg.Brokers, topic, cfg.GroupID)

	cleanup := func() {
		logHelper.Info("closing kafka consumer")
		if err := r.Close(); err != nil {
			logHelper.Errorf("close kafka consumer: %v", err)
		}
	}

	return &Consumer{reader: r, log: logHelper}, cleanup, nil
}

// ReadMessage blocks until a message is available or ctx is cancelled.
// Returns the raw kafka.Message; the caller must call CommitMessages after processing.
func (c *Consumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	msg, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("kafka: read message: %w", err)
	}
	c.log.WithContext(ctx).Debugf("received from topic %s, partition=%d, offset=%d",
		msg.Topic, msg.Partition, msg.Offset)
	return msg, nil
}

// CommitMessages manually commits the offsets of the provided messages.
// On failure, logs the error but does not return it (offset will reset on reconnect).
func (c *Consumer) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	if err := c.reader.CommitMessages(ctx, msgs...); err != nil {
		c.log.WithContext(ctx).Errorf("kafka: commit messages: %v", err)
		return fmt.Errorf("kafka: commit messages: %w", err)
	}
	return nil
}
