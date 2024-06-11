package storage

import (
	"context"
	"errors"
	"time"

	sqlite3Storage "github.com/ReidMason/plex-ani-sync/internal/storage/sqlite3"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

type UserManager interface {
	GetUser() (User, error)
	AddUser(name, plexUrl, hostUrl string, libraryKeys []string) error
	UpdateUser(userId int64, userUpdate UserUpdate) error
	DeleteUser() error
}

type User struct {
	CreatedAt        time.Time
	UpdatedAt        time.Time
	PlexToken        *string
	PlexUrl          string
	HostUrl          string
	Name             string
	ClientIdentifier string
	Libraries        []Library
	Id               int64
}

type Library struct {
	CreatedAt  time.Time
	LibraryKey string
	UserId     int64
}

func (s Sqlite) GetUser() (User, error) {
	ctx := context.Background()

	sqlUserRows, err := s.queries.GetUser(ctx)
	if err != nil {
		return User{}, err
	}

	return sqliteUserRowToUser(sqlUserRows)
}

func (s Sqlite) AddUser(name, plexUrl, hostUrl string, libraryKeys []string) error {
	ctx := context.Background()
	user, err := s.queries.AddUser(ctx, sqlite3Storage.AddUserParams{
		Name:             name,
		PlexUrl:          plexUrl,
		HostUrl:          hostUrl,
		ClientIdentifier: uuid.New().String(),
	})

	if err != nil {
		return err
	}

	return s.setSelectedLibraries(user.ID, libraryKeys)
}

type UserUpdate struct {
	User      *User
	Libraries []string
}

func (s Sqlite) UpdateUser(userId int64, userUpdate UserUpdate) error {
	ctx := context.Background()

	if userUpdate.User != nil {
		userUpdateParams := sqlite3Storage.UpdateUserParams{
			ID:        userId,
			Name:      userUpdate.User.Name,
			PlexUrl:   userUpdate.User.PlexUrl,
			HostUrl:   userUpdate.User.HostUrl,
			PlexToken: stringToSqlNullString(userUpdate.User.PlexToken),
		}

		if err := s.queries.UpdateUser(ctx, userUpdateParams); err != nil {
			return err
		}
	}

	return s.setSelectedLibraries(userId, userUpdate.Libraries)
}

func sqliteUserRowToUser(sqliteUserRows []sqlite3Storage.GetUserRow) (User, error) {
	if len(sqliteUserRows) == 0 {
		return User{}, errors.New("No user found")
	}

	sqliteUserRow := sqliteUserRows[0]

	createdAt, err := parseIso8601Time(sqliteUserRow.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	updatedAt, err := parseIso8601Time(sqliteUserRow.UpdatedAt)
	if err != nil {
		updatedAt = time.Now()
	}

	user := User{
		Id:               sqliteUserRow.ID,
		Name:             sqliteUserRow.Name,
		PlexUrl:          sqliteUserRow.PlexUrl,
		HostUrl:          sqliteUserRow.HostUrl,
		ClientIdentifier: sqliteUserRow.ClientIdentifier,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		Libraries:        []Library{},
	}

	for _, library := range sqliteUserRows {
		user.Libraries = append(user.Libraries, Library{
			LibraryKey: library.LibraryKey,
			UserId:     library.UserID,
			CreatedAt:  createdAt,
		})
	}

	if sqliteUserRow.PlexToken.Valid {
		user.PlexToken = &sqliteUserRow.PlexToken.String
	}

	return user, nil
}

func (s Sqlite) DeleteUser() error {
	ctx := context.Background()
	return s.queries.DeleteUser(ctx)
}

func (s Sqlite) SetupUser(name, plexUrl, hostUrl string) error {
	_, err := s.GetUser()
	if err == nil {
		s.DeleteUser()
	}

	return s.AddUser(name, plexUrl, hostUrl, nil)
}

func (s Sqlite) getSelectedLibraries(userId int64) ([]Library, error) {
	ctx := context.Background()
	libraries, err := s.queries.GetLibraries(ctx, userId)
	if err != nil {
		return nil, err
	}

	result := make([]Library, 0)
	for _, library := range libraries {
		createdAt, err := parseIso8601Time(library.CreatedAt)
		if err != nil {
			createdAt = time.Now()
		}

		result = append(result, Library{
			UserId:     library.UserID,
			LibraryKey: library.LibraryKey,
			CreatedAt:  createdAt,
		})
	}

	return result, nil
}

func (s Sqlite) setSelectedLibraries(userId int64, libraryKeys []string) error {
	ctx := context.Background()

	if libraryKeys == nil {
		libraryKeys = []string{}
	}

	existingLibraries, err := s.queries.GetLibraries(ctx, userId)
	if err != nil {
		return err
	}

	for _, library := range existingLibraries {
		if slices.Contains(libraryKeys, library.LibraryKey) {
			continue
		}

		if err = s.queries.DeleteLibrary(ctx, sqlite3Storage.DeleteLibraryParams{
			UserID:     userId,
			LibraryKey: library.LibraryKey,
		}); err != nil {
			s.log.Error("Failed to delete library", slog.Int64("userId", userId), slog.String("libraryKey", library.LibraryKey), slog.Any("error", err))
		}
	}

	for _, libraryId := range libraryKeys {
		for _, library := range existingLibraries {
			if libraryId == library.LibraryKey {
				continue
			}
		}

		if err = s.queries.AddLibrary(ctx, sqlite3Storage.AddLibraryParams{
			UserID:     userId,
			LibraryKey: libraryId,
		}); err != nil {
			s.log.Error("Failed to add library", slog.Int64("userId", userId), slog.String("libraryKey", libraryId), slog.Any("error", err))
		}
	}

	return nil
}
