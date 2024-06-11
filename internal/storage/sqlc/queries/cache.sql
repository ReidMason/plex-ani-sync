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
