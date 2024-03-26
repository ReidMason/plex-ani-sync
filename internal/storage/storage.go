package storage

import (
	"time"
)

type Storage interface {
	GetUser() (User, error)
	DeleteUser() (User, error)
	CreateUser(name, plexUrl, hostUrl string) (User, error)
	UpdateUser(user User) (User, error)
	GetSelectedLibraries(userId int32) ([]Library, error)
	AddSelectedLibraries(userId int32, libraryIds []string) error
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

type Library struct {
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LibraryKey string
	Id         int32
	UserId     int32
}
