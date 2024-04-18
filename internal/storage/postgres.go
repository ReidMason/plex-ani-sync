package storage

import (
	"context"
	"fmt"

	postgresStorage "github.com/ReidMason/plex-ani-sync/internal/storage/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const MIGRATIONS_PATH = "file://db/migrations"

func buildConnectionString(username, password, host, port, dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", username, password, host, port, dbName)
}

func connectToDatabase(connectionString string) (*pgx.Conn, error) {
	return pgx.Connect(context.Background(), connectionString)
}

type Postgres struct {
	queries *postgresStorage.Queries
}

func NewPostgresStorage(username, password, host, port, database string) (*Postgres, error) {
	connectionString := buildConnectionString(username, password, host, port, database)
	driver, err := connectToDatabase(connectionString)
	if err != nil {
		return nil, err
	}

	queries := postgresStorage.New(driver)
	return &Postgres{queries: queries}, nil
}

func pgTypeTextToString(text pgtype.Text) *string {
	if text.Valid {
		return &text.String
	}

	return nil
}

func stringToPgTypeText(stringValue *string) pgtype.Text {
	if stringValue == nil {
		return pgtype.Text{
			String: "",
			Valid:  false,
		}
	}

	return pgtype.Text{
		String: *stringValue,
		Valid:  true,
	}
}

func pgUserToUser(user postgresStorage.User) User {
	return User{
		Id:               user.ID,
		Name:             user.Name,
		PlexToken:        pgTypeTextToString(user.PlexToken),
		PlexUrl:          user.PlexUrl,
		HostUrl:          user.HostUrl,
		ClientIdentifier: user.ClientIdentifier,
		CreatedAt:        user.CreatedAt.Time,
		UpdatedAt:        user.UpdatedAt.Time,
	}
}
