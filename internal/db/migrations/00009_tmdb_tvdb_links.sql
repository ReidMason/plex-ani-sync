-- +goose Up
CREATE TABLE tmdb_tvdb_links (
    tmdb_id bigint PRIMARY KEY,
    tvdb_id bigint NOT NULL
);

-- +goose Down
DROP TABLE tmdb_tvdb_links;
