package db

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

const MIGRATIONS_PATH = "file://db/migrations"

func BuildConnectionString(username, password, host, port, dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", username, password, host, port, dbName)
}

func OpenDbConnection(connectionString string) (*sql.DB, error) {
	return sql.Open("postgres", connectionString)
}

func CreateDbDriver(db *sql.DB) (database.Driver, error) {
	return postgres.WithInstance(db, &postgres.Config{})
}

func Migrate(driver database.Driver) error {
	m, err := migrate.NewWithDatabaseInstance(
		MIGRATIONS_PATH,
		"postgres", driver)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && err.Error() != "no change" {
		return err
	}

	return nil
}
