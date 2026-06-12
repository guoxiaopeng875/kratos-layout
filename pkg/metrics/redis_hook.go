package metrics

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisHook implements redis.Hook to record command metrics.
type RedisHook struct{}

// NewRedisHook creates a new RedisHook.
func NewRedisHook() *RedisHook {
	return &RedisHook{}
}

// DialHook passes through to the next hook.
func (h *RedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

// ProcessHook wraps command execution to record metrics.
func (h *RedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		duration := time.Since(start).Seconds()

		command := strings.ToLower(cmd.Name())
		result := "success"
		if err != nil && !errors.Is(err, redis.Nil) {
			result = "fail"
		}

		RedisCommandsTotal.WithLabelValues(command, result).Inc()
		RedisCommandDuration.WithLabelValues(command).Observe(duration)
		return err
	}
}

// ProcessPipelineHook wraps pipeline execution to record metrics.
func (h *RedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmds)
		duration := time.Since(start).Seconds()

		result := "success"
		if err != nil {
			result = "fail"
		}

		RedisCommandsTotal.WithLabelValues("pipeline", result).Inc()
		RedisCommandDuration.WithLabelValues("pipeline").Observe(duration)
		return err
	}
}
