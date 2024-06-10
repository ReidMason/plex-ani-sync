-- name: CreateUser :one
-- CreateUser creates a new user.
  INSERT INTO users (name, plex_url, plex_token, host_url, client_identifier)
  VALUES (?, ?, ?, ?, ?)
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
  SET name = ?,
      plex_url = ?,
      plex_token = ?,
      host_url = ?,
      updated_at = datetime('now')
  WHERE id = ?
  RETURNING *;

-- name: GetCache :one
-- GetCache retrieves a cache entry.
  SELECT * FROM cache
  WHERE key = ?
  LIMIT 1;

-- name: SetCache :exec
-- SetCache sets a cache entry.
  INSERT INTO cache (key, value, expires_at)
  VALUES (?, ?, ?)
  ON CONFLICT (key) DO UPDATE
  SET value = excluded.value,
      expires_at = excluded.expires_at,
      updated_at = datetime('now')
  RETURNING *;
