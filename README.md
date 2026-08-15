# catalog

Go HTTP API template: DDD inside a vertical slice, no framework leaking into the domain.
`product` is the reference slice — copy it to add a new one.

## Layout

```
cmd/api                     process entrypoint: config, wiring, graceful shutdown
cmd/migrate                 applies pending migrations, tracked in schema_migrations
internal/product            the vertical slice
  product.go                aggregate + invariants (New)
  create.go / get.go        use cases: pure decision, I/O only through Store
  database.go               Store port + Postgres adapter
  handler.go                HTTP transport: DTOs, routes, error mapping
  errors.go                 domain errors (ErrNotFound, InvalidError)
internal/platform           shared kernel, no business rules
  config                    environment, fail fast on startup
  httpx                     router, middlewares, RFC 7807 errors, metrics, health
  logging                   structured slog + request_id propagated via context
  postgres                  connection pool
internal/migrations         goose migrations embedded in the binary
```

## Rules the template enforces

- **A slice owns its full stack.** Domain, use case, persistence and transport live in one
  package because they change together. Slices never import each other; the consumer declares
  the port it needs and `cmd/api` wires the implementation.
- **Decision separated from I/O.** `New` applies the invariants and touches nothing external.
  Use cases read and write only at the edges, through `Store`.
- **Invariants live in the type.** A `Product` cannot exist invalid — there is no path to one
  except `New`, so validation is not scattered across handlers.
- **Interfaces only where they are actually swapped.** `Store` exists because tests replace it;
  the pool and the logger are concrete types.
- **Testable without a container.** Every test runs on `go test ./...` with a fake store.
- **Observability at the edges.** Each request gets a `request_id` carried in the context; the
  slice logs with it, no argument threading. Latency and status are exported per route on
  `/metrics` in Prometheus text format.
- **One error shape.** Every failure path — validation, domain, panic, 404, 405, 415 — answers
  `application/problem+json` with `instance` set to the request id.
- **Bounded I/O.** Every query runs under `DB_TIMEOUT`, every request under `REQUEST_TIMEOUT`,
  request bodies are capped at 1 MiB, and the server shuts down gracefully.
- **Versioned surface.** Slices register into a `huma.Group` prefixed with `/v1`; health checks
  stay outside it. Unknown query parameters and non-JSON content types are rejected.

## Adding a slice

1. `internal/<name>/` with the aggregate and its `New`.
2. One file per use case, taking `ctx`, the port and a command struct.
3. `database.go`: the port the slice needs, plus the Postgres adapter.
4. `handler.go`: DTOs, `RegisterRoutes`, and the domain error to HTTP mapping.
5. Wire it in `cmd/api/main.go`. Needs data from another slice? Declare a narrow interface in
   the consumer and wire the adapter in `main` — never import the other slice.

## Running

```sh
cp .env.example .env
make up          # postgres, migrations, then the api on :8080
make check       # fmt, lint, test -race, build, govulncheck
```

OpenAPI at `http://localhost:8080/docs`, health at `/healthz` and `/readyz`.

## Endpoints

| Method | Path                | Description      |
| ------ | ------------------- | ---------------- |
| POST   | `/v1/products`      | Create a product |
| GET    | `/v1/products/{id}` | Get a product    |
| GET    | `/healthz`          | Liveness         |
| GET    | `/readyz`           | Readiness (DB)   |
| GET    | `/metrics`          | Prometheus text  |

Errors follow RFC 7807 (`application/problem+json`), with `instance` carrying the request id
so a client report maps straight to a log line.

## Migrations

SQL files live in `internal/migrations/sql`, embedded in the binary and applied by [goose](https://github.com/pressly/goose)
through `cmd/migrate` — as a library, so deploys ship one binary and no extra CLI. Each migration
runs in its own transaction and is recorded in `goose_db_version`, making re-runs a no-op. Compose
runs it as a job that must finish before the API starts.

```sh
make migrate                          # apply pending
make migrate-status                   # list pending
make migrate-create name=add_stock    # scaffold a new pair of Up/Down
```

Schema changes go out as expand → backfill → contract, never a destructive `ALTER` in a single
deploy: add the nullable column, write to both, backfill, move reads, then drop the old one.

## Stack

chi (routing + middleware) · huma (OpenAPI, validation, RFC 7807 errors) · goose (migrations) · pgx (Postgres) · slog (logs) · golangci-lint

## License

MIT — see [LICENSE](LICENSE).
