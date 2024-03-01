package storage

import (
	"context"
	"fmt"

	postgresStorage "github.com/ReidMason/plex-ani-sync/internal/storage/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const MIGRATIONS_PATH = "file://db/migrations"

func BuildConnectionString(username, password, host, port, dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", username, password, host, port, dbName)
}

func ConnectToDatabase(connectionString string) (*pgx.Conn, error) {
	return pgx.Connect(context.Background(), connectionString)
}

type Postgres struct {
	queries *postgresStorage.Queries
}

func NewPostgresStorage(queries *postgresStorage.Queries) Postgres {
	return Postgres{queries: queries}
}

func (p Postgres) GetUser() (User, error) {
	ctx := context.Background()
	user, err := p.queries.GetUser(ctx)
	if err != nil {
		return User{}, err
	}

	return pgUserToUser(user), nil
}

func (p Postgres) DeleteUser() (User, error) {
	ctx := context.Background()
	user, err := p.queries.DeleteUser(ctx)
	if err != nil {
		return User{}, err
	}

	return pgUserToUser(user), nil
}

func (p Postgres) CreateUser(name, plexUrl, hostUrl string) (User, error) {
	ctx := context.Background()
	user, err := p.queries.CreateUser(ctx, postgresStorage.CreateUserParams{
		Name:             name,
		PlexUrl:          plexUrl,
		HostUrl:          hostUrl,
		ClientIdentifier: uuid.New().String(),
	})

	if err != nil {
		return User{}, err
	}

	return pgUserToUser(user), nil
}

func (p Postgres) UpdateUser(userUpdate User) (User, error) {
	ctx := context.Background()
	obj := postgresStorage.UpdateUserParams{
		ID:        userUpdate.Id,
		Name:      userUpdate.Name,
		PlexUrl:   userUpdate.PlexUrl,
		HostUrl:   userUpdate.HostUrl,
		PlexToken: stringToPgTypeText(userUpdate.PlexToken),
	}
	user, err := p.queries.UpdateUser(ctx, obj)
	if err != nil {
		return User{}, err
	}

	return pgUserToUser(user), nil
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
