package kafka

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/go-kratos/kratos-layout/internal/conf"
)

func TestNewConfigFromProto_Basic(t *testing.T) {
	proto := &conf.Kafka{
		Brokers:      "127.0.0.1:9092",
		GroupId:      "my-group",
		WriteTimeout: durationpb.New(5 * time.Second),
		ReadTimeout:  durationpb.New(3 * time.Second),
	}

	cfg := NewConfigFromProto(proto)

	require.Equal(t, []string{"127.0.0.1:9092"}, cfg.Brokers)
	require.Equal(t, "my-group", cfg.GroupID)
	require.Equal(t, 5*time.Second, cfg.WriteTimeout)
	require.Equal(t, 3*time.Second, cfg.ReadTimeout)
}

func TestNewConfigFromProto_MultipleBrokers(t *testing.T) {
	proto := &conf.Kafka{
		Brokers: "host1:9092, host2:9092,  host3:9093 ",
		GroupId: "grp",
	}

	cfg := NewConfigFromProto(proto)

	require.Equal(t, []string{"host1:9092", "host2:9092", "host3:9093"}, cfg.Brokers)
}

func TestNewConfigFromProto_DefaultTimeouts(t *testing.T) {
	proto := &conf.Kafka{
		Brokers: "127.0.0.1:9092",
		GroupId: "grp",
		// WriteTimeout and ReadTimeout not set → should default to 10s
	}

	cfg := NewConfigFromProto(proto)

	require.Equal(t, 10*time.Second, cfg.WriteTimeout)
	require.Equal(t, 10*time.Second, cfg.ReadTimeout)
}

func TestNewConfigFromProto_ZeroDurationDefaultsTo10s(t *testing.T) {
	proto := &conf.Kafka{
		Brokers:      "127.0.0.1:9092",
		WriteTimeout: durationpb.New(0),
		ReadTimeout:  durationpb.New(0),
	}

	cfg := NewConfigFromProto(proto)

	require.Equal(t, 10*time.Second, cfg.WriteTimeout)
	require.Equal(t, 10*time.Second, cfg.ReadTimeout)
}

func TestNewConfigFromProto_EmptyBrokers(t *testing.T) {
	proto := &conf.Kafka{
		Brokers: "",
		GroupId: "grp",
	}

	cfg := NewConfigFromProto(proto)

	require.Empty(t, cfg.Brokers)
}

func TestNewConfigFromProto_EmptyGroupID(t *testing.T) {
	proto := &conf.Kafka{
		Brokers: "127.0.0.1:9092",
		GroupId: "",
	}

	cfg := NewConfigFromProto(proto)

	require.Equal(t, "", cfg.GroupID)
}
