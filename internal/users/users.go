package users

import (
	"context"
	"log"

	"github.com/ReidMason/plex-ani-sync/internal/db"
	plexAnilistSyncDb "github.com/ReidMason/plex-ani-sync/internal/plexAnilistSyncDb/postgres"
)

type CreateUserParams struct {
	PlexToken string
	Name      string
}

func CreateUser(newUser plexAnilistSyncDb.User) (plexAnilistSyncDb.User, error) {
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
		PlexToken: newUser.PlexToken,
	})
}

type UpdateUserParams struct {
	Name      string
	PlexToken string
	Id        int32
}

func UpdateUser(user plexAnilistSyncDb.User) (plexAnilistSyncDb.User, error) {
	var updatedUser plexAnilistSyncDb.User

	connectionString := db.BuildConnectionString("testuser", "testpass", "localhost", "5432", "plexAnilistSync")
	driver, err := db.ConnectToDatabase(connectionString)
	if err != nil {
		return updatedUser, err
	}

	queries := plexAnilistSyncDb.New(driver)
	ctx := context.Background()
	obj := plexAnilistSyncDb.UpdateUserParams{
		ID:        user.ID,
		Name:      user.Name,
		PlexToken: user.PlexToken,
	}
	log.Println(obj)
	return queries.UpdateUser(ctx, obj)
}

func GetUser(id int32) (plexAnilistSyncDb.User, error) {
	var user plexAnilistSyncDb.User

	connectionString := db.BuildConnectionString("testuser", "testpass", "localhost", "5432", "plexAnilistSync")
	driver, err := db.ConnectToDatabase(connectionString)
	if err != nil {
		return user, err
	}

	queries := plexAnilistSyncDb.New(driver)
	ctx := context.Background()
	return queries.GetUser(ctx, id)
}
