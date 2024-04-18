package storage

import (
	"context"
	"errors"
	"net/url"
	"time"

	postgresStorage "github.com/ReidMason/plex-ani-sync/internal/storage/postgres"
	"github.com/google/uuid"
)

type UserManager interface {
	GetUser() (User, error)
	UpdateUser(user User) (User, error)
	DeleteUser() (User, error)
	GetSelectedLibraries(userId int32) ([]Library, error)
	AddSelectedLibraries(userId int32, libraryIds []string) error
	SetupUser(name, plexUrl, hostUrl string) (User, error)
}

type User struct {
	CreatedAt        time.Time
	UpdatedAt        time.Time
	PlexToken        *string
	PlexUrl          string
	HostUrl          string
	Name             string
	ClientIdentifier string
	Id               int32
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

func (p Postgres) SetupUser(name, plexUrl, hostUrl string) (User, error) {
	if valid, err := ValidateName(name); !valid {
		return User{}, errors.New(err)
	}

	if valid, err := ValidatePlexUrl(plexUrl); !valid {
		return User{}, errors.New(err)
	}

	if valid, err := ValidateHostUrl(hostUrl); !valid {
		return User{}, errors.New(err)
	}

	_, err := p.GetUser()
	if err == nil {
		p.DeleteUser()
	}

	return p.CreateUser(name, plexUrl, hostUrl)
}

func ValidateName(name string) (bool, string) {
	if name == "" {
		return false, "Name is required"
	}

	return true, ""
}

func ValidateHostUrl(hostUrl string) (bool, string) {
	if hostUrl == "" {
		return false, "A host URL is required"
	}

	_, err := url.ParseRequestURI(hostUrl)
	if err != nil {
		return false, "Host URL is invalid"
	}

	return true, ""
}

func ValidatePlexUrl(plexUrl string) (bool, string) {
	if plexUrl == "" {
		return false, "Plex URL is required"
	}

	_, err := url.ParseRequestURI(plexUrl)
	if err != nil {
		return false, "Plex URL is invalid"
	}

	return true, ""
}
