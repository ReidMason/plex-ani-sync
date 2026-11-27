-- +goose Up
ALTER TABLE trackers ADD COLUMN token_encrypted bytea;

-- +goose Down
ALTER TABLE trackers DROP COLUMN token_encrypted;
