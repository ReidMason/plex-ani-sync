-- name: InsertSettingIfMissing :exec
INSERT INTO app_settings (key, value) VALUES ($1, $2)
ON CONFLICT (key) DO NOTHING;

-- name: GetSetting :one
SELECT value FROM app_settings WHERE key = $1;

-- name: UpsertPlexAccount :exec
INSERT INTO plex_accounts (user_id, token_encrypted, username)
VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO UPDATE
SET token_encrypted = EXCLUDED.token_encrypted, username = EXCLUDED.username, updated_at = now();

-- name: GetPlexAccount :one
SELECT token_encrypted, username FROM plex_accounts WHERE user_id = $1;

-- name: InsertMediaSource :one
INSERT INTO media_sources (user_id, url) VALUES ($1, $2)
ON CONFLICT (user_id, url) DO NOTHING
RETURNING id, url;

-- name: ListMediaSources :many
SELECT id, url FROM media_sources WHERE user_id = $1 ORDER BY id;
