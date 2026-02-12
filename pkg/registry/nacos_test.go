package registry

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseServerAddrs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []ServerAddr
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []ServerAddr{{IP: "127.0.0.1", Port: 8848}},
		},
		{
			name:     "single address with port",
			input:    "192.168.1.1:8848",
			expected: []ServerAddr{{IP: "192.168.1.1", Port: 8848}},
		},
		{
			name:     "single address without port",
			input:    "192.168.1.1",
			expected: []ServerAddr{{IP: "192.168.1.1", Port: 8848}},
		},
		{
			name:  "multiple addresses with ports",
			input: "192.168.1.1:8848,192.168.1.2:8849",
			expected: []ServerAddr{
				{IP: "192.168.1.1", Port: 8848},
				{IP: "192.168.1.2", Port: 8849},
			},
		},
		{
			name:  "multiple addresses mixed",
			input: "192.168.1.1:8848,192.168.1.2",
			expected: []ServerAddr{
				{IP: "192.168.1.1", Port: 8848},
				{IP: "192.168.1.2", Port: 8848},
			},
		},
		{
			name:  "addresses with spaces",
			input: "192.168.1.1:8848 , 192.168.1.2:8849",
			expected: []ServerAddr{
				{IP: "192.168.1.1", Port: 8848},
				{IP: "192.168.1.2", Port: 8849},
			},
		},
		{
			name:  "addresses with empty parts",
			input: "192.168.1.1:8848,,192.168.1.2:8849",
			expected: []ServerAddr{
				{IP: "192.168.1.1", Port: 8848},
				{IP: "192.168.1.2", Port: 8849},
			},
		},
		{
			name:     "only commas",
			input:    ",,,",
			expected: []ServerAddr{{IP: "127.0.0.1", Port: 8848}},
		},
		{
			name:     "custom port",
			input:    "10.0.0.1:9999",
			expected: []ServerAddr{{IP: "10.0.0.1", Port: 9999}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseServerAddrs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseServerAddr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ServerAddr
	}{
		{
			name:     "with port",
			input:    "192.168.1.1:8848",
			expected: ServerAddr{IP: "192.168.1.1", Port: 8848},
		},
		{
			name:     "without port",
			input:    "192.168.1.1",
			expected: ServerAddr{IP: "192.168.1.1", Port: 8848},
		},
		{
			name:     "invalid port",
			input:    "192.168.1.1:abc",
			expected: ServerAddr{IP: "192.168.1.1:abc", Port: 8848},
		},
		{
			name:     "localhost with port",
			input:    "localhost:8848",
			expected: ServerAddr{IP: "localhost", Port: 8848},
		},
		{
			name:     "ipv6 address",
			input:    "[::1]:8848",
			expected: ServerAddr{IP: "[::1]", Port: 8848},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseServerAddr(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewNacosConfigFromEnv(t *testing.T) {
	// Clean up environment after test
	defer func() {
		os.Unsetenv(EnvNacosServerAddrs)
		os.Unsetenv(EnvNacosNamespaceID)
		os.Unsetenv(EnvNacosLogDir)
		os.Unsetenv(EnvNacosCacheDir)
		os.Unsetenv(EnvNacosLogLevel)
	}()

	t.Run("default values", func(t *testing.T) {
		os.Unsetenv(EnvNacosServerAddrs)
		os.Unsetenv(EnvNacosNamespaceID)
		os.Unsetenv(EnvNacosLogDir)
		os.Unsetenv(EnvNacosCacheDir)
		os.Unsetenv(EnvNacosLogLevel)

		cfg := NewNacosConfigFromEnv()

		assert.Equal(t, []ServerAddr{{IP: "127.0.0.1", Port: 8848}}, cfg.ServerAddrs)
		assert.Equal(t, "", cfg.NamespaceID)
		assert.Equal(t, DefaultNacosLogDir, cfg.LogDir)
		assert.Equal(t, DefaultNacosCacheDir, cfg.CacheDir)
		assert.Equal(t, DefaultNacosLogLevel, cfg.LogLevel)
	})

	t.Run("custom values", func(t *testing.T) {
		os.Setenv(EnvNacosServerAddrs, "10.0.0.1:8848,10.0.0.2:8848")
		os.Setenv(EnvNacosNamespaceID, "test-namespace")
		os.Setenv(EnvNacosLogDir, "/var/log/nacos")
		os.Setenv(EnvNacosCacheDir, "/var/cache/nacos")
		os.Setenv(EnvNacosLogLevel, "debug")

		cfg := NewNacosConfigFromEnv()

		assert.Equal(t, []ServerAddr{
			{IP: "10.0.0.1", Port: 8848},
			{IP: "10.0.0.2", Port: 8848},
		}, cfg.ServerAddrs)
		assert.Equal(t, "test-namespace", cfg.NamespaceID)
		assert.Equal(t, "/var/log/nacos", cfg.LogDir)
		assert.Equal(t, "/var/cache/nacos", cfg.CacheDir)
		assert.Equal(t, "debug", cfg.LogLevel)
	})
}
