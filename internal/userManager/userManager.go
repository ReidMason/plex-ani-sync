package userManager

import (
	"errors"
	"net/url"

	"github.com/ReidMason/plex-ani-sync/internal/storage"
)

type UserManager struct {
	store storage.Storage
}

func NewUserManager(store storage.Storage) UserManager {
	return UserManager{store: store}
}

func (u UserManager) GetUser() (storage.User, error) {
	return u.store.GetUser()
}

func (u UserManager) UpdateUser(user storage.User) (storage.User, error) {
	return u.store.UpdateUser(user)
}

func (u UserManager) DeleteUser() (storage.User, error) {
	return u.store.DeleteUser()
}

func (u UserManager) GetSelectedLibraries(userId int32) ([]storage.Library, error) {
	return u.store.GetSelectedLibraries(userId)
}

func (u UserManager) AddSelectedLibraries(userId int32, libraryIds []string) error {
	return u.store.AddSelectedLibraries(userId, libraryIds)
}

func (u UserManager) SetupUser(name, plexUrl, hostUrl string) (storage.User, error) {
	if valid, err := ValidateName(name); !valid {
		return storage.User{}, errors.New(err)
	}

	if valid, err := ValidatePlexUrl(plexUrl); !valid {
		return storage.User{}, errors.New(err)
	}

	if valid, err := ValidateHostUrl(hostUrl); !valid {
		return storage.User{}, errors.New(err)
	}

	_, err := u.GetUser()
	if err == nil {
		u.DeleteUser()
	}

	return u.store.CreateUser(name, plexUrl, hostUrl)
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
