# Plex ↔ AniList sync. Loads `.env` via the Go app (godotenv).

default:
    @just --list

run-api:
    go run ./cmd/server

test:
    go test ./...

# Snapshot AniList list to JSON (full sync runs). Needs ANILIST_TOKEN in .env.
# `just save-anilist` · `just save-anilist path=fixtures/my-list.json`
save-anilist path="data/anilist-list.json":
    ANILIST_SAVE_LIST="{{path}}" go run ./cmd/server

# Sync using a saved list (no AniList API). Token not required.
# `just run-mock` · `just run-mock path=fixtures/my-list.json`
run-mock path="data/anilist-list.json":
    ANILIST_MOCK=1 ANILIST_MOCK_FILE="{{path}}" go run ./cmd/server

# Built-in Gate/TvDB-295222 AniList fixture. Plex mock only if PLEX_* unset in .env.
run-mock-builtin:
    ANILIST_MOCK=1 go run ./cmd/server
