// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0
// source: queries.sql

package storage

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createUser = `-- name: CreateUser :one
  INSERT INTO users (name, plex_token)
  VALUES ($1, $2)
  RETURNING id, name, plex_token, created_at, updated_at
`

type CreateUserParams struct {
	Name      string
	PlexToken pgtype.Text
}

// CreateUser creates a new user.
func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRow(ctx, createUser, arg.Name, arg.PlexToken)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.PlexToken,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getUser = `-- name: GetUser :one
  SELECT id, name, plex_token, created_at, updated_at FROM users 
  LIMIT 1
`

// GetUser retrieves the user.
func (q *Queries) GetUser(ctx context.Context) (User, error) {
	row := q.db.QueryRow(ctx, getUser)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.PlexToken,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const updateUser = `-- name: UpdateUser :one
  UPDATE users
  SET name = $1,
      plex_token = $2,
      updated_at = NOW()
  WHERE id = $3
  RETURNING id, name, plex_token, created_at, updated_at
`

type UpdateUserParams struct {
	Name      string
	PlexToken pgtype.Text
	ID        int32
}

// UpdateUser updates a user's information.
func (q *Queries) UpdateUser(ctx context.Context, arg UpdateUserParams) (User, error) {
	row := q.db.QueryRow(ctx, updateUser, arg.Name, arg.PlexToken, arg.ID)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.PlexToken,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
