# catalog

Go HTTP API template: DDD inside a vertical slice, no framework leaking into the domain.
`product` is the reference slice — copy it to add a new one.

## Layout

```
cmd/api                     process entrypoint: config, wiring, graceful shutdown
internal/product            the vertical slice
  product.go                aggregate + invariants (New)
  create.go / get.go        use cases: pure decision, I/O only through Store
  database.go               Store port + Postgres adapter
  handler.go                HTTP transport: DTOs, routes, error mapping
  errors.go                 domain errors (ErrNotFound, InvalidError)
internal/platform           shared kernel, no business rules
  config                    environment, fail fast on startup
  httpx                     chi router, middlewares, /healthz and /readyz
  logging                   structured slog + request_id propagated via context
  postgres                  connection pool
migrations                  plain SQL, applied by compose on first boot
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
  slice logs with it, no argument threading.
- **Bounded I/O.** Every query runs under `DB_TIMEOUT`; the server has read/write timeouts and
  shuts down gracefully.

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
make up          # postgres + api on :8080, migrations applied on first boot
make check       # fmt, lint, test -race, build
```

OpenAPI at `http://localhost:8080/docs`, health at `/healthz` and `/readyz`.

## Endpoints

| Method | Path             | Description        |
| ------ | ---------------- | ------------------ |
| POST   | `/products`      | Create a product   |
| GET    | `/products/{id}` | Get a product      |
| GET    | `/healthz`       | Liveness           |
| GET    | `/readyz`        | Readiness (DB)     |

## Stack

chi (routing) · huma (OpenAPI + validation) · pgx (Postgres) · slog (logs) · golangci-lint

## License

MIT — see [LICENSE](LICENSE).
