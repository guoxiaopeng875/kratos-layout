package data

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/go-kratos/kratos-layout/pkg/orm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData, NewTransaction,
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

// NewTransaction returns a shard.Transaction backed by Data.
func NewTransaction(d *Data) biz.Transaction {
	return d
}

// Redis returns the redis.Client instance.
func (d *Data) Redis() *redis.Client {
	return d.rdb
}

// NewData creates a new Data instance and returns a cleanup function.
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	logHelper := log.NewHelper(logger)

	dbConf := &orm.DBConfig{
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

	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		Password:     c.Redis.Password,
		DB:           int(c.Redis.Db),
		DialTimeout:  c.Redis.DialTimeout.AsDuration(),
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
	})

	// add redis ping check
	pingTimeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := rdb.Ping(pingTimeoutCtx).Result(); err != nil {
		logHelper.Errorf("failed to ping redis: %v", err)
		return nil, nil, err
	}

	cleanup := func() {
		logHelper.Info("closing the data resources")

		if err := rdb.Close(); err != nil {
			logHelper.Errorf("failed to close redis data resources: %v", err)
		}

		if err := ormDB.Close(); err != nil {
			logHelper.Errorf("failed to close database data resources: %v", err)
		}
	}

	return &Data{
		db:  ormDB.GetDB(),
		rdb: rdb,
	}, cleanup, nil
}
