// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0
// source: queries.sql

package db

import (
	"context"
)

const createConfig = `-- name: CreateConfig :one
  INSERT INTO config (app_name, client_identifier, plex_server_url, plex_server_token)
  VALUES ($1, $2, $3, $4)
  RETURNING id, app_name, client_identifier, plex_server_url, plex_server_token
`

type CreateConfigParams struct {
	AppName          string
	ClientIdentifier string
	PlexServerUrl    string
	PlexServerToken  string
}

// statement: |
func (q *Queries) CreateConfig(ctx context.Context, arg CreateConfigParams) (Config, error) {
	row := q.db.QueryRow(ctx, createConfig,
		arg.AppName,
		arg.ClientIdentifier,
		arg.PlexServerUrl,
		arg.PlexServerToken,
	)
	var i Config
	err := row.Scan(
		&i.ID,
		&i.AppName,
		&i.ClientIdentifier,
		&i.PlexServerUrl,
		&i.PlexServerToken,
	)
	return i, err
}