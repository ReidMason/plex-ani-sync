package users

import (
	"context"

	"github.com/ReidMason/plex-ani-sync/internal/db"
	plexAnilistSyncDb "github.com/ReidMason/plex-ani-sync/internal/plexAnilistSyncDb/postgres"
)

type CreateUserParams struct {
	Name      string
	PlexToken string
}

func CreateUser(newUser CreateUserParams) (plexAnilistSyncDb.User, error) {
	var user plexAnilistSyncDb.User

	connectionString := db.BuildConnectionString("testuser", "testpass", "localhost", "5432", "plexAnilistSync")
	driver, err := db.ConnectToDatabase(connectionString)
	if err != nil {
		return user, err
	}

	queries := plexAnilistSyncDb.New(driver)
	ctx := context.Background()
	return queries.CreateUser(ctx, plexAnilistSyncDb.CreateUserParams{
		Name:      newUser.Name,
		PlexToken: db.StringToPgTypeText(newUser.PlexToken),
	})
}

type UpdateUserParams struct {
	Name      string
	PlexToken string
	Id        int32
}

func UpdateUser(user UpdateUserParams) (plexAnilistSyncDb.User, error) {
	var updatedUser plexAnilistSyncDb.User

	connectionString := db.BuildConnectionString("testuser", "testpass", "localhost", "5432", "plexAnilistSync")
	driver, err := db.ConnectToDatabase(connectionString)
	if err != nil {
		return updatedUser, err
	}

	queries := plexAnilistSyncDb.New(driver)
	ctx := context.Background()
	return queries.UpdateUser(ctx, plexAnilistSyncDb.UpdateUserParams{
		ID:        user.Id,
		Name:      user.Name,
		PlexToken: db.StringToPgTypeText(user.PlexToken),
	})
}
