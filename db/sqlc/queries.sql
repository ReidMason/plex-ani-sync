-- name: CreateConfig :one
-- statement: |
  INSERT INTO config (app_name, client_identifier, plex_server_url, plex_server_token)
  VALUES ($1, $2, $3, $4)
  RETURNING id, app_name, client_identifier, plex_server_url, plex_server_token;
--
-- This is a SQL query that inserts a new row into the config table and returns the newly inserted row.
-- The query takes four parameters: app_name, client_identifier, plex_server_url, and plex_server_token.
-- The query returns the id, app_name, client_identifier, plex_server_url, and plex_server_token of the newly inserted row.
-- The query is named CreateConfig.
-- The query is defined in the db/sqlc/queries.sql file.
-- The query is used in the db/sqlc/db.go file to insert a new row into the config table.
-- The query is used in the db/sqlc/db_test.go file to test the insert operation.
  
-- name: GetConfig
-- statement: |
--   SELECT id, app_name, client_identifier, plex_server_url, plex_server_token
--   FROM config
--   WHERE id = $1;
--
-- This is a SQL query that retrieves a row from the config table based on the id.
-- The query takes one parameter: id.
-- The query returns the id, app_name, client_identifier, plex_server_url, and plex_server_token of the row that matches the id.
-- The query is named GetConfig.
-- The query is defined in the db/sqlc/queries.sql file.
-- The query is used in the db/sqlc/db.go file to retrieve a row from the config table.
-- The query is used in the db/sqlc/db_test.go file to test the retrieve operation.

-- name: UpdateConfig
-- statement: |
--   UPDATE config
--   SET app_name = $1, client_identifier = $2, plex_server_url = $3, plex_server_token = $4
--   WHERE id = $5
--   RETURNING id, app_name, client_identifier, plex_server_url, plex_server_token;
--
-- This is a SQL query that updates a row in the config table based on the id.
-- The query takes five parameters: app_name, client_identifier, plex_server_url, plex_server_token, and id.
-- The query returns the id, app_name, client_identifier, plex_server_url, and plex_server_token of the updated row.
-- The query is named UpdateConfig.
-- The query is defined in the db/sqlc/queries.sql file.
-- The query is used in the db/sqlc/db.go file to update a row in the config table.
-- The query is used in the db/sqlc/db_test.go file to test the update operation.
  
