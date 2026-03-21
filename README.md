# myapp — Hexagonal Architecture Template

## Layer Map

```
cmd/server          composition root, DI wiring only
    └── imports
internal/adapter/   HTTP, gRPC, Postgres — translate external world ↔ app
    └── imports
internal/app/       use-cases / business logic
    └── imports
internal/port/      interfaces (repository + service contracts)
    └── imports
internal/domain/    pure entities & value objects — no internal imports
```

## Dependency Direction (enforced by Go's import cycle checker)

```
domain ← port ← app ← adapter ← cmd
```

Inner layers **never** import outer layers. Violating this direction produces a
compile-time import-cycle error — the compiler is the enforcer, not convention.

| Layer | May import | Must NOT import |
|-------|-----------|-----------------|
| `domain` | stdlib only | anything in `internal/` |
| `port` | `domain` | `app`, `adapter`, `cmd` |
| `app` | `domain`, `port` | `adapter`, `cmd` |
| `adapter/*` | `app`, `port`, `domain` | `cmd` |
| `cmd/server` | everything | — |

## Directory Structure

```
.
├── cmd/
│   └── server/
│       └── main.go          # composition root / DI wiring
├── internal/
│   ├── domain/              # entities, value objects — no ORM/HTTP tags
│   ├── port/                # repository & service interfaces
│   ├── app/                 # use-cases; unit-testable without DB or HTTP
│   └── adapter/
│       ├── postgres/        # implements port.Repository interfaces
│       ├── http/            # REST handlers (net/http)
│       └── grpc/            # gRPC handlers (stub until proto is generated)
└── pkg/
    └── validate/            # generic helpers; zero domain knowledge
```

## Running

```bash
# add a postgres driver first, e.g.:
go get github.com/jackc/pgx/v5/stdlib

# then uncomment the blank import in cmd/server/main.go, and:
go run ./cmd/server
```

## Testing the Application Layer

The `app` package tests require no external process — a `stubUserRepo` in the
test file satisfies `port.UserRepository` entirely in memory:

```bash
go test ./internal/app/...
```
