package storage

import (
	storage "github.com/ReidMason/plex-ani-sync/internal/storage/postgres"
)

type Storage interface {
	GetUser() (storage.User, error)
	CreateUser(newUser storage.User) (storage.User, error)
	UpdateUser(user storage.User) (storage.User, error)
}

type UpdateUserParams struct {
	Name      string
	PlexToken string
	Id        int32
}
