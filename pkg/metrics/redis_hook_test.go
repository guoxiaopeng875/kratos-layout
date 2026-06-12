package metrics_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/go-kratos/kratos-layout/pkg/metrics"
)

func init() {
	metrics.Setup(context.Background(), "test-service", log.DefaultLogger) //nolint:errcheck
}

func TestRedisHook_ProcessHook_Success(t *testing.T) {
	hook := metrics.NewRedisHook()

	cmd := redis.NewStringCmd(context.Background(), "GET", "mykey")

	processFn := hook.ProcessHook(func(ctx context.Context, cmd redis.Cmder) error {
		return nil
	})
	err := processFn(context.Background(), cmd)
	require.NoError(t, err)
}

func TestRedisHook_ProcessHook_Error(t *testing.T) {
	hook := metrics.NewRedisHook()

	cmd := redis.NewStringCmd(context.Background(), "SET", "mykey", "val")

	processFn := hook.ProcessHook(func(ctx context.Context, cmd redis.Cmder) error {
		return errors.New("connection refused")
	})
	err := processFn(context.Background(), cmd)
	require.Error(t, err)
}
