package kafka

import (
	"strings"
	"time"

	"github.com/go-kratos/kratos-layout/internal/conf"
)

const (
	defaultWriteTimeout = 10 * time.Second
	defaultReadTimeout  = 10 * time.Second
)

// Config holds Kafka client configuration.
type Config struct {
	Brokers      []string      // Broker addresses (e.g., ["127.0.0.1:9092"])
	GroupID      string        // Consumer group ID
	WriteTimeout time.Duration // Producer write timeout (default 10s)
	ReadTimeout  time.Duration // Consumer read timeout (default 10s)
}

// NewConfigFromProto creates a Config from proto configuration.
func NewConfigFromProto(c *conf.Kafka) *Config {
	cfg := &Config{
		GroupID:      c.GroupId,
		WriteTimeout: defaultWriteTimeout,
		ReadTimeout:  defaultReadTimeout,
	}

	// Parse comma-separated brokers
	for _, b := range strings.Split(c.Brokers, ",") {
		if b = strings.TrimSpace(b); b != "" {
			cfg.Brokers = append(cfg.Brokers, b)
		}
	}

	if c.WriteTimeout != nil && c.WriteTimeout.AsDuration() > 0 {
		cfg.WriteTimeout = c.WriteTimeout.AsDuration()
	}
	if c.ReadTimeout != nil && c.ReadTimeout.AsDuration() > 0 {
		cfg.ReadTimeout = c.ReadTimeout.AsDuration()
	}

	return cfg
}
