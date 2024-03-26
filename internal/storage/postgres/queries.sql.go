// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0
// source: queries.sql

package postgresStorage

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type AddLibrariesParams struct {
	UserID     int32
	LibraryKey string
}

const createUser = `-- name: CreateUser :one
  INSERT INTO users (name, plex_url, plex_token, host_url, client_identifier)
  VALUES ($1, $2, $3, $4, $5)
  RETURNING id, name, plex_url, plex_token, host_url, client_identifier, created_at, updated_at
`

type CreateUserParams struct {
	Name             string
	PlexUrl          string
	PlexToken        pgtype.Text
	HostUrl          string
	ClientIdentifier string
}

// CreateUser creates a new user.
func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRow(ctx, createUser,
		arg.Name,
		arg.PlexUrl,
		arg.PlexToken,
		arg.HostUrl,
		arg.ClientIdentifier,
	)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.PlexUrl,
		&i.PlexToken,
		&i.HostUrl,
		&i.ClientIdentifier,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteSelectedLibraries = `-- name: DeleteSelectedLibraries :exec
  DELETE FROM selected_plex_libraries
  WHERE user_id = $1
`

// DeleteSelectedLibraries deletes all selected libraries for a user.
func (q *Queries) DeleteSelectedLibraries(ctx context.Context, userID int32) error {
	_, err := q.db.Exec(ctx, deleteSelectedLibraries, userID)
	return err
}

const deleteUser = `-- name: DeleteUser :one
  DELETE FROM users
  RETURNING id, name, plex_url, plex_token, host_url, client_identifier, created_at, updated_at
`

// DeleteUser deletes the user
func (q *Queries) DeleteUser(ctx context.Context) (User, error) {
	row := q.db.QueryRow(ctx, deleteUser)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.PlexUrl,
		&i.PlexToken,
		&i.HostUrl,
		&i.ClientIdentifier,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getSelectedLibraries = `-- name: GetSelectedLibraries :many
  SELECT id, user_id, library_key, created_at, updated_at FROM selected_plex_libraries
  WHERE user_id = $1
`

// GetSelectedLibraries retrieves all selected libraries for a user.
func (q *Queries) GetSelectedLibraries(ctx context.Context, userID int32) ([]SelectedPlexLibrary, error) {
	rows, err := q.db.Query(ctx, getSelectedLibraries, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SelectedPlexLibrary
	for rows.Next() {
		var i SelectedPlexLibrary
		if err := rows.Scan(
			&i.ID,
			&i.UserID,
			&i.LibraryKey,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getUser = `-- name: GetUser :one
  SELECT id, name, plex_url, plex_token, host_url, client_identifier, created_at, updated_at FROM users 
  LIMIT 1
`

// GetUser retrieves the user.
func (q *Queries) GetUser(ctx context.Context) (User, error) {
	row := q.db.QueryRow(ctx, getUser)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.PlexUrl,
		&i.PlexToken,
		&i.HostUrl,
		&i.ClientIdentifier,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const updateUser = `-- name: UpdateUser :one
  UPDATE users
  SET name = $1,
      plex_url = $2,
      plex_token = $3,
      host_url = $4,
      updated_at = NOW()
  WHERE id = $5
  RETURNING id, name, plex_url, plex_token, host_url, client_identifier, created_at, updated_at
`

type UpdateUserParams struct {
	Name      string
	PlexUrl   string
	PlexToken pgtype.Text
	HostUrl   string
	ID        int32
}

// UpdateUser updates a user's information.
func (q *Queries) UpdateUser(ctx context.Context, arg UpdateUserParams) (User, error) {
	row := q.db.QueryRow(ctx, updateUser,
		arg.Name,
		arg.PlexUrl,
		arg.PlexToken,
		arg.HostUrl,
		arg.ID,
	)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.PlexUrl,
		&i.PlexToken,
		&i.HostUrl,
		&i.ClientIdentifier,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
