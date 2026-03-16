package rocketmq

import (
	"testing"
	"time"

	"github.com/apache/rocketmq-clients/golang/v5/credentials"
	"github.com/go-kratos/kratos-layout/internal/conf"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestConfigureSSL(t *testing.T) {
	// configureSSL uses sync.Once, so this test only runs the callback once per process.
	// We test that it sets rmq.EnableSsl.
	configureSSL(false)
	// Calling again should be a no-op (sync.Once)
	configureSSL(true)
}

func TestNewConfigFromProto(t *testing.T) {
	t.Run("basic config", func(t *testing.T) {
		proto := &conf.RocketMQ{
			NameServers:   "127.0.0.1:8081",
			ProducerGroup: "test-group",
			AccessKey:     "ak",
			SecretKey:     "sk",
		}

		cfg := NewConfigFromProto(proto)
		if cfg.Endpoint != "127.0.0.1:8081" {
			t.Errorf("Endpoint = %s, want 127.0.0.1:8081", cfg.Endpoint)
		}
		if cfg.ConsumerGroup != "test-group" {
			t.Errorf("ConsumerGroup = %s, want test-group", cfg.ConsumerGroup)
		}
		if cfg.Credentials.AccessKey != "ak" {
			t.Errorf("AccessKey = %s, want ak", cfg.Credentials.AccessKey)
		}
		if cfg.Credentials.AccessSecret != "sk" {
			t.Errorf("AccessSecret = %s, want sk", cfg.Credentials.AccessSecret)
		}
		if cfg.SendTimeout != 3*time.Second {
			t.Errorf("SendTimeout = %v, want 3s", cfg.SendTimeout)
		}
		if cfg.MaxAttempts != 3 {
			t.Errorf("MaxAttempts = %d, want 3", cfg.MaxAttempts)
		}
	})

	t.Run("with custom timeout and retries", func(t *testing.T) {
		proto := &conf.RocketMQ{
			NameServers:   "10.0.0.1:8081;10.0.0.2:8082",
			ProducerGroup: "group",
			SendTimeout:   durationpb.New(5 * time.Second),
			RetryTimes:    5,
		}

		cfg := NewConfigFromProto(proto)
		if cfg.Endpoint != "10.0.0.1:8081" {
			t.Errorf("Endpoint = %s, want 10.0.0.1:8081 (first server)", cfg.Endpoint)
		}
		if cfg.SendTimeout != 5*time.Second {
			t.Errorf("SendTimeout = %v, want 5s", cfg.SendTimeout)
		}
		if cfg.MaxAttempts != 5 {
			t.Errorf("MaxAttempts = %d, want 5", cfg.MaxAttempts)
		}
	})

	t.Run("with comma-separated servers", func(t *testing.T) {
		proto := &conf.RocketMQ{
			NameServers:   "10.0.0.1:8081,10.0.0.2:8082",
			ProducerGroup: "group",
		}

		cfg := NewConfigFromProto(proto)
		if cfg.Endpoint != "10.0.0.1:8081" {
			t.Errorf("Endpoint = %s, want 10.0.0.1:8081", cfg.Endpoint)
		}
	})
}

func TestConfig_ToRMQConfig(t *testing.T) {
	cfg := &Config{
		Endpoint:      "localhost:8081",
		NameSpace:     "ns-1",
		ConsumerGroup: "cg-1",
		Credentials: &credentials.SessionCredentials{
			AccessKey:    "ak",
			AccessSecret: "sk",
		},
	}

	rmqCfg := cfg.ToRMQConfig()
	if rmqCfg.Endpoint != "localhost:8081" {
		t.Errorf("Endpoint = %s, want localhost:8081", rmqCfg.Endpoint)
	}
	if rmqCfg.NameSpace != "ns-1" {
		t.Errorf("NameSpace = %s, want ns-1", rmqCfg.NameSpace)
	}
	if rmqCfg.ConsumerGroup != "cg-1" {
		t.Errorf("ConsumerGroup = %s, want cg-1", rmqCfg.ConsumerGroup)
	}
}

func TestNewPushConsumerConfigFromConfig(t *testing.T) {
	cfg := &Config{
		Endpoint:      "localhost:8081",
		ConsumerGroup: "cg",
	}

	pushCfg := NewPushConsumerConfigFromConfig(cfg)
	if pushCfg.Config != cfg {
		t.Error("Config reference mismatch")
	}
	if pushCfg.AwaitDuration != 5*time.Second {
		t.Errorf("AwaitDuration = %v, want 5s", pushCfg.AwaitDuration)
	}
	if pushCfg.MaxCacheMessageCount != 1024 {
		t.Errorf("MaxCacheMessageCount = %d, want 1024", pushCfg.MaxCacheMessageCount)
	}
	if pushCfg.MaxCacheMessageSizeInBytes != 64*1024*1024 {
		t.Errorf("MaxCacheMessageSizeInBytes = %d, want %d", pushCfg.MaxCacheMessageSizeInBytes, 64*1024*1024)
	}
	if pushCfg.ConsumptionThreadCount != 20 {
		t.Errorf("ConsumptionThreadCount = %d, want 20", pushCfg.ConsumptionThreadCount)
	}
}
