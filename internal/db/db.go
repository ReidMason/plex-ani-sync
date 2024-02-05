package db

import (
	"context"
	"fmt"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
)

const MIGRATIONS_PATH = "file://db/migrations"

func BuildConnectionString(username, password, host, port, dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", username, password, host, port, dbName)
}

func ConnectToDatabase(connectionString string) (*pgx.Conn, error) {
	return pgx.Connect(context.Background(), connectionString)
}
