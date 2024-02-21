package storage

import (
	"time"
)

type Storage interface {
	GetUser() (User, error)
	DeleteUser() (User, error)
	CreateUser(name string) (User, error)
	UpdateUser(user UpdateUserParams) (User, error)
}

type User struct {
	Id               int32
	Name             string
	PlexToken        *string
	ClientIdentifier string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type UpdateUserParams struct {
	Name      string
	PlexToken *string
	Id        int32
}
