//go:build integration

package orm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

const (
	mysqlTestDBName   = "app_test"
	mysqlTestUser     = "testuser"
	mysqlTestPassword = "testpass"
)

// startMySQLContainer starts a real MySQL container and returns a DBConfig.
// The container is automatically terminated when the test finishes.
func startMySQLContainer(t *testing.T) *DBConfig {
	t.Helper()

	ctx := context.Background()
	container, err := mysql.Run(ctx, "mysql:8.0",
		mysql.WithDatabase(mysqlTestDBName),
		mysql.WithUsername(mysqlTestUser),
		mysql.WithPassword(mysqlTestPassword),
	)
	require.NoError(t, err, "start mysql container")

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("terminate mysql container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	require.NoError(t, err, "get mysql host")

	port, err := container.MappedPort(ctx, "3306/tcp")
	require.NoError(t, err, "get mysql port")

	return &DBConfig{
		Driver:   "mysql",
		Username: mysqlTestUser,
		Password: mysqlTestPassword,
		Host:     host,
		Port:     port.Port(),
		DBName:   mysqlTestDBName,
	}
}

func TestMysqlMakeDBUtil(t *testing.T) {
	dbConf := startMySQLContainer(t)

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)
	defer utilDB.Close()

	err = utilDB.CreateDB()
	require.NoError(t, err)

	err = utilDB.DropDB()
	require.NoError(t, err)
}

func TestMysqlMakeDB(t *testing.T) {
	dbConf := startMySQLContainer(t)

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

func TestMysqlGetUtilDB(t *testing.T) {
	dbConf := startMySQLContainer(t)

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)
	defer utilDB.Close()

	result := utilDB.GetUtilDB()
	require.NotNil(t, result)
}

func TestMysqlClose(t *testing.T) {
	dbConf := startMySQLContainer(t)

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)

	err = utilDB.Close()
	require.NoError(t, err)

	// Close again should not error
	err = utilDB.Close()
	require.NoError(t, err)
}
