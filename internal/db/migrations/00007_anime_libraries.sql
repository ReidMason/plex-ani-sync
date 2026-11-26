-- +goose Up
CREATE TABLE anime_libraries (
    media_source_id bigint NOT NULL REFERENCES media_sources (id) ON DELETE CASCADE,
    library_key    text NOT NULL,
    PRIMARY KEY (media_source_id, library_key)
);

-- +goose Down
DROP TABLE anime_libraries;
