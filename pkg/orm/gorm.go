package orm

import (
	"database/sql"
	"flag"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBUtil interface {
	CreateDB() error
	DropDB() error
	GetUtilDB() *gorm.DB
	Close() error
}

type DB interface {
	GetDB() *gorm.DB
	ClearAllData() error
	Close() error
}

// DBConfig is the configuration for the database
type DBConfig struct {
	Driver          string
	Username        string
	Password        string
	Host            string
	Port            string
	DBName          string
	MaxIdleConns    int
	MaxOpenConns    int
	DBCharset       string
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	MultiStatements bool
}

const (
	DriverPostgres = "postgres"
	DriverMySQL    = "mysql"

	defaultCharset = "utf8mb4"
)

// getCharset returns the charset, defaulting to utf8mb4
func (c *DBConfig) getCharset() string {
	if c.DBCharset == "" {
		return defaultCharset
	}
	return c.DBCharset
}

// getConnMaxLifetime returns the connection max lifetime, defaulting to 1 hour
func (c *DBConfig) getConnMaxLifetime() time.Duration {
	if c.ConnMaxLifetime == 0 {
		return time.Hour
	}
	return c.ConnMaxLifetime
}

// getConnMaxIdleTime returns the connection max idle time, defaulting to 10 minutes
func (c *DBConfig) getConnMaxIdleTime() time.Duration {
	if c.ConnMaxIdleTime == 0 {
		return 10 * time.Minute
	}
	return c.ConnMaxIdleTime
}

// dialect is the internal interface for database-specific SQL behavior
type dialect interface {
	buildDSN(config *DBConfig, dbName string) string
	dialector(dsn string) gorm.Dialector
	createDBSQL(dbName string) string
	dropDBSQL(dbName string) string
	listTablesSQL() string
	clearTableSQL(table string) string
	quoteIdentifier(name string) string
	utilDBName() string
}

func getDialect(driver string) (dialect, error) {
	switch strings.ToLower(driver) {
	case DriverMySQL:
		return &mysqlDialect{}, nil
	case DriverPostgres, "postgresql", "":
		return &postgresDialect{}, nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}

func MakeDBUtil(dbConfig *DBConfig) (DBUtil, error) {
	d, err := getDialect(dbConfig.Driver)
	if err != nil {
		return nil, err
	}
	return newGormDB(dbConfig, d, true)
}

func MakeDB(dbConfig *DBConfig) (DB, error) {
	d, err := getDialect(dbConfig.Driver)
	if err != nil {
		return nil, err
	}
	return newGormDB(dbConfig, d, false)
}

func newGormDB(dbConfig *DBConfig, d dialect, forUtil bool) (*gormDB, error) {
	gdb := &gormDB{dbConfig: dbConfig, d: d}

	var err error
	if forUtil {
		err = gdb.initUtilDB()
	} else {
		err = gdb.initGormDB()
	}

	if err != nil {
		return nil, err
	}

	return gdb, nil
}

type gormDB struct {
	dbConfig *DBConfig
	d        dialect
	db       *gorm.DB
	utilDB   *gorm.DB
	sqlDB    *sql.DB
}

// Close closes the database connection
func (g *gormDB) Close() error {
	if g.sqlDB != nil {
		return g.sqlDB.Close()
	}
	return nil
}

// CreateDB creates the database if it does not exist
func (g *gormDB) CreateDB() error {
	if g.utilDB == nil {
		return fmt.Errorf("util db is nil, please use MakeDBUtil first")
	}

	createSQL := g.d.createDBSQL(g.dbConfig.DBName)
	if err := g.utilDB.Exec(createSQL).Error; err != nil {
		// PostgreSQL does not support IF NOT EXISTS for CREATE DATABASE,
		// so we ignore the "already exists" error (SQLSTATE 42P04).
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return fmt.Errorf("create db failed: %w", err)
	}

	return nil
}

// DropDB drops the database if it exists
func (g *gormDB) DropDB() error {
	if g.utilDB == nil {
		return fmt.Errorf("util db is nil, please use MakeDBUtil first")
	}

	dropSQL := g.d.dropDBSQL(g.dbConfig.DBName)
	if err := g.utilDB.Exec(dropSQL).Error; err != nil {
		return fmt.Errorf("drop db failed: %w", err)
	}

	return nil
}

// GetUtilDB returns the utility database connection for database management operations
func (g *gormDB) GetUtilDB() *gorm.DB {
	return g.utilDB
}

// GetDB returns the main database connection
func (g *gormDB) GetDB() *gorm.DB {
	return g.db
}

// ClearAllData clears all data from all tables (only works in test environment with test/dev database)
func (g *gormDB) ClearAllData() error {
	if flag.Lookup("test.v") == nil {
		return fmt.Errorf("ClearAllData can only be called in test environment")
	}

	if !strings.Contains(g.dbConfig.DBName, "test") && !strings.Contains(g.dbConfig.DBName, "dev") {
		return fmt.Errorf("ClearAllData can only be used with test or dev database, got: %s", g.dbConfig.DBName)
	}

	if g.db == nil {
		return fmt.Errorf("db is nil, please init db first")
	}

	rs, err := g.db.Raw(g.d.listTablesSQL()).Rows()
	if err != nil {
		return fmt.Errorf("get table list failed: %w", err)
	}
	defer rs.Close()

	var tName string
	for rs.Next() {
		if err := rs.Scan(&tName); err != nil {
			return fmt.Errorf("scan table name failed: %w", err)
		}
		if tName == "" {
			continue
		}

		if err := g.db.Exec(g.d.clearTableSQL(tName)).Error; err != nil {
			return fmt.Errorf("clear data from table %s failed: %w", tName, err)
		}
	}

	if err := rs.Err(); err != nil {
		return fmt.Errorf("iterate tables failed: %w", err)
	}

	return nil
}

// openConnection creates a new database connection with the given DSN
func (g *gormDB) openConnection(dsn string, silent bool) (gormDatabase *gorm.DB, sqlDatabase *sql.DB, err error) {
	gormConfig := &gorm.Config{}
	if silent {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	gormDatabase, err = gorm.Open(g.d.dialector(dsn), gormConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open gorm: %w", err)
	}

	sqlDatabase, err = gormDatabase.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDatabase.SetMaxIdleConns(g.dbConfig.MaxIdleConns)
	sqlDatabase.SetMaxOpenConns(g.dbConfig.MaxOpenConns)
	sqlDatabase.SetConnMaxLifetime(g.dbConfig.getConnMaxLifetime())
	sqlDatabase.SetConnMaxIdleTime(g.dbConfig.getConnMaxIdleTime())

	return gormDatabase, sqlDatabase, nil
}

func (g *gormDB) initGormDB() error {
	if g.db != nil {
		return fmt.Errorf("gorm db already initialized")
	}

	dsn := g.d.buildDSN(g.dbConfig, g.dbConfig.DBName)
	db, sqlDB, err := g.openConnection(dsn, true)
	if err != nil {
		return err
	}

	g.db = db
	g.sqlDB = sqlDB
	return nil
}

func (g *gormDB) initUtilDB() error {
	if g.utilDB != nil {
		return fmt.Errorf("util db already initialized")
	}

	dsn := g.d.buildDSN(g.dbConfig, g.d.utilDBName())
	db, sqlDB, err := g.openConnection(dsn, false)
	if err != nil {
		return err
	}

	g.utilDB = db
	g.sqlDB = sqlDB
	return nil
}
