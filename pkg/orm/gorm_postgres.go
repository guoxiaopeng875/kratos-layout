package orm

import (
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type postgresDialect struct{}

func (p *postgresDialect) buildDSN(config *DBConfig, dbName string) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Host,
		config.Port,
		config.Username,
		config.Password,
		dbName)
}

func (p *postgresDialect) dialector(dsn string) gorm.Dialector {
	return postgres.Open(dsn)
}

func (p *postgresDialect) createDBSQL(dbName string) string {
	quoted := p.quoteIdentifier(dbName)
	return fmt.Sprintf("CREATE DATABASE %s ENCODING 'UTF8'", quoted)
}

func (p *postgresDialect) dropDBSQL(dbName string) string {
	quoted := p.quoteIdentifier(dbName)
	return fmt.Sprintf("DROP DATABASE IF EXISTS %s;", quoted)
}

func (p *postgresDialect) listTablesSQL() string {
	return "SELECT tablename FROM pg_tables WHERE schemaname = 'public';"
}

func (p *postgresDialect) clearTableSQL(table string) string {
	return fmt.Sprintf("TRUNCATE TABLE %s CASCADE", p.quoteIdentifier(table))
}

func (p *postgresDialect) quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (p *postgresDialect) utilDBName() string {
	return DriverPostgres
}
