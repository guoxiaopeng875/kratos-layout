package orm

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type testDatabaseConfig struct {
	Driver       string `yaml:"driver"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	DBName       string `yaml:"db_name"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	DBCharset    string `yaml:"db_charset"`
}

type testYAMLConfig struct {
	Data struct {
		Database testDatabaseConfig `yaml:"database"`
	} `yaml:"data"`
}

func loadTestDBConfig(t *testing.T) *DBConfig {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get current file path")

	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
	configPath := filepath.Join(projectRoot, "configs", ".local.config.yaml")

	data, err := os.ReadFile(configPath)
	require.NoError(t, err, "failed to read config file: %s", configPath)

	var cfg testYAMLConfig
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(t, err, "failed to parse config file")

	db := cfg.Data.Database
	return &DBConfig{
		Driver:       db.Driver,
		Username:     db.Username,
		Password:     db.Password,
		Host:         db.Host,
		Port:         fmt.Sprintf("%d", db.Port),
		DBName:       db.DBName,
		MaxIdleConns: db.MaxIdleConns,
		MaxOpenConns: db.MaxOpenConns,
		DBCharset:    db.DBCharset,
	}
}

// --- Unit tests (no database connection required) ---

func TestGetDialect(t *testing.T) {
	tests := []struct {
		name      string
		driver    string
		wantType  string
		wantError bool
	}{
		{"mysql driver", "mysql", "mysql", false},
		{"MySQL uppercase", "MySQL", "mysql", false},
		{"postgres driver", "postgres", "postgres", false},
		{"postgresql alias", "postgresql", "postgres", false},
		{"empty defaults to postgres", "", "postgres", false},
		{"invalid driver", "sqlite", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := getDialect(tt.driver)
			if tt.wantError {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unsupported database driver")
				return
			}
			require.NoError(t, err)
			switch tt.wantType {
			case "mysql":
				require.IsType(t, &mysqlDialect{}, d)
			case "postgres":
				require.IsType(t, &postgresDialect{}, d)
			}
		})
	}
}

func TestMysqlDialect_QuoteIdentifier(t *testing.T) {
	d := &mysqlDialect{}
	tests := []struct {
		input    string
		expected string
	}{
		{"users", "`users`"},
		{"my_table", "`my_table`"},
		{"table`name", "`table``name`"},
		{"db`test`table", "`db``test``table`"},
	}

	for _, tt := range tests {
		result := d.quoteIdentifier(tt.input)
		require.Equal(t, tt.expected, result)
	}
}

func TestPostgresDialect_QuoteIdentifier(t *testing.T) {
	d := &postgresDialect{}
	tests := []struct {
		input    string
		expected string
	}{
		{"users", `"users"`},
		{"my_table", `"my_table"`},
		{`table"name`, `"table""name"`},
		{`db"test"table`, `"db""test""table"`},
	}

	for _, tt := range tests {
		result := d.quoteIdentifier(tt.input)
		require.Equal(t, tt.expected, result)
	}
}

