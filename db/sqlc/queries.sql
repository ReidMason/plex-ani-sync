-- name: CreateUser :one
-- CreateUser creates a new user.
  INSERT INTO users (name, plex_url, plex_token, host_url, client_identifier)
  VALUES ($1, $2, $3, $4, $5)
  RETURNING *;

-- name: GetUser :one
-- GetUser retrieves the user.
  SELECT * FROM users 
  LIMIT 1;

-- name: DeleteUser :one
-- DeleteUser deletes the user
  DELETE FROM users
  RETURNING *;

-- name: UpdateUser :one
-- UpdateUser updates a user's information.
  UPDATE users
  SET name = $1,
      plex_url = $2,
      plex_token = $3,
      host_url = $4,
      updated_at = NOW()
  WHERE id = $5
  RETURNING *;

-- name: GetSelectedLibraries :many
-- GetSelectedLibraries retrieves all selected libraries for a user.
  SELECT * FROM selected_plex_libraries
  WHERE user_id = $1;

-- name: DeleteSelectedLibraries :exec
-- DeleteSelectedLibraries deletes all selected libraries for a user.
  DELETE FROM selected_plex_libraries
  WHERE user_id = $1;

-- name: AddLibraries :copyfrom
-- AddLibraries adds selected libraries for a user.
  INSERT INTO selected_plex_libraries (user_id, library_key)
  VALUES ($1, $2);

-- name: GetCache :one
-- GetCache retrieves a cache entry.
  SELECT * FROM cache
  WHERE key = $1
  LIMIT 1;

-- name: SetCache :exec
-- SetCache sets a cache entry.
  INSERT INTO cache (key, value, expires_at)
  VALUES ($1, $2, $3)
  ON CONFLICT (key) DO UPDATE
  SET value = $2,
      expires_at = $3,
      updated_at = NOW()
  RETURNING *;

-- name: DeleteCache :exec
-- DeleteCache deletes a cache entry.
  DELETE FROM cache
  WHERE key = $1
  RETURNING *;
