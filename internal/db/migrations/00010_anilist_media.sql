-- +goose Up
CREATE TABLE anilist_media (
    anilist_id    bigint PRIMARY KEY,
    episodes      integer,
    title_romaji  text NOT NULL,
    title_english text NOT NULL,
    fetched_at    timestamptz NOT NULL
);

-- +goose Down
DROP TABLE anilist_media;
