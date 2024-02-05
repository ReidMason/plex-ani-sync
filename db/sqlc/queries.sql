-- name: CreateConfig :one
-- This is a SQL query that inserts a new row into the config table and returns the newly inserted row.
  INSERT INTO config (app_name, client_identifier, plex_server_url, plex_server_token)
  VALUES ($1, $2, $3, $4)
  RETURNING id, app_name, client_identifier, plex_server_url, plex_server_token;
