-- +goose Up
ALTER TABLE media_sources ADD COLUMN kind text NOT NULL DEFAULT 'plex';
ALTER TABLE media_sources ALTER COLUMN kind DROP DEFAULT;

-- +goose Down
ALTER TABLE media_sources DROP COLUMN kind;
