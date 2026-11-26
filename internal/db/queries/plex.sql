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

-- name: GetMediaSource :one
SELECT id, url, kind FROM media_sources WHERE id = $1 AND user_id = $2;

-- name: ListAnimeLibraryKeys :many
SELECT library_key FROM anime_libraries WHERE media_source_id = $1;

-- name: ClearAnimeLibraries :exec
DELETE FROM anime_libraries WHERE media_source_id = $1;

-- name: AddAnimeLibraries :exec
INSERT INTO anime_libraries (media_source_id, library_key)
SELECT sqlc.arg(media_source_id), unnest(sqlc.arg(library_keys)::text[]);

-- name: ListAnimeLibrariesForUser :many
SELECT ms.id AS media_source_id, ms.url, ms.kind, al.library_key
FROM anime_libraries al
JOIN media_sources ms ON ms.id = al.media_source_id
WHERE ms.user_id = $1
ORDER BY ms.id, al.library_key;
