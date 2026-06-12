package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/encoding/json"
	"github.com/go-kratos/kratos/v2/log"
	kratostracing "github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/go-kratos/kratos-layout/pkg/env"
	zapLog "github.com/go-kratos/kratos-layout/pkg/log"
	"github.com/go-kratos/kratos-layout/pkg/metrics"
	"github.com/go-kratos/kratos-layout/pkg/registry"
	"github.com/go-kratos/kratos-layout/pkg/registry/nacos"
	"github.com/go-kratos/kratos-layout/pkg/tracing"
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

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server, r *nacos.Registry) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(gs, hs),
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
	baseLogger := zapLog.InitDefaultLogger(parseLogLevel())
	logHelper := log.NewHelper(baseLogger)

	// Load configuration
	bc, cleanup, err := loadConfig()
	if err != nil {
		logHelper.Errorf("failed to load config: %v", err)
		return err
	}
	defer cleanup()

	// Install global OpenTelemetry tracer provider + propagator.
	tracingCleanup, err := tracing.Setup(context.Background(), bc.GetTracing(), Name, baseLogger)
	if err != nil {
		logHelper.Errorf("failed to setup tracing: %v", err)
		return err
	}
	defer tracingCleanup(context.Background())

	// Install the dedicated Prometheus registry; HTTP server exposes /metrics
	// against this registry.
	metricsCleanup, err := metrics.Setup(context.Background(), Name, baseLogger)
	if err != nil {
		logHelper.Errorf("failed to setup metrics: %v", err)
		return err
	}
	defer metricsCleanup()

	// Decorate the base logger so every log line carries trace/span IDs from
	// the current OTel context.
	logger := log.With(baseLogger,
		"trace_id", kratostracing.TraceID(),
		"span_id", kratostracing.SpanID(),
	)

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
