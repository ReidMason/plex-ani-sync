-- name: CreateUser :one
-- CreateUser creates a new user.
  INSERT INTO users (name, plex_token)
  VALUES ($1, $2)
  RETURNING *;

-- name: GetUser :one
-- GetUser retrieves the user.
  SELECT * FROM users 
  LIMIT 1;

  -- name: UpdateUser :one
-- UpdateUser updates a user's information.
  UPDATE users
  SET name = $1,
      plex_token = $2,
      updated_at = NOW()
  WHERE id = $3
  RETURNING *;
