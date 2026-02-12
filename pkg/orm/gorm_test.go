package orm

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var password = "root"

func TestMakeDBUtil(t *testing.T) {
	dbConf := &DBConfig{
		Username:     "root",
		Password:     password,
		Host:         "127.0.0.1",
		Port:         "3306",
		DBName:       "hahaha_test",
		MaxIdleConns: 10,
		MaxOpenConns: 100,
		DBCharset:    "utf8mb4",
	}

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)
	defer utilDB.Close()

	err = utilDB.CreateDB()
	require.NoError(t, err)

	err = utilDB.DropDB()
	require.NoError(t, err)
}

func TestMakeDB(t *testing.T) {
	dbConf := &DBConfig{
		Username:     "root",
		Password:     password,
		Host:         "127.0.0.1",
		Port:         "3306",
		DBName:       "hahaha_test",
		MaxIdleConns: 10,
		MaxOpenConns: 100,
		DBCharset:    "utf8mb4",
	}

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)
	defer utilDB.Close()

	err = utilDB.CreateDB()
	require.NoError(t, err)
	defer func() {
		dropErr := utilDB.DropDB()
		require.NoError(t, dropErr)
	}()

	db, err := MakeDB(dbConf)
	require.NoError(t, err)
	defer db.Close()

	err = db.ClearAllData()
	require.NoError(t, err)
}

func TestQuoteIdentifier(t *testing.T) {
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
		result := quoteIdentifier(tt.input)
		require.Equal(t, tt.expected, result)
	}
}

func TestDBConfig_getCharset(t *testing.T) {
	tests := []struct {
		name     string
		config   *DBConfig
		expected string
	}{
		{
			name:     "default charset",
			config:   &DBConfig{DBCharset: ""},
			expected: "utf8mb4",
		},
		{
			name:     "custom charset",
			config:   &DBConfig{DBCharset: "utf8"},
			expected: "utf8",
		},
		{
			name:     "utf8mb4 charset",
			config:   &DBConfig{DBCharset: "utf8mb4"},
			expected: "utf8mb4",
		},
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
		{
			name:     "default lifetime",
			config:   &DBConfig{ConnMaxLifetime: 0},
			expected: time.Hour,
		},
		{
			name:     "custom lifetime",
			config:   &DBConfig{ConnMaxLifetime: 2 * time.Hour},
			expected: 2 * time.Hour,
		},
		{
			name:     "30 minutes lifetime",
			config:   &DBConfig{ConnMaxLifetime: 30 * time.Minute},
			expected: 30 * time.Minute,
		},
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
		{
			name:     "default idle time",
			config:   &DBConfig{ConnMaxIdleTime: 0},
			expected: 10 * time.Minute,
		},
		{
			name:     "custom idle time",
			config:   &DBConfig{ConnMaxIdleTime: 5 * time.Minute},
			expected: 5 * time.Minute,
		},
		{
			name:     "15 minutes idle time",
			config:   &DBConfig{ConnMaxIdleTime: 15 * time.Minute},
			expected: 15 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.getConnMaxIdleTime()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGormMysql_GetUtilDB(t *testing.T) {
	dbConf := &DBConfig{
		Username:     "root",
		Password:     password,
		Host:         "127.0.0.1",
		Port:         "3306",
		DBName:       "hahaha_test",
		MaxIdleConns: 10,
		MaxOpenConns: 100,
		DBCharset:    "utf8mb4",
	}

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)
	defer utilDB.Close()

	gm := utilDB.(*gormMysql)
	result := gm.GetUtilDB()
	require.NotNil(t, result)
	require.Equal(t, gm.utilDB, result)
}

func TestGormMysql_GetDB(t *testing.T) {
	dbConf := &DBConfig{
		Username:     "root",
		Password:     password,
		Host:         "127.0.0.1",
		Port:         "3306",
		DBName:       "hahaha_test",
		MaxIdleConns: 10,
		MaxOpenConns: 100,
		DBCharset:    "utf8mb4",
	}

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)
	defer utilDB.Close()

	err = utilDB.CreateDB()
	require.NoError(t, err)
	defer func() {
		dropErr := utilDB.DropDB()
		require.NoError(t, dropErr)
	}()

	db, err := MakeDB(dbConf)
	require.NoError(t, err)
	defer db.Close()

	gm := db.(*gormMysql)
	result := gm.GetDB()
	require.NotNil(t, result)
	require.Equal(t, gm.db, result)
}

func TestGormMysql_CreateDB_Error(t *testing.T) {
	dbConf := &DBConfig{
		Username:     "root",
		Password:     password,
		Host:         "127.0.0.1",
		Port:         "3306",
		DBName:       "hahaha_test",
		MaxIdleConns: 10,
		MaxOpenConns: 100,
		DBCharset:    "utf8mb4",
	}

	// Create a gormMysql instance without initializing utilDB
	gm := &gormMysql{dbConfig: dbConf}

	err := gm.CreateDB()
	require.Error(t, err)
	require.Contains(t, err.Error(), "util db is nil")
}

func TestGormMysql_DropDB_Error(t *testing.T) {
	dbConf := &DBConfig{
		Username:     "root",
		Password:     password,
		Host:         "127.0.0.1",
		Port:         "3306",
		DBName:       "hahaha_test",
		MaxIdleConns: 10,
		MaxOpenConns: 100,
		DBCharset:    "utf8mb4",
	}

	// Create a gormMysql instance without initializing utilDB
	gm := &gormMysql{dbConfig: dbConf}

	err := gm.DropDB()
	require.Error(t, err)
	require.Contains(t, err.Error(), "util db is nil")
}

func TestGormMysql_ClearAllData_Error(t *testing.T) {
	dbConf := &DBConfig{
		Username:     "root",
		Password:     password,
		Host:         "127.0.0.1",
		Port:         "3306",
		DBName:       "production_db",
		MaxIdleConns: 10,
		MaxOpenConns: 100,
		DBCharset:    "utf8mb4",
	}

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)
	defer utilDB.Close()

	err = utilDB.CreateDB()
	require.NoError(t, err)
	defer func() {
		dropErr := utilDB.DropDB()
		require.NoError(t, dropErr)
	}()

	db, err := MakeDB(dbConf)
	require.NoError(t, err)
	defer db.Close()

	gm := db.(*gormMysql)
	err = gm.ClearAllData()
	require.Error(t, err)
	require.Contains(t, err.Error(), "test or dev database")
}

