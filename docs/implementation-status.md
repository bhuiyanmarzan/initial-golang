# Foundation implementation status

Implemented:
- Application-owned DB pool, transaction helper, logger and limiter lifecycle.
- Graceful HTTP shutdown, server timeouts, strict configuration, migration-role separation.
- Identity status consolidation with auth/users compatibility routes.
- Request/service/response DTO separation demonstrated by readiness, explicit mapping,
  strict JSON decoder and optional PATCH values for future business handlers.
- Trusted proxy client-IP resolution, memory bounds/cleanup, Redis atomic token buckets,
  strict sensitive policies, general fallback and reusable authenticated/operation policies.
- Transaction-local RLS settings; schema-dependent isolation is explicitly pending.
- Structured logs, request IDs, panic recovery, body limits, liveness/readiness.
- Build/deployment configuration and manual verification instructions.

Verification performed: gofmt, go build ./..., go vet ./..., and git diff --check
passed. Builds and vet used GOCACHE=/tmp/eghissab-go-cache because the default
cache was read-only. No automated tests run.
No live DB migration, Redis exercise, Docker build, or deployment was performed.

Prerequisites for business implementation:
1. Define/apply the EGHissab identity and tenant schema; only listings exists here.
2. Implement authentication, request DTOs and use cases against that schema.
3. Place authenticated-user policies after real authentication, not status endpoints.
4. Authorize tenant Scope and implement/verify RLS policies and grants.
5. Perform the manual verification guide with the real proxy and Redis deployment.

The old README is retained in legacy-learning-notes.md. Existing .env was not
modified. The original Go module name remains unchanged. No tests were added.
