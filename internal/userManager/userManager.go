package userManager

import (
	"context"

	plexAnilistSyncDb "github.com/ReidMason/plex-ani-sync/internal/plexAnilistSyncDb/postgres"
)

type CreateUserParams struct {
	PlexToken string
	Name      string
}

type UserManager struct {
	queries *plexAnilistSyncDb.Queries
}

func New(queries *plexAnilistSyncDb.Queries) UserManager {
	return UserManager{queries: queries}
}

func (um UserManager) CreateUser(newUser plexAnilistSyncDb.User) (plexAnilistSyncDb.User, error) {
	ctx := context.Background()
	return um.queries.CreateUser(ctx, plexAnilistSyncDb.CreateUserParams{
		Name:      newUser.Name,
		PlexToken: newUser.PlexToken,
	})
}

type UpdateUserParams struct {
	Name      string
	PlexToken string
	Id        int32
}

func (um UserManager) UpdateUser(user plexAnilistSyncDb.User) (plexAnilistSyncDb.User, error) {
	ctx := context.Background()
	obj := plexAnilistSyncDb.UpdateUserParams{
		ID:        user.ID,
		Name:      user.Name,
		PlexToken: user.PlexToken,
	}
	return um.queries.UpdateUser(ctx, obj)
}

func (um UserManager) GetUser(id int32) (plexAnilistSyncDb.User, error) {
	ctx := context.Background()
	return um.queries.GetUser(ctx, id)
}
