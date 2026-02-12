package orm

import (
	"database/sql"
	"flag"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/mysql"
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

// getCharset returns the charset, defaulting to utf8mb4
func (c *DBConfig) getCharset() string {
	if c.DBCharset == "" {
		return "utf8mb4"
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

// quoteIdentifier escapes a SQL identifier to prevent SQL injection
func quoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func MakeDBUtil(dbConfig *DBConfig) (DBUtil, error) {
	return newGormMysql(dbConfig, true)
}

func MakeDB(dbConfig *DBConfig) (DB, error) {
	return newGormMysql(dbConfig, false)
}

func newGormMysql(dbConfig *DBConfig, forUtil bool) (*gormMysql, error) {
	gm := &gormMysql{dbConfig: dbConfig}

	var err error
	if forUtil {
		err = gm.initUtilDB()
	} else {
		err = gm.initGormDB()
	}

	if err != nil {
		return nil, err
	}

	return gm, nil
}

type gormMysql struct {
	dbConfig *DBConfig
	db       *gorm.DB
	utilDB   *gorm.DB
	sqlDB    *sql.DB
}

// Close closes the database connection
func (gm *gormMysql) Close() error {
	if gm.sqlDB != nil {
		return gm.sqlDB.Close()
	}
	return nil
}

// CreateDB creates the database if it does not exist
func (gm *gormMysql) CreateDB() error {
	if gm.utilDB == nil {
		return fmt.Errorf("util db is nil, please use MakeDBUtil first")
	}

	dbName := quoteIdentifier(gm.dbConfig.DBName)
	charset := gm.dbConfig.getCharset()
	createDBSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s DEFAULT CHARSET %s COLLATE %s_general_ci;",
		dbName, charset, charset)

	if err := gm.utilDB.Exec(createDBSQL).Error; err != nil {
		return fmt.Errorf("create db failed: %w", err)
	}

	return nil
}

// DropDB drops the database if it exists
func (gm *gormMysql) DropDB() error {
	if gm.utilDB == nil {
		return fmt.Errorf("util db is nil, please use MakeDBUtil first")
	}

	dbName := quoteIdentifier(gm.dbConfig.DBName)
	dropDBSQL := fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName)

	if err := gm.utilDB.Exec(dropDBSQL).Error; err != nil {
		return fmt.Errorf("drop db failed: %w", err)
	}

	return nil
}

// GetUtilDB returns the utility database connection for database management operations
func (gm *gormMysql) GetUtilDB() *gorm.DB {
	return gm.utilDB
}

// GetDB returns the main database connection
func (gm *gormMysql) GetDB() *gorm.DB {
	return gm.db
}

// ClearAllData clears all data from all tables (only works in test environment with test/dev database)
func (gm *gormMysql) ClearAllData() error {
	if flag.Lookup("test.v") == nil {
		return fmt.Errorf("ClearAllData can only be called in test environment")
	}

	if !strings.Contains(gm.dbConfig.DBName, "test") && !strings.Contains(gm.dbConfig.DBName, "dev") {
		return fmt.Errorf("ClearAllData can only be used with test or dev database, got: %s", gm.dbConfig.DBName)
	}

	if gm.db == nil {
		return fmt.Errorf("db is nil, please init db first")
	}

	rs, err := gm.db.Raw("SHOW TABLES;").Rows()
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

		quotedTable := quoteIdentifier(tName)
		if err := gm.db.Exec(fmt.Sprintf("DELETE FROM %s", quotedTable)).Error; err != nil {
			return fmt.Errorf("clear data from table %s failed: %w", tName, err)
		}
	}

	if err := rs.Err(); err != nil {
		return fmt.Errorf("iterate tables failed: %w", err)
	}

	return nil
}

// openConnection creates a new database connection with the given DSN
func (gm *gormMysql) openConnection(dsn string, silent bool) (gormDB *gorm.DB, sqlDB *sql.DB, err error) {
	sqlDB, err = sql.Open("mysql", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB.SetMaxIdleConns(gm.dbConfig.MaxIdleConns)
	sqlDB.SetMaxOpenConns(gm.dbConfig.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(gm.dbConfig.getConnMaxLifetime())
	sqlDB.SetConnMaxIdleTime(gm.dbConfig.getConnMaxIdleTime())

	gormConfig := &gorm.Config{}
	if silent {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	gormDB, err = gorm.Open(mysql.New(mysql.Config{Conn: sqlDB}), gormConfig)
	if err != nil {
		sqlDB.Close()
		return nil, nil, fmt.Errorf("failed to open gorm: %w", err)
	}

	return gormDB, sqlDB, nil
}

// buildDSN constructs a MySQL DSN string
func (gm *gormMysql) buildDSN(dbName string) string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		gm.dbConfig.Username,
		gm.dbConfig.Password,
		gm.dbConfig.Host,
		gm.dbConfig.Port,
		dbName,
		gm.dbConfig.getCharset())
	if gm.dbConfig.MultiStatements {
		dsn += "&multiStatements=true"
	}
	return dsn
}

func (gm *gormMysql) initGormDB() error {
	if gm.db != nil {
		return fmt.Errorf("gorm db already initialized")
	}

	dsn := gm.buildDSN(gm.dbConfig.DBName)
	db, sqlDB, err := gm.openConnection(dsn, true)
	if err != nil {
		return err
	}

	gm.db = db
	gm.sqlDB = sqlDB
	return nil
}

func (gm *gormMysql) initUtilDB() error {
	if gm.utilDB != nil {
		return fmt.Errorf("util db already initialized")
	}

	dsn := gm.buildDSN("information_schema")
	db, sqlDB, err := gm.openConnection(dsn, false)
	if err != nil {
		return err
	}

	gm.utilDB = db
	gm.sqlDB = sqlDB
	return nil
}
