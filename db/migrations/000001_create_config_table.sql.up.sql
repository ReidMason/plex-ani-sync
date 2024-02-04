CREATE TABLE config (
  id SERIAL PRIMARY KEY,
  app_name TEXT NOT NULL,
  client_identifier TEXT NOT NULL,
  plex_server_url TEXT NOT NULL,
  plex_server_token TEXT NOT NULL
);
