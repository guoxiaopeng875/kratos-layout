package data

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	gormtracing "gorm.io/plugin/opentelemetry/tracing"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/go-kratos/kratos-layout/pkg/metrics"
	"github.com/go-kratos/kratos-layout/pkg/orm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewRedisClient,
	NewData,
	NewTransaction,
	NewGreeterRepo,
)

// contextTxKey is the context key for storing a GORM transaction.
type contextTxKey struct{}

// Data is the data layer dependency container.
type Data struct {
	db  *gorm.DB
	rdb *redis.Client
}

// DB returns a context-aware *gorm.DB.
// If a transaction was started via InTx, returns the transaction;
// otherwise returns the default database session with the given context.
func (d *Data) DB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(contextTxKey{}).(*gorm.DB); ok {
		return tx
	}
	return d.db.WithContext(ctx)
}

// InTx executes fn within a database transaction.
// The transaction is stored in context so that all repos using DB(ctx) share it.
func (d *Data) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, contextTxKey{}, tx))
	})
}

// NewTransaction returns a biz.Transaction backed by Data.
func NewTransaction(d *Data) biz.Transaction {
	return d
}

// Redis returns the redis.Client instance.
func (d *Data) Redis() *redis.Client {
	return d.rdb
}

// NewRedisClient creates a Redis client from config and returns a cleanup
// function. The client has OpenTelemetry tracing and the Prometheus metrics
// hook installed.
func NewRedisClient(c *conf.Data, logger log.Logger) (*redis.Client, func(), error) {
	logHelper := log.NewHelper(logger)

	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		Password:     c.Redis.Password,
		DB:           int(c.Redis.Db),
		DialTimeout:  c.Redis.DialTimeout.AsDuration(),
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
	})

	pingTimeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := rdb.Ping(pingTimeoutCtx).Result(); err != nil {
		logHelper.Errorf("failed to ping redis: %v", err)
		return nil, nil, err
	}

	if err := redisotel.InstrumentTracing(rdb); err != nil {
		logHelper.Errorf("failed to install redis tracing: %v", err)
		return nil, nil, err
	}

	rdb.AddHook(metrics.NewRedisHook())

	cleanup := func() {
		if err := rdb.Close(); err != nil {
			logHelper.Errorf("failed to close redis: %v", err)
		}
	}

	return rdb, cleanup, nil
}

// NewData creates a new Data instance and returns a cleanup function. The
// underlying *gorm.DB has the OpenTelemetry tracing plugin installed and its
// connection pool is registered as a Prometheus collector.
func NewData(c *conf.Data, rdb *redis.Client, logger log.Logger) (*Data, func(), error) {
	logHelper := log.NewHelper(logger)

	dbConf := &orm.DBConfig{
		Driver:          c.Database.Driver,
		Username:        c.Database.Username,
		Password:        c.Database.Password,
		Host:            c.Database.Host,
		Port:            fmt.Sprintf("%d", c.Database.Port),
		DBName:          c.Database.DbName,
		MaxIdleConns:    int(c.Database.MaxIdleConns),
		MaxOpenConns:    int(c.Database.MaxOpenConns),
		DBCharset:       c.Database.DbCharset,
		ConnMaxLifetime: c.Database.ConnMaxLifetime.AsDuration(),
		ConnMaxIdleTime: c.Database.ConnMaxIdleTime.AsDuration(),
	}

	ormDB, err := orm.MakeDB(dbConf)
	if err != nil {
		return nil, nil, err
	}

	if pluginErr := ormDB.GetDB().Use(gormtracing.NewPlugin(
		gormtracing.WithoutMetrics(),
		gormtracing.WithoutQueryVariables(),
	)); pluginErr != nil {
		return nil, nil, fmt.Errorf("install gorm tracing plugin: %w", pluginErr)
	}

	cleanup := func() {
		logHelper.Info("closing the data resources")
		if err := ormDB.Close(); err != nil {
			logHelper.Errorf("failed to close database data resources: %v", err)
		}
	}

	d := &Data{
		db:  ormDB.GetDB(),
		rdb: rdb,
	}

	registerDBCollector(d.db)

	return d, cleanup, nil
}

// registerDBCollector registers a Prometheus collector for the DB connection pool.
func registerDBCollector(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	metrics.RegisterCollector(metrics.NewDBCollector(sqlDB))
}
