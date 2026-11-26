-- +goose Up
CREATE TABLE mapping_origins (
    name text PRIMARY KEY
);

INSERT INTO mapping_origins (name) VALUES ('dataset'), ('custom');

CREATE TABLE mappings (
    id                 bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tvdb_id            bigint NOT NULL,
    tvdb_season        integer NOT NULL,
    first_episode      integer NOT NULL,
    last_episode       integer,
    anilist_id         bigint NOT NULL,
    entry_first_episode integer NOT NULL,
    entry_last_episode integer,
    anidb_id           bigint,
    origin             text NOT NULL REFERENCES mapping_origins (name)
);

CREATE INDEX mappings_tvdb_id_idx ON mappings (tvdb_id);

-- +goose Down
DROP TABLE mappings;
DROP TABLE mapping_origins;
