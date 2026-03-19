//go:build integration

package orm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	pgTestDBName   = "app_test"
	pgTestUser     = "testuser"
	pgTestPassword = "testpass"
)

// startPostgresContainer starts a real PostgreSQL container and returns a DBConfig.
// The container is automatically terminated when the test finishes.
func startPostgresContainer(t *testing.T) *DBConfig {
	t.Helper()

	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase(pgTestDBName),
		postgres.WithUsername(pgTestUser),
		postgres.WithPassword(pgTestPassword),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "start postgres container")

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	require.NoError(t, err, "get postgres host")

	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err, "get postgres port")

	return &DBConfig{
		Driver:   "postgres",
		Username: pgTestUser,
		Password: pgTestPassword,
		Host:     host,
		Port:     port.Port(),
		DBName:   pgTestDBName,
	}
}

func TestPostgresMakeDBUtil(t *testing.T) {
	dbConf := startPostgresContainer(t)

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)
	defer utilDB.Close()

	err = utilDB.CreateDB()
	require.NoError(t, err)

	err = utilDB.DropDB()
	require.NoError(t, err)
}

func TestPostgresMakeDB(t *testing.T) {
	dbConf := startPostgresContainer(t)

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

func TestPostgresGetUtilDB(t *testing.T) {
	dbConf := startPostgresContainer(t)

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)
	defer utilDB.Close()

	result := utilDB.GetUtilDB()
	require.NotNil(t, result)
}

func TestPostgresClose(t *testing.T) {
	dbConf := startPostgresContainer(t)

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)

	err = utilDB.Close()
	require.NoError(t, err)

	// Close again should not error
	err = utilDB.Close()
	require.NoError(t, err)
}