func TestMysqlDialect_BuildDSN(t *testing.T) {
	dbConf := &DBConfig{
		Username:     "testuser",
		Password:     "testpass",
		Host:         "localhost",
		Port:         "3307",
		DBName:       "testdb",
		MaxIdleConns: 10,
		MaxOpenConns: 100,
		DBCharset:    "utf8",
	}

	d := &mysqlDialect{}

	tests := []struct {
		name     string
		dbName   string
		expected string
	}{
		{
			name:     "default database",
			dbName:   "testdb",
			expected: "testuser:testpass@tcp(localhost:3307)/testdb?charset=utf8&parseTime=True&loc=Local",
		},
		{
			name:     "information_schema",
			dbName:   "information_schema",
			expected: "testuser:testpass@tcp(localhost:3307)/information_schema?charset=utf8&parseTime=True&loc=Local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.buildDSN(dbConf, tt.dbName)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMysqlDialect_BuildDSN_WithMultiStatements(t *testing.T) {
	dbConf := &DBConfig{
		Username:        "user",
		Password:        "pass",
		Host:            "host",
		Port:            "3306",
		MultiStatements: true,
	}

	d := &mysqlDialect{}
	result := d.buildDSN(dbConf, "testdb")
	require.Contains(t, result, "&multiStatements=true")
}

func TestMysqlDialect_BuildDSN_DefaultCharset(t *testing.T) {
	dbConf := &DBConfig{
		Username:  "user",
		Password:  "pass",
		Host:      "host",
		Port:      "3306",
		DBCharset: "",
	}

	d := &mysqlDialect{}
	result := d.buildDSN(dbConf, "testdb")
	require.Contains(t, result, "charset=utf8mb4")
}

func TestPostgresDialect_BuildDSN(t *testing.T) {
	dbConf := &DBConfig{
		Username: "pguser",
		Password: "pgpass",
		Host:     "pghost",
		Port:     "5432",
		DBName:   "pgdb",
	}

	d := &postgresDialect{}

	tests := []struct {
		name     string
		dbName   string
		expected string
	}{
		{
			name:     "default database",
			dbName:   "pgdb",
			expected: "host=pghost port=5432 user=pguser password=pgpass dbname=pgdb sslmode=disable",
		},
		{
			name:     "postgres system db",
			dbName:   "postgres",
			expected: "host=pghost port=5432 user=pguser password=pgpass dbname=postgres sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.buildDSN(dbConf, tt.dbName)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestPostgresDialect_BuildDSN_IgnoresMySQLParams(t *testing.T) {
	dbConf := &DBConfig{
		Username:        "user",
		Password:        "pass",
		Host:            "host",
		Port:            "5432",
		DBCharset:       "utf8mb4",
		MultiStatements: true,
	}

	d := &postgresDialect{}
	result := d.buildDSN(dbConf, "testdb")
	require.NotContains(t, result, "charset")
	require.NotContains(t, result, "multiStatements")
	require.Equal(t, "host=host port=5432 user=user password=pass dbname=testdb sslmode=disable", result)
}

func TestMysqlDialect_UtilDBName(t *testing.T) {
	d := &mysqlDialect{}
	require.Equal(t, "information_schema", d.utilDBName())
}

func TestPostgresDialect_UtilDBName(t *testing.T) {
	d := &postgresDialect{}
	require.Equal(t, "postgres", d.utilDBName())
}

func TestMysqlDialect_SQL(t *testing.T) {
	d := &mysqlDialect{}
	require.Contains(t, d.createDBSQL("mydb"), "CREATE DATABASE IF NOT EXISTS")
	require.Contains(t, d.dropDBSQL("mydb"), "DROP DATABASE IF EXISTS")
	require.Equal(t, "SHOW TABLES;", d.listTablesSQL())
	require.Contains(t, d.clearTableSQL("users"), "DELETE FROM")
}

func TestPostgresDialect_SQL(t *testing.T) {
	d := &postgresDialect{}
	require.Contains(t, d.createDBSQL("mydb"), "CREATE DATABASE")
	require.Contains(t, d.dropDBSQL("mydb"), "DROP DATABASE IF EXISTS")
	require.Contains(t, d.listTablesSQL(), "pg_tables")
	require.Contains(t, d.clearTableSQL("users"), "TRUNCATE TABLE")
	require.Contains(t, d.clearTableSQL("users"), "CASCADE")
}

func TestDBConfig_getCharset(t *testing.T) {
	tests := []struct {
		name     string
		config   *DBConfig
		expected string
	}{
		{"default charset", &DBConfig{DBCharset: ""}, "utf8mb4"},
		{"custom charset", &DBConfig{DBCharset: "utf8"}, "utf8"},
		{"utf8mb4 charset", &DBConfig{DBCharset: "utf8mb4"}, "utf8mb4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.getCharset()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDBConfig_getConnMaxLifetime(t *testing.T) {
	tests := []struct {
		name     string
		config   *DBConfig
		expected time.Duration
	}{
		{"default lifetime", &DBConfig{ConnMaxLifetime: 0}, time.Hour},
		{"custom lifetime", &DBConfig{ConnMaxLifetime: 2 * time.Hour}, 2 * time.Hour},
		{"30 minutes lifetime", &DBConfig{ConnMaxLifetime: 30 * time.Minute}, 30 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.getConnMaxLifetime()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDBConfig_getConnMaxIdleTime(t *testing.T) {
	tests := []struct {
		name     string
		config   *DBConfig
		expected time.Duration
	}{
		{"default idle time", &DBConfig{ConnMaxIdleTime: 0}, 10 * time.Minute},
		{"custom idle time", &DBConfig{ConnMaxIdleTime: 5 * time.Minute}, 5 * time.Minute},
		{"15 minutes idle time", &DBConfig{ConnMaxIdleTime: 15 * time.Minute}, 15 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.getConnMaxIdleTime()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGormDB_CreateDB_NilUtilDB(t *testing.T) {
	g := &gormDB{dbConfig: &DBConfig{}, d: &mysqlDialect{}}
	err := g.CreateDB()
	require.Error(t, err)
	require.Contains(t, err.Error(), "util db is nil")
}

func TestGormDB_DropDB_NilUtilDB(t *testing.T) {
	g := &gormDB{dbConfig: &DBConfig{}, d: &mysqlDialect{}}
	err := g.DropDB()
	require.Error(t, err)
	require.Contains(t, err.Error(), "util db is nil")
}

func TestGormDB_ClearAllData_NonTestDB(t *testing.T) {
	g := &gormDB{dbConfig: &DBConfig{DBName: "production_db"}, d: &mysqlDialect{}}
	err := g.ClearAllData()
	require.Error(t, err)
	require.Contains(t, err.Error(), "test or dev database")
}

func TestGormDB_ClearAllData_NilDB(t *testing.T) {
	g := &gormDB{dbConfig: &DBConfig{DBName: "app_test"}, d: &mysqlDialect{}}
	err := g.ClearAllData()
	require.Error(t, err)
	require.Contains(t, err.Error(), "db is nil")
}

func TestGormDB_Close_Nil(t *testing.T) {
	g := &gormDB{}
	err := g.Close()
	require.NoError(t, err)
}
