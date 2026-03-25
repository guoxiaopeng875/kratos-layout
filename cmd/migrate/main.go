package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"

	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/go-kratos/kratos-layout/internal/data/model"
	"github.com/go-kratos/kratos-layout/pkg/orm"
)

var flagConf string

func main() {
	flag.StringVar(&flagConf, "conf", "", "config file path (e.g., ./configs/config.yaml)")
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "migrate failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("migrate completed successfully")
}

func run() error {
	bc, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	dbConf := &orm.DBConfig{
		Driver:          bc.Data.Database.Driver,
		Username:        bc.Data.Database.Username,
		Password:        bc.Data.Database.Password,
		Host:            bc.Data.Database.Host,
		Port:            fmt.Sprintf("%d", bc.Data.Database.Port),
		DBName:          bc.Data.Database.DbName,
		MaxIdleConns:    int(bc.Data.Database.MaxIdleConns),
		MaxOpenConns:    int(bc.Data.Database.MaxOpenConns),
		DBCharset:       bc.Data.Database.DbCharset,
		ConnMaxLifetime: bc.Data.Database.ConnMaxLifetime.AsDuration(),
		ConnMaxIdleTime: bc.Data.Database.ConnMaxIdleTime.AsDuration(),
	}

	// Step 1: Create database if not exists
	dbUtil, err := orm.MakeDBUtil(dbConf)
	if err != nil {
		return fmt.Errorf("connect to system database: %w", err)
	}
	if createErr := dbUtil.CreateDB(); createErr != nil {
		dbUtil.Close()
		return fmt.Errorf("create database: %w", createErr)
	}
	dbUtil.Close()
	fmt.Printf("database %q ensured\n", dbConf.DBName)

	// Step 2: AutoMigrate tables
	db, err := orm.MakeDB(dbConf)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()

	if err := db.GetDB().AutoMigrate(
		&model.Greeter{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	fmt.Println("tables migrated")

	return nil
}

func loadConfig() (*conf.Bootstrap, error) {
	confFile := flagConf
	if confFile == "" {
		confFile = os.Getenv("CONFIG_FILE")
	}
	if confFile == "" {
		return nil, fmt.Errorf("config file is required: use -conf flag or CONFIG_FILE env var")
	}

	var bc conf.Bootstrap
	c := config.New(config.WithSource(file.NewSource(confFile)))
	if err := c.Load(); err != nil {
		return nil, err
	}
	defer c.Close()

	if err := c.Scan(&bc); err != nil {
		return nil, err
	}
	return &bc, nil
}
