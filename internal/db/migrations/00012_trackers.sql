-- +goose Up
CREATE TABLE trackers (
    user_id    bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       text NOT NULL CHECK (kind IN ('anilist')),
    username   text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, kind)
);

-- +goose Down
DROP TABLE trackers;
