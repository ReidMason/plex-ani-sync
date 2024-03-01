package storage

import (
	"time"
)

type Storage interface {
	GetUser() (User, error)
	DeleteUser() (User, error)
	CreateUser(name, plexUrl string) (User, error)
	UpdateUser(user User) (User, error)
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
