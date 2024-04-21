CREATE TABLE Mappings (
    id SERIAL PRIMARY KEY,
    anime_id integer not null,
    anime_episode_start integer not null,
    anime_episode_end integer not null,
    media_id integer not null,
    media_episode_start integer not null,
    media_episode_end integer not null,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(media_id, anime_id)
);

CREATE TABLE Cache (
    id SERIAL PRIMARY KEY,
    key text not null,
    value text not null,
    expires_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(key)
);
