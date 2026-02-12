package registry

import (
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"

	"github.com/go-kratos/kratos-layout/pkg/env"
	"github.com/go-kratos/kratos-layout/pkg/registry/nacos"
)

// Environment variable keys for Nacos configuration.
const (
	EnvNacosServerAddrs = "NACOS_SERVER_ADDRS" // Comma-separated list of server addresses (e.g., "192.168.1.1:8848,192.168.1.2:8848")
	EnvNacosNamespaceID = "NACOS_NAMESPACE_ID" // Namespace ID
	EnvNacosLogDir      = "NACOS_LOG_DIR"      // Log directory
	EnvNacosCacheDir    = "NACOS_CACHE_DIR"    // Cache directory
	EnvNacosLogLevel    = "NACOS_LOG_LEVEL"    // Log level (debug, info, warn, error)
)

// Default values for Nacos configuration.
const (
	DefaultNacosServerAddr = "127.0.0.1:8848"
	DefaultNacosLogDir     = "/tmp/nacos/log"
	DefaultNacosCacheDir   = "/tmp/nacos/cache"
	DefaultNacosLogLevel   = "warn"
)

// NacosConfig holds the configuration for Nacos client.
type NacosConfig struct {
	ServerAddrs []ServerAddr
	NamespaceID string
	LogDir      string
	CacheDir    string
	LogLevel    string
}

// ServerAddr represents a Nacos server address.
type ServerAddr struct {
	IP   string
	Port uint64
}

// NewNacosConfigFromEnv creates a NacosConfig from environment variables.
func NewNacosConfigFromEnv() *NacosConfig {
	return &NacosConfig{
		ServerAddrs: parseServerAddrs(env.GetOrDefault(EnvNacosServerAddrs, DefaultNacosServerAddr)),
		NamespaceID: env.Get(EnvNacosNamespaceID),
		LogDir:      env.GetOrDefault(EnvNacosLogDir, DefaultNacosLogDir),
		CacheDir:    env.GetOrDefault(EnvNacosCacheDir, DefaultNacosCacheDir),
		LogLevel:    env.GetOrDefault(EnvNacosLogLevel, DefaultNacosLogLevel),
	}
}

// parseServerAddrs parses a comma-separated list of server addresses.
// Format: "ip1:port1,ip2:port2" or "ip1,ip2" (default port 8848)
func parseServerAddrs(addrs string) []ServerAddr {
	if addrs == "" {
		return []ServerAddr{{IP: "127.0.0.1", Port: 8848}}
	}

	parts := strings.Split(addrs, ",")
	result := make([]ServerAddr, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		addr := parseServerAddr(part)
		result = append(result, addr)
	}

	if len(result) == 0 {
		return []ServerAddr{{IP: "127.0.0.1", Port: 8848}}
	}

	return result
}

// parseServerAddr parses a single server address in "ip:port" format.
func parseServerAddr(addr string) ServerAddr {
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		ip := addr[:idx]
		portStr := addr[idx+1:]
		if port, err := strconv.ParseUint(portStr, 10, 64); err == nil {
			return ServerAddr{IP: ip, Port: port}
		}
	}
	// Default port if not specified or invalid
	return ServerAddr{IP: addr, Port: 8848}
}

// NewNacosNamingClient creates a Nacos naming client from configuration.
func NewNacosNamingClient(cfg *NacosConfig) (naming_client.INamingClient, error) {
	serverConfigs := make([]constant.ServerConfig, 0, len(cfg.ServerAddrs))
	for _, addr := range cfg.ServerAddrs {
		serverConfigs = append(serverConfigs, constant.ServerConfig{
			IpAddr: addr.IP,
			Port:   addr.Port,
		})
	}

	clientConfig := &constant.ClientConfig{
		NamespaceId:         cfg.NamespaceID,
		NotLoadCacheAtStart: true,
		LogDir:              cfg.LogDir,
		CacheDir:            cfg.CacheDir,
		LogLevel:            cfg.LogLevel,
	}

	return clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
}

// NewNacosRegistry creates a Kratos registry using Nacos.
func NewNacosRegistry(client naming_client.INamingClient) *nacos.Registry {
	return nacos.New(client)
}

// NewNacosRegistryFromEnv creates a Nacos registry from environment variables.
// This is a convenience function that combines configuration loading, client creation, and registry creation.
func NewNacosRegistryFromEnv() (*nacos.Registry, error) {
	cfg := NewNacosConfigFromEnv()
	client, err := NewNacosNamingClient(cfg)
	if err != nil {
		return nil, err
	}
	return NewNacosRegistry(client), nil
}
