package storage

import (
	"time"
)

type Storage interface {
	GetSelectedLibraries(userId int32) ([]Library, error)
	AddSelectedLibraries(userId int32, libraryIds []string) error
}

type Library struct {
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LibraryKey string
	Id         int32
	UserId     int32
}
