package orm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMysqlMakeDBUtil(t *testing.T) {
	dbConf := loadTestDBConfig(t)
	dbConf.Driver = "mysql"

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)
	defer utilDB.Close()

	err = utilDB.CreateDB()
	require.NoError(t, err)

	err = utilDB.DropDB()
	require.NoError(t, err)
}

func TestMysqlMakeDB(t *testing.T) {
	dbConf := loadTestDBConfig(t)
	dbConf.Driver = "mysql"

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
	dbConf := loadTestDBConfig(t)
	dbConf.Driver = "mysql"

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)
	defer utilDB.Close()

	result := utilDB.GetUtilDB()
	require.NotNil(t, result)
}

func TestMysqlClose(t *testing.T) {
	dbConf := loadTestDBConfig(t)
	dbConf.Driver = "mysql"

	utilDB, err := MakeDBUtil(dbConf)
	require.NoError(t, err)

	err = utilDB.Close()
	require.NoError(t, err)

	// Close again should not error
	err = utilDB.Close()
	require.NoError(t, err)
}
