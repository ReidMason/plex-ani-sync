CREATE TABLE config (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  plex_url TEXT NOT NULL,
  plex_token TEXT,
  anilist_token TEXT
);

INSERT INTO config (plex_url, plex_token, anilist_token)
values ("http://localhost:32400", null, null);

CREATE TABLE anime_search_cache (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  search_term TEXT NOT NULL UNIQUE,
  data TEXT NOT NULL 
);

CREATE TABLE anime_cache (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  anime_id TEXT NOT NULL UNIQUE,
  data TEXT NOT NULL
);

