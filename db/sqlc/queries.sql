-- name: CreateUser :one
-- CreateUser creates a new user.
  INSERT INTO users (name, plex_token)
  VALUES ($1, $2)
  RETURNING *;

-- name: GetUser :one
-- GetUser retrieves a user by ID.
  SELECT * FROM users WHERE id = $1;

  -- name: UpdateUser :one
-- UpdateUser updates a user's information.
  UPDATE users
  SET name = COALESCE($1, name),
      plex_token = COALESCE($2, plex_token),
      updated_at = NOW()
  WHERE id = $3
  RETURNING *;
