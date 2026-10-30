# Manual verification (no automated tests)

These checks are instructions for the developer; they have NOT been executed
against live services as part of this refactor. Use a development database.

## Build and startup

Run `make build` and optionally `make vet`. Invalid numeric values, negative
limits, invalid proxy CIDRs, and production without Redis must fail startup.
Unreachable required PostgreSQL/Redis must fail startup. Set the correct
DATABASE_URL; do not print it in logs or screenshots.

## DTOs and routing

- GET /healthz and GET /api/healthz: 200 with the existing success envelope.
- GET /readyz: 200 when all configured dependencies are reachable, 503 otherwise.
- GET /readyz?dependency=database: database readiness only.
- GET /readyz?dependency=unknown or duplicate dependency parameters: JSON 400.
- GET /api/v1/identity/health, /api/v1/auth/health, /api/v1/users/health: typed
  status responses. These are not login/registration implementations.
- Unknown route: JSON 404. Unsupported method on a registered route: standard
  ServeMux handling (the generic fallback may return JSON 404).
- Oversized bodies: 413. JSON decode/validation checks belong to the first JSON
  business endpoint; no public echo/debug endpoint was added just to exercise them.

## Rate limits

Restart with RATE_LIMIT_RPM=6 RATE_LIMIT_BURST=2 in development. Manually request
/api/v1/users/health three times rapidly: first two 200, third 429 with JSON
rate_limited and Retry-After. After approximately 10 seconds one token is
available. Changing RPM does not change burst unless RATE_LIMIT_BURST changes.
GET/HEAD /healthz, /api/healthz and /readyz remain exempt.

Use AUTH_RATE_LIMIT_RPM=6 AUTH_RATE_LIMIT_BURST=1 to observe the independent
strict policy at /api/v1/auth/health. Separate source IPs have independent
buckets. Loopback traffic from one host normally shares one IP.

With no trusted proxies, varying X-Forwarded-For must not reset limits. Through
a configured trusted nginx, verify real client addresses remain independent and
spoofed client-supplied forwarded headers are overwritten. Check both IPv4 and
IPv6. Headers over 2048 bytes or 32 hops from trusted proxies return 400.

Use Redis mode and start two API processes with the same REDIS_PREFIX and
policies on different ports. Alternating requests must share the same bucket.
Restarting an API must not reset the Redis bucket. All instances must use the
same policy values. Use a different prefix for each environment.

Stop Redis after successful startup. Sensitive auth/identity paths should return
503, general paths should use the bounded local fallback, readiness should be
503, and liveness should remain 200. Restore Redis and verify shared limiting
resumes. The fallback allows its own initial burst; it is not globally consistent.
A process restart while Redis is down must fail startup.

For local capacity, configure RATE_LIMIT_MAX_CLIENTS=1 and consume one bucket;
a second independent key/policy should get 503 until inactive state expires.
Memory expiration includes the full refill duration plus one minute, and cleanup
runs every minute. This prevents low-refill buckets from resetting prematurely.

## Shutdown and migration role

Send SIGTERM during normal operation. Observe clean shutdown and closed resources;
restart without orphan listeners. Migrations must require MIGRATION_DATABASE_URL
and never fall back to DATABASE_URL. Do not execute down on valuable data.

## Future identity/RLS gates

After identity and tenant schema implementation, manually verify 401 without a
verified principal, user limits across IP changes, denied unauthorized store IDs,
and RLS denial across organizations/stores using the actual restricted DB role.
Exercise rollback, panic cleanup, cancellation, and pooled-connection reuse.
The current repository does not contain the tenant tables/policies needed to
perform those checks honestly.
