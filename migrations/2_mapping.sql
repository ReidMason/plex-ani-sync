CREATE TABLE list_provider (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE
);

INSERT INTO list_provider (name)
values ('Anilist');

CREATE TABLE anime (
  anime_id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  episodes INTEGER
);

CREATE TABLE mapping (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  list_provider_id INT NOT NULL,
  plex_id TEXT NOT NULL,
  plex_series_id TEXT NOT NULL,
  plex_episode_start INT NOT NULL,
  season_length INT NOT NULL,
  anime_list_id TEXT NOT NULL,
  episode_start INT NOT NULL,
  enabled BOOLEAN NOT NULL,
  ignored BOOLEAN NOT NULL,
  FOREIGN KEY(list_provider_id) REFERENCES list_provider(id),
  FOREIGN KEY(anime_list_id) REFERENCES anime(anime_id)
);


