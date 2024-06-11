-- name: AddUser :one
-- Creates a new user.
  INSERT INTO users (name, plex_url, plex_token, host_url, client_identifier)
  VALUES (?, ?, ?, ?, ?)
  RETURNING *;

-- name: GetUser :many
-- GetUser retrieves the user.
SELECT * FROM (
    SELECT * FROM users
    LIMIT 1
) as usr
LEFT JOIN plex_user_libraries as spl ON spl.user_id = usr.id;

-- name: DeleteUser :exec
-- DeleteUser deletes the user
  DELETE FROM users;

-- name: UpdateUser :exec
-- Update a user
  UPDATE users
  SET name = ?,
      plex_url = ?,
      plex_token = ?,
      host_url = ?,
      updated_at = datetime('now')
  WHERE id = ?;

-- name: GetLibraries :many
-- Retrieve all libraries for a user
  SELECT * FROM plex_user_libraries
  WHERE user_id = ?;

-- name: DeleteLibrary :exec
-- Delete library from user
  DELETE FROM plex_user_libraries
  WHERE user_id = ? AND library_key = ?;

-- name: AddLibrary :exec
-- AddLibraries adds selected library for a user.
  INSERT INTO plex_user_libraries (user_id, library_key)
  VALUES (?, ?);
