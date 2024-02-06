package users

import (
	"context"
	"log"

	"github.com/ReidMason/plex-ani-sync/internal/db"
	plexAnilistSyncDb "github.com/ReidMason/plex-ani-sync/internal/plexAnilistSyncDb/postgres"
)

type CreateUserParams struct {
	Name      string
	PlexToken string
}

func CreateUser(newUser CreateUserParams) {
	connectionString := db.BuildConnectionString("testuser", "testpass", "localhost", "5432", "plexAnilistSync")
	driver, err := db.ConnectToDatabase(connectionString)
	if err != nil {
		panic(err)
	}

	queries := plexAnilistSyncDb.New(driver)
	ctx := context.Background()
	user, err := queries.CreateUser(ctx, plexAnilistSyncDb.CreateUserParams{
		Name:      newUser.Name,
		PlexToken: db.StringToPgTypeText(newUser.PlexToken),
	})
	if err != nil {
		panic(err)
	}

	log.Println(user)
}

type UpdateUserParams struct {
	Name      string
	PlexToken string
	Id        int32
}

func UpdateUser(user UpdateUserParams) {
	connectionString := db.BuildConnectionString("testuser", "testpass", "localhost", "5432", "plexAnilistSync")
	driver, err := db.ConnectToDatabase(connectionString)
	if err != nil {
		panic(err)
	}

	queries := plexAnilistSyncDb.New(driver)
	ctx := context.Background()
	updatedUser, err := queries.UpdateUser(ctx, plexAnilistSyncDb.UpdateUserParams{
		ID:        user.Id,
		Name:      user.Name,
		PlexToken: db.StringToPgTypeText(user.PlexToken),
	})
	if err != nil {
		panic(err)
	}

	log.Println(updatedUser)
}
