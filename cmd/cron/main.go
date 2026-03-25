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
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/go-kratos/kratos-layout/internal/job"
	"github.com/go-kratos/kratos-layout/pkg/env"
	zapLog "github.com/go-kratos/kratos-layout/pkg/log"
)

var (
	Name     string
	Version  string
	id       string
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
		Name = env.GetOrDefault("SERVICE_NAME", "xxx-cron")
	}

	if Version == "" {
		Version = env.GetOrDefault("SERVICE_VERSION", "0.0.1")
	}
}

func newApp(logger log.Logger, cronServer *job.CronServer) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(cronServer),
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

	bc, cleanup, err := loadConfig()
	if err != nil {
		logHelper.Errorf("failed to load config: %v", err)
		return err
	}
	defer cleanup()

	app, appCleanup, err := wireApp(bc.CronTasks, logger)
	if err != nil {
		logHelper.Errorf("failed to wire app: %v", err)
		return err
	}
	defer appCleanup()

	if err := app.Run(); err != nil {
		logHelper.Errorf("app exited with error: %v", err)
		return err
	}
	return nil
}

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
