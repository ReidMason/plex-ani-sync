# Plex Ani Sync

Syncs the anime you watch in Plex to the sites where you track it. Go backend serving an embedded Vite/React SPA, Postgres for storage. Sign-in goes through an OIDC provider such as Authentik.

## Run

Set up the OIDC provider first (see [Authentik setup](#authentik-setup)). Copy `.env.example` to `.env` (git-ignored) and fill it in. Generate the encryption key once and keep it (losing it means everyone re-signs in to Plex):

```sh
openssl rand -base64 32
# without openssl:
head -c 32 /dev/urandom | base64
# PowerShell:
[Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }))
```

```sh
docker compose up --build
```

Open <http://localhost:8080>.

## Configuration

| Variable | Description |
| --- | --- |
| `DATABASE_URL` | Postgres connection string (required; set by compose) |
| `PUBLIC_BASE_URL` | Externally visible base URL (required). The OIDC redirect URI is `<PUBLIC_BASE_URL>/auth/callback` |
| `OIDC_ISSUER_URL` | OIDC issuer URL; discovery runs at startup and the app exits if it fails (required). It must be reachable from inside the container, so not `localhost` |
| `OIDC_CLIENT_ID` | OIDC client ID (required) |
| `OIDC_CLIENT_SECRET` | OIDC client secret (required) |
| `APP_ENCRYPTION_KEY` | Base64 of 32 random bytes; encrypts Plex tokens at rest (required). Losing it means everyone re-signs in to Plex |
| `LISTEN_ADDR` | Listen address, default `:8080` |
| `ANILIST_CLIENT_ID` | AniList API client ID. Optional, but must be set together with `ANILIST_CLIENT_SECRET`; enables authorising Trackers on AniList. See [AniList setup](#anilist-setup) |
| `ANILIST_CLIENT_SECRET` | AniList API client secret. Optional, but must be set together with `ANILIST_CLIENT_ID` |
| `ANILIST_REQUESTS_PER_MINUTE` | Cap on requests to AniList, default `30` (AniList documents 90 but has lowered it before) |

## Authentik setup

1. **Applications > Providers > Create > OAuth2/OpenID Provider.** Client type `Confidential`, redirect URI strict match on `<PUBLIC_BASE_URL>/auth/callback`.
2. **Applications > Applications > New Application > With existing provider.** Set a name and slug (e.g. `plex-ani-sync`).
3. Copy the client ID and secret from the provider into `.env`. The issuer is normally `https://<authentik-host>/application/o/<slug>/`.

## AniList setup

Optional. Without it Users can still link an AniList username and read their Sync Plan, but cannot authorise the app to change their list.

1. Open <https://anilist.co/settings/developer> and **Create New Client**.
2. Set the redirect URL to `<PUBLIC_BASE_URL>/auth/anilist/callback`.
3. Copy the client ID and secret into `ANILIST_CLIENT_ID` and `ANILIST_CLIENT_SECRET` in `.env`. Setting only one makes startup fail.

## Development

```sh
docker compose up -d postgres
cd web && pnpm install && pnpm build
go test ./...   # needs Docker (Postgres via testcontainers)
```

Run the server against the compose Postgres:

```sh
export DATABASE_URL=postgres://plex-ani-sync:plex-ani-sync@localhost:5432/plex-ani-sync?sslmode=disable
export PUBLIC_BASE_URL=http://localhost:8080
export OIDC_ISSUER_URL=... OIDC_CLIENT_ID=... OIDC_CLIENT_SECRET=... APP_ENCRYPTION_KEY=...
go run ./cmd/plex-ani-sync
cd web && pnpm dev   # Vite on :5173, proxies /api and /auth to :8080
```

After editing migrations or queries, run `sqlc generate`.

## Credits

Show-to-AniList mappings come from [Fribb/anime-lists](https://github.com/Fribb/anime-lists). Episode-level mappings for Seasons folded into one Entry come from [Anime-Lists](https://github.com/Anime-Lists/anime-lists). Thanks to the maintainers and contributors of both.
