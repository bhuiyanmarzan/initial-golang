# EGHissab backend

Single-service Go modular monolith using net/http, database/sql with pgx, raw SQL
migrations, explicit API DTOs, and configurable token-bucket rate limiting.
The existing Go module path is preserved: github.com/bhuiyanmarzan/olx-api.

## Start locally

1. Install the Go version declared in go.mod and run PostgreSQL.
2. Copy `.env.sample` to `.env` if you do not already have one. Set your own
   database credentials. Existing `.env` files are never overwritten by setup.
3. Run `make migrate-up` using migration-role credentials. The current schema
   contains ONLY the original listings table, not the EGHissab business schema.
4. Run `make run` (or `make dev` if Air is installed).

| Endpoint | Behavior |
| --- | --- |
| GET /healthz | Liveness; exempt from limiting |
| GET /api/healthz | Legacy liveness alias; exempt from limiting |
| GET /readyz | Checks PostgreSQL and Redis when configured; exempt from limiting |
| GET /readyz?dependency=database | Check PostgreSQL only |
| GET /readyz?dependency=redis | Check Redis only; 503 if not configured |
| GET /api/v1/identity/health | Identity module status; strict IP policy |
| GET /api/v1/auth/health | Compatibility status endpoint; strict IP policy |
| GET /api/v1/users/health | Compatibility status endpoint; general IP policy |

Identity status routes do not implement login or registration. They never issue
sessions or establish a verified principal.

## Development commands

- `make build`: compile API and migration binaries.
- `make fmt`: format code.
- `make vet`: static analysis, not automated tests.
- `make migrate-up`: apply migrations with MIGRATION_DATABASE_URL.
- `make migrate-down`: roll back one migration; outside development also requires
  ALLOW_MIGRATION_DOWN=true. Review data loss before running.

## Rate limits

Defaults allow 30 immediate general requests per IP and replenish 2 tokens per
second (120 RPM). RPM is a refill rate, not a fixed-minute request counter.
Auth/identity paths use 10 RPM with a burst of 5. All GET/HEAD operational probes
are exempt; use `/api/v1/users/health` to manually observe the general limit.

Set RATE_LIMIT_BACKEND=redis and REDIS_URL for shared limits. Staging and
production require Redis. On a Redis outage, sensitive paths return 503;
general paths use bounded per-process fallback buckets. Startup requires Redis
to be reachable; readiness becomes 503 during outages. Fallback limits are not
fleet-wide and restart with the process.

## Guides

- [Architecture and DTO conventions](docs/architecture.md)
- [Manual verification](docs/manual-verification.md)
- [Deployment](docs/deployment.md)
- [Implementation status and remaining prerequisites](docs/implementation-status.md)
- [Original learning notes](docs/legacy-learning-notes.md)
