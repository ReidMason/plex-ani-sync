-- +goose Up
ALTER TABLE anilist_media ADD COLUMN missing boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE anilist_media DROP COLUMN missing;
