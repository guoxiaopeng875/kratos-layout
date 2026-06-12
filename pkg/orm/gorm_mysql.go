package orm

import (
	"fmt"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type mysqlDialect struct{}

const mysqlUtilDB = "information_schema"

func (m *mysqlDialect) buildDSN(config *DBConfig, dbName string) string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		dbName,
		config.getCharset())
	if config.MultiStatements {
		dsn += "&multiStatements=true"
	}
	return dsn
}

func (m *mysqlDialect) dialector(dsn string) gorm.Dialector {
	return mysql.Open(dsn)
}

func (m *mysqlDialect) createDBSQL(dbName string) string {
	quoted := m.quoteIdentifier(dbName)
	return fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci;", quoted)
}

func (m *mysqlDialect) dropDBSQL(dbName string) string {
	quoted := m.quoteIdentifier(dbName)
	return fmt.Sprintf("DROP DATABASE IF EXISTS %s;", quoted)
}

func (m *mysqlDialect) listTablesSQL() string {
	return "SHOW TABLES;"
}

func (m *mysqlDialect) clearTableSQL(table string) string {
	return fmt.Sprintf("DELETE FROM %s", m.quoteIdentifier(table))
}

func (m *mysqlDialect) quoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func (m *mysqlDialect) utilDBName() string {
	return mysqlUtilDB
}
