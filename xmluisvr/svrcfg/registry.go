package cfgldr

import (
	"fmt"
)

type DatabaseType string

const (
	SQLite3Database  DatabaseType = "sqlite3"
	PostgresDatabase DatabaseType = "postgres"
	DuckDBDatabase   DatabaseType = "duckdb"
	MySQLDatabase    DatabaseType = "mysql"
	MariaDBDatabase  DatabaseType = "mariadb"
)

var databaseConfigs = make([]DatabaseConfig, 0)

func registerDatabaseConfig(dbc DatabaseConfig) {
	databaseConfigs = append(databaseConfigs, dbc)
}
func GetDatabaseConfig(dt DatabaseType) (dbc DatabaseConfig, err error) {
	for _, dbc = range databaseConfigs {
		if dbc.DatabaseType() != dt {
			continue
		}
		goto end
	}
	err = fmt.Errorf("database type '%s' not supported", dt)
	dbc = nil
end:
	return dbc, err
}
