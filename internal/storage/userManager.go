package storage

import (
	"context"
	"time"

	"golang.org/x/exp/slog"
)

type UserManager interface {
	GetUser() (User, error)
	// UpdateUser(user User) (User, error)
	// DeleteUser() (User, error)
	// GetSelectedLibraries(userId int64) ([]Library, error)
	// AddSelectedLibraries(userId int64, libraryIds []string) error
	// SetupUser(name, plexUrl, hostUrl string) (User, error)
}

func (s Sqlite) GetUser() (User, error) {
	ctx := context.Background()

	user, err := s.queries.GetUser(ctx)
	if err != nil {
		return User{}, err
	}

	slog.Info("Got user", slog.Any("User:", user))

	createdAt, err := parseIso8601Time(user.CreatedAt)
	if err != nil {
		return User{}, err
	}

	updatedAt, err := parseIso8601Time(user.UpdatedAt)
	if err != nil {
		return User{}, err
	}

	var plexToken *string
	if user.PlexToken.Valid {
		plexToken = &user.PlexToken.String
	}

	return User{
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		PlexToken:        plexToken,
		PlexUrl:          user.PlexUrl,
		HostUrl:          user.HostUrl,
		Name:             user.Name,
		ClientIdentifier: user.ClientIdentifier,
		Id:               user.ID,
	}, nil
}

type User struct {
	CreatedAt        time.Time
	UpdatedAt        time.Time
	PlexToken        *string
	PlexUrl          string
	HostUrl          string
	Name             string
	ClientIdentifier string
	Id               int64
}

// type Library struct {
// 	CreatedAt  time.Time
// 	UpdatedAt  time.Time
// 	LibraryKey string
// 	Id         int32
// 	UserId     int32
// }
//
// func (p Postgres) GetUser() (User, error) {
// 	ctx := context.Background()
// 	user, err := p.queries.GetUser(ctx)
// 	if err != nil {
// 		return User{}, err
// 	}
//
// 	return pgUserToUser(user), nil
// }
//
// func (p Postgres) DeleteUser() (User, error) {
// 	ctx := context.Background()
// 	user, err := p.queries.DeleteUser(ctx)
// 	if err != nil {
// 		return User{}, err
// 	}
//
// 	return pgUserToUser(user), nil
// }
//
// func (p Postgres) CreateUser(name, plexUrl, hostUrl string) (User, error) {
// 	ctx := context.Background()
// 	user, err := p.queries.CreateUser(ctx, postgresStorage.CreateUserParams{
// 		Name:             name,
// 		PlexUrl:          plexUrl,
// 		HostUrl:          hostUrl,
// 		ClientIdentifier: uuid.New().String(),
// 	})
//
// 	if err != nil {
// 		return User{}, err
// 	}
//
// 	return pgUserToUser(user), nil
// }
//
// func (p Postgres) UpdateUser(userUpdate User) (User, error) {
// 	ctx := context.Background()
// 	obj := postgresStorage.UpdateUserParams{
// 		Name:      userUpdate.Name,
// 		PlexUrl:   userUpdate.PlexUrl,
// 		HostUrl:   userUpdate.HostUrl,
// 		PlexToken: stringToPgTypeText(userUpdate.PlexToken),
// 	}
// 	user, err := p.queries.UpdateUser(ctx, obj)
// 	if err != nil {
// 		return User{}, err
// 	}
//
// 	return pgUserToUser(user), nil
// }
//
// func (p Postgres) SetupUser(name, plexUrl, hostUrl string) (User, error) {
// 	if valid, err := ValidateName(name); !valid {
// 		return User{}, errors.New(err)
// 	}
//
// 	if valid, err := ValidatePlexUrl(plexUrl); !valid {
// 		return User{}, errors.New(err)
// 	}
//
// 	if valid, err := ValidateHostUrl(hostUrl); !valid {
// 		return User{}, errors.New(err)
// 	}
//
// 	_, err := p.GetUser()
// 	if err == nil {
// 		p.DeleteUser()
// 	}
//
// 	return p.CreateUser(name, plexUrl, hostUrl)
// }
//
// func (p Postgres) GetSelectedLibraries(userId int32) ([]Library, error) {
// 	ctx := context.Background()
// 	libraries, err := p.queries.GetSelectedLibraries(ctx, userId)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	result := make([]Library, 0)
// 	for _, library := range libraries {
// 		result = append(result, Library{
// 			Id:         library.ID,
// 			UserId:     library.UserID,
// 			LibraryKey: library.LibraryKey,
// 			CreatedAt:  library.CreatedAt.Time,
// 			UpdatedAt:  library.UpdatedAt.Time,
// 		})
// 	}
//
// 	return result, nil
// }
//
// func (p Postgres) AddSelectedLibraries(userId int32, libraryIds []string) error {
// 	ctx := context.Background()
// 	err := p.queries.DeleteSelectedLibraries(ctx, userId)
// 	if err != nil {
// 		slog.Error("error deleting selected libraries", slog.Any("error", err))
// 		return err
// 	}
//
// 	libaries := make([]postgresStorage.AddLibrariesParams, 0)
// 	for _, libraryKey := range libraryIds {
// 		libaries = append(libaries, postgresStorage.AddLibrariesParams{
// 			UserID:     userId,
// 			LibraryKey: libraryKey,
// 		})
// 	}
//
// 	_, err = p.queries.AddLibraries(ctx, libaries)
// 	return err
// }
//
// func ValidateName(name string) (bool, string) {
// 	if name == "" {
// 		return false, "Name is required"
// 	}
//
// 	return true, ""
// }
//
// func ValidateHostUrl(hostUrl string) (bool, string) {
// 	if hostUrl == "" {
// 		return false, "A host URL is required"
// 	}
//
// 	_, err := url.ParseRequestURI(hostUrl)
// 	if err != nil {
// 		return false, "Host URL is invalid"
// 	}
//
// 	return true, ""
// }
//
// func ValidatePlexUrl(plexUrl string) (bool, string) {
// 	if plexUrl == "" {
// 		return false, "Plex URL is required"
// 	}
//
// 	_, err := url.ParseRequestURI(plexUrl)
// 	if err != nil {
// 		return false, "Plex URL is invalid"
// 	}
//
// 	return true, ""
// }
