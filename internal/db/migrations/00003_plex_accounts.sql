-- +goose Up
CREATE TABLE app_settings (
    key   text PRIMARY KEY,
    value text NOT NULL
);

CREATE TABLE plex_accounts (
    user_id         bigint PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    token_encrypted bytea NOT NULL,
    username        text NOT NULL,
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE plex_accounts;
DROP TABLE app_settings;
