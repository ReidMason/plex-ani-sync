-- name: InsertSettingIfMissing :exec
INSERT INTO app_settings (key, value) VALUES ($1, $2)
ON CONFLICT (key) DO NOTHING;

-- name: GetSetting :one
SELECT value FROM app_settings WHERE key = $1;

-- name: UpsertPlexAccount :exec
INSERT INTO plex_accounts (user_id, token_encrypted)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE
SET token_encrypted = EXCLUDED.token_encrypted, updated_at = now();

-- name: GetPlexAccount :one
SELECT token_encrypted FROM plex_accounts WHERE user_id = $1;

-- name: InsertMediaSource :one
INSERT INTO media_sources (user_id, url, kind) VALUES ($1, $2, $3)
ON CONFLICT (user_id, url) DO NOTHING
RETURNING id, url, kind;

-- name: ListMediaSources :many
SELECT id, url, kind FROM media_sources WHERE user_id = $1 ORDER BY id;

-- name: DeleteMediaSource :execrows
DELETE FROM media_sources WHERE id = $1 AND user_id = $2;
