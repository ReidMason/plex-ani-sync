-- +goose Up
ALTER TABLE plex_accounts DROP COLUMN username;

-- +goose Down
ALTER TABLE plex_accounts ADD COLUMN username text NOT NULL DEFAULT '';
ALTER TABLE plex_accounts ALTER COLUMN username DROP DEFAULT;