func TestGormMysql_ClearAllData_DBNil(t *testing.T) {
	dbConf := &DBConfig{
		Username:     "root",
		Password:     password,
		Host:         "127.0.0.1",
		Port:         "3306",
		DBName:       "hahaha_test",
		MaxIdleConns: 10,
		MaxOpenConns: 100,
		DBCharset:    "utf8mb4",
	}

	// Create a gormMysql instance without initializing db
	gm := &gormMysql{dbConfig: dbConf}

	err := gm.ClearAllData()
	require.Error(t, err)
	require.Contains(t, err.Error(), "db is nil")
}

func TestGormMysql_Close(t *testing.T) {
	dbConf := &DBConfig{
		Username:     "root",
		Password:     password,
		Host:         "127.0.0.1",
		Port:         "3306",
		DBName:       "hahaha_test",
		MaxIdleConns: 10,
		MaxOpenConns: 100,
		DBCharset:    "utf8mb4",
	}

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)

	err = utilDB.Close()
	require.NoError(t, err)

	// Close again should not error
	err = utilDB.Close()
	require.NoError(t, err)
}

func TestGormMysql_Close_Nil(t *testing.T) {
	gm := &gormMysql{}

	err := gm.Close()
	require.NoError(t, err)
}

func TestGormMysql_buildDSN(t *testing.T) {
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

	gm := &gormMysql{dbConfig: dbConf}

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
		{
			name:     "custom database",
			dbName:   "mydb",
			expected: "testuser:testpass@tcp(localhost:3307)/mydb?charset=utf8&parseTime=True&loc=Local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gm.buildDSN(tt.dbName)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGormMysql_buildDSN_DefaultCharset(t *testing.T) {
	dbConf := &DBConfig{
		Username:     "user",
		Password:     "pass",
		Host:         "host",
		Port:         "3306",
		DBName:       "db",
		MaxIdleConns: 10,
		MaxOpenConns: 100,
		DBCharset:    "", // Empty charset should default to utf8mb4
	}

	gm := &gormMysql{dbConfig: dbConf}

	result := gm.buildDSN("testdb")
	require.Contains(t, result, "charset=utf8mb4")
	require.True(t, strings.HasSuffix(result, "&parseTime=True&loc=Local"))
}
