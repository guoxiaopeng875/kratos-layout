package rocketmq

import (
	"testing"

	rmq "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/go-kratos/kratos/v2/log"
)

func TestNewPushConsumer_EmptySubscriptions(t *testing.T) {
	cfg := NewPushConsumerConfigFromConfig(&Config{
		Endpoint:      "localhost:8081",
		ConsumerGroup: "test-group",
	})

	_, _, err := NewPushConsumer(cfg, nil, func(msg *MessageView) ConsumerResult {
		return ConsumeSuccess
	}, log.DefaultLogger)
	if err == nil {
		t.Fatal("expected error for empty subscriptions")
	}
	if err.Error() != "subscriptions cannot be empty" {
		t.Errorf("error = %q, want 'subscriptions cannot be empty'", err.Error())
	}
}

func TestNewPushConsumer_EmptySubscriptionsMap(t *testing.T) {
	cfg := NewPushConsumerConfigFromConfig(&Config{
		Endpoint:      "localhost:8081",
		ConsumerGroup: "test-group",
	})

	subs := map[string]*rmq.FilterExpression{}
	_, _, err := NewPushConsumer(cfg, subs, func(msg *MessageView) ConsumerResult {
		return ConsumeSuccess
	}, log.DefaultLogger)
	if err == nil {
		t.Fatal("expected error for empty subscriptions map")
	}
}

func TestNewPushConsumer_NilHandler(t *testing.T) {
	cfg := NewPushConsumerConfigFromConfig(&Config{
		Endpoint:      "localhost:8081",
		ConsumerGroup: "test-group",
	})
	subs := map[string]*rmq.FilterExpression{
		"test-topic": SubAll,
	}

	_, _, err := NewPushConsumer(cfg, subs, nil, log.DefaultLogger)
	if err == nil {
		t.Fatal("expected error for nil handler")
	}
	if err.Error() != "handler cannot be nil" {
		t.Errorf("error = %q, want 'handler cannot be nil'", err.Error())
	}
}
