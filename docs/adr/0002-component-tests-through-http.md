# Test through the HTTP entry point with real Postgres and faked external services

Behaviour is tested as component tests: each test starts the real server via `server.New`, drives it over HTTP like a browser, runs against a real Postgres, and replaces every external service (Authentik, plex.tv, Plex servers, AniList, mapping datasets) with an in-process fake. We chose this over unit tests with mocks so internal contracts, schema and implementation can be refactored without rewriting tests; a test should only break when observable behaviour changes.

## Consequences

- Unit tests are reserved for complex, self-contained logic (`mappings.Resolve`, `secretbox`); everything else is reached through the API.
- Each adapter has a contract test against data recorded from the real service, which is what keeps the fakes honest.
- Tests don't read or write tables directly. Security properties that are only visible at rest, such as hashed session IDs and encrypted tokens, are checked by scanning the whole database for the raw secret, not by naming columns.
- Tests never reach the network beyond loopback: external URLs must be configured explicitly, and the test transport refuses non-loopback hosts.
- Database-failure branches are left untested rather than faked.
