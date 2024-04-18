package storage

import (
	"context"
	"fmt"
	"log/slog"

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

func (p Postgres) GetSelectedLibraries(userId int32) ([]Library, error) {
	ctx := context.Background()
	libraries, err := p.queries.GetSelectedLibraries(ctx, userId)
	if err != nil {
		return nil, err
	}

	result := make([]Library, 0)
	for _, library := range libraries {
		result = append(result, Library{
			Id:         library.ID,
			UserId:     library.UserID,
			LibraryKey: library.LibraryKey,
			CreatedAt:  library.CreatedAt.Time,
			UpdatedAt:  library.UpdatedAt.Time,
		})
	}

	return result, nil
}

func (p Postgres) AddSelectedLibraries(userId int32, libraryIds []string) error {
	ctx := context.Background()
	err := p.queries.DeleteSelectedLibraries(ctx, userId)
	if err != nil {
		slog.Error("error deleting selected libraries", slog.Any("error", err))
		return err
	}

	libaries := make([]postgresStorage.AddLibrariesParams, 0)
	for _, libraryKey := range libraryIds {
		libaries = append(libaries, postgresStorage.AddLibrariesParams{
			UserID:     userId,
			LibraryKey: libraryKey,
		})
	}

	_, err = p.queries.AddLibraries(ctx, libaries)
	return err
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
