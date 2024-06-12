-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE TABLE users (
  id INTEGER NOT NULL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  plex_url TEXT NOT NULL,
  plex_token TEXT,
  host_url TEXT NOT NULL,
  client_identifier TEXT NOT NULL,
  animelist_api_client_id TEXT,
  animelist_secret TEXT,
  animelist_token TEXT,
  animelist_refresh_token TEXT,
  created_at TEXT NOT NULL DEFAULT(datetime('now')),
  updated_at TEXT NOT NULL DEFAULT(datetime('now'))
);

CREATE TABLE plex_user_libraries (
  user_id INTEGER NOT NULL REFERENCES users(id),
  library_key TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT(datetime('now')),
  PRIMARY KEY (user_id, library_key)
);
 
CREATE TABLE mappings (
    id INTEGER NOT NULL PRIMARY KEY,
    anime_id INTEGER NOT NULL,
    anime_episode_start INTEGER NOT NULL,
    anime_episode_end INTEGER NOT NULL,
    media_id INTEGER NOT NULL,
    media_episode_start INTEGER NOT NULL,
    media_episode_end INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT(datetime('now')),
    updated_at TEXT NOT NULL DEFAULT(datetime('now')),
    UNIQUE(media_id, anime_id)
);

CREATE TABLE cache (
    id INTEGER NOT NULL PRIMARY KEY,
    key text NOT NULL UNIQUE,
    value text NOT NULL,
    updated_at TEXT NOT NULL DEFAULT(datetime('now')),
    expires_at TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS selected_plex_libraries;
DROP TABLE IF EXISTS mappings;
DROP TABLE IF EXISTS cache;
-- +goose StatementEnd
