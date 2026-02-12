package rocketmq

import (
	"strings"
	"sync"
	"time"

	rmq "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/apache/rocketmq-clients/golang/v5/credentials"

	"github.com/go-kratos/kratos-layout/internal/conf"
)

var sslOnce sync.Once

// configureSSL sets the global SSL flag once in a thread-safe manner.
// The first call determines the value; subsequent calls are no-ops.
func configureSSL(enable bool) {
	sslOnce.Do(func() {
		rmq.EnableSsl = enable
	})
}

// Config holds RocketMQ client configuration for v5 SDK.
type Config struct {
	Endpoint      string                          // gRPC endpoint (e.g., "127.0.0.1:8081")
	NameSpace     string                          // Optional namespace
	ConsumerGroup string                          // Consumer group name
	Credentials   *credentials.SessionCredentials // Authentication credentials
	SendTimeout   time.Duration                   // Message send timeout
	MaxAttempts   int32                           // Max retry attempts for producer
	EnableSSL     bool                            // Whether to enable SSL
}

// NewConfigFromProto creates a Config from proto configuration.
// For v5 SDK, name_servers is treated as the gRPC endpoint.
func NewConfigFromProto(c *conf.RocketMQ) *Config {
	cfg := &Config{
		ConsumerGroup: c.ProducerGroup,
		SendTimeout:   3 * time.Second,
		MaxAttempts:   3,
		EnableSSL:     false,
		Credentials: &credentials.SessionCredentials{
			AccessKey:    c.AccessKey,
			AccessSecret: c.SecretKey,
		},
	}

	// Use name_servers as endpoint for v5 SDK
	// v5 uses gRPC protocol with a single endpoint
	servers := strings.ReplaceAll(c.NameServers, ";", ",")
	parts := strings.Split(servers, ",")
	if len(parts) > 0 {
		cfg.Endpoint = strings.TrimSpace(parts[0])
	}

	if c.SendTimeout != nil {
		cfg.SendTimeout = c.SendTimeout.AsDuration()
	}
	if c.RetryTimes > 0 {
		cfg.MaxAttempts = c.RetryTimes
	}

	return cfg
}

// ToRMQConfig converts Config to RocketMQ v5 SDK Config.
func (c *Config) ToRMQConfig() *rmq.Config {
	return &rmq.Config{
		Endpoint:      c.Endpoint,
		NameSpace:     c.NameSpace,
		ConsumerGroup: c.ConsumerGroup,
		Credentials:   c.Credentials,
	}
}
