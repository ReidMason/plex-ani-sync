package storage

import (
	"context"

	storage "github.com/ReidMason/plex-ani-sync/internal/storage/postgres"
)

type Postgres struct {
	queries *storage.Queries
}

func NewPostgresStorage(queries *storage.Queries) Postgres {
	return Postgres{queries: queries}
}

func (p Postgres) GetUser() (storage.User, error) {
	ctx := context.Background()
	return p.queries.GetUser(ctx)
}

func (p Postgres) DeleteUser() (storage.User, error) {
	ctx := context.Background()
	return p.queries.DeleteUser(ctx)
}

func (p Postgres) CreateUser(newUser storage.User) (storage.User, error) {
	ctx := context.Background()
	return p.queries.CreateUser(ctx, storage.CreateUserParams{
		Name:      newUser.Name,
		PlexToken: newUser.PlexToken,
	})
}

func (p Postgres) UpdateUser(user storage.User) (storage.User, error) {
	ctx := context.Background()
	obj := storage.UpdateUserParams{
		ID:        user.ID,
		Name:      user.Name,
		PlexToken: user.PlexToken,
	}
	return p.queries.UpdateUser(ctx, obj)
}
