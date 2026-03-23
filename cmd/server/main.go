package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/encoding/json"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/go-kratos/kratos-layout/internal/job"
	"github.com/go-kratos/kratos-layout/pkg/env"
	zapLog "github.com/go-kratos/kratos-layout/pkg/log"
	"github.com/go-kratos/kratos-layout/pkg/registry"
	"github.com/go-kratos/kratos-layout/pkg/registry/nacos"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// id is the service instance id.
	id string
	// Command line flags
	flagConf string
)

func init() {
	json.MarshalOptions = protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}

	var err error
	id, err = os.Hostname()
	if err != nil {
		id = "unknown"
	}

	if Name == "" {
		Name = env.GetOrDefault("SERVICE_NAME", "xxx-service")
	}

	if Version == "" {
		Version = env.GetOrDefault("SERVICE_VERSION", "0.0.1")
	}
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server, r *nacos.Registry, jobs *job.Registry) *kratos.App {
	servers := make([]transport.Server, 0, 2+len(jobs.Servers()))
	servers = append(servers, gs, hs)
	servers = append(servers, jobs.Servers()...)
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(servers...),
		kratos.Registrar(r),
	)
}

func main() {
	flag.StringVar(&flagConf, "conf", "", "config file path (e.g., ./configs/config.yaml)")
	flag.Parse()

	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	logger := zapLog.InitDefaultLogger(parseLogLevel())
	logHelper := log.NewHelper(logger)

	// Load configuration
	bc, cleanup, err := loadConfig()
	if err != nil {
		logHelper.Errorf("failed to load config: %v", err)
		return err
	}
	defer cleanup()

	r, err := registry.NewNacosRegistryFromEnv()
	if err != nil {
		logHelper.Errorf("failed to create nacos registry: %v", err)
		return err
	}

	app, appCleanup, err := wireApp(bc.Server, bc.Data, r, logger)
	if err != nil {
		logHelper.Errorf("failed to wire app: %v", err)
		return err
	}
	defer appCleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		logHelper.Errorf("app exited with error: %v", err)
		return err
	}
	return nil
}

// loadConfig loads configuration from file.
// Priority: -conf flag > CONFIG_FILE env
func loadConfig() (*conf.Bootstrap, func(), error) {
	confFile := flagConf
	if confFile == "" {
		confFile = env.GetOrDefault("CONFIG_FILE", "")
	}

	if confFile == "" {
		return nil, nil, fmt.Errorf("config file not specified: use -conf flag or CONFIG_FILE env")
	}

	var bc conf.Bootstrap

	c := config.New(
		config.WithSource(
			file.NewSource(confFile),
		),
	)

	if err := c.Load(); err != nil {
		return nil, nil, err
	}

	if err := c.Scan(&bc); err != nil {
		return nil, nil, err
	}

	return &bc, func() { c.Close() }, nil
}

// parseLogLevel parses the LOG_LEVEL environment variable to a zapcore.Level.
// Defaults to InfoLevel for production safety.
func parseLogLevel() zapcore.Level {
	switch strings.ToLower(env.GetOrDefault("LOG_LEVEL", "info")) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
