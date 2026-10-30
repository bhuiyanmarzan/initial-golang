# Modular monolith contract

## Ownership

`cmd/api` owns the process. `internal/app` constructs shared dependencies, mounts
modules, assembles middleware, and owns shutdown. `internal/platform` contains
infrastructure and must not import business modules. Each business module owns
its routes, DTOs, business rules, and persistence. Cross-module calls use explicit
service contracts, not another module's repository or tables.

Module dependencies must stay acyclic. Keep the existing module path until a
repository rename is deliberately coordinated. Do not create empty business
modules solely to mirror the product roadmap.

## DTO convention

Use module-local, operation-specific request and response DTOs. Transport types
have JSON tags; domain models and service inputs do not need them. DTOs never
contain SQL driver types, password hashes, or internal authorization flags.

Request flow:

1. Handler uses httpx.DecodeJSON for JSON bodies (requires application/json,
   rejects unknown fields and trailing JSON, global middleware bounds body size).
2. Validate required fields, formats, and lengths in request DTO validation.
3. Explicitly map into a service input, adding verified actor context separately.
4. Service authorizes the operation and applies business rules.
5. Repository persists through the supplied transaction.
6. Explicitly map the result to a response DTO and use httpx.JSON.

The implemented health module demonstrates this boundary: ReadinessRequest ->
CheckInput -> Result -> Response. Its request is a validated query DTO, since
health does not accept JSON bodies. Identity status uses StatusResponse.
There are no fabricated registration DTOs or unused CRUD repositories: introduce
those with the identity schema and actual registration use case.

A future identity module should add handler.go, dto.go, mapper.go, service.go,
model.go, repository.go and repository_postgres.go as those responsibilities are
implemented. Repository interfaces are owned by their consuming module. Keep
files flat until a module's size justifies subpackages.

PATCH DTOs may use httpx.Optional[T] for omitted/null/value distinction. Validate
whether null is allowed per field. Avoid automatic field copying; explicit
mapping prevents clients from setting role, owner, or tenant fields accidentally.

## Middleware and protection

Request ID -> request logging/recovery -> body limit -> IP policy -> router.
When authentication is implemented, protected routes must use authentication ->
App.Protected -> authorization -> handler. Only the authentication adapter may
set authcontext.Principal after validating a token/session. App.Protected returns
401 without that context; it does not authenticate a request itself. Expensive
routes can additionally use App.Operation with a unique policy name.

Public and sensitive IP policies are selected before routing, including invalid
routes in those namespaces. General public-IP limits and authenticated-user
limits are independent layers. Client IP resolution walks trusted proxy hops
right-to-left, ignores forwarded headers from untrusted peers, normalizes IPv4,
and bounds forwarded-header size and hop count. Do not trust all networks.

Redis uses an atomic Lua script with Redis TIME, namespaced hashed client keys,
and TTLs that do not reset outstanding debt early. The small RESP2 adapter
supports redis/rediss, AUTH, SELECT, PING and EVAL; it has bounded connections,
response sizes, deadlines, and no automatic retries. It is a single-endpoint
adapter, not a Redis Cluster/Sentinel client. Local buckets are bounded and
periodically cleaned; capacity exhaustion returns 503 instead of bypassing limits.

## Transactions and RLS

Use Transactor.Within for non-tenant operations. For tenant operations, obtain
an authorized Scope and call WithinScope. Repositories must use the callback's
*sql.Tx or DBTX, never the shared *sql.DB, for every statement in that operation.
Nested transactions are not implicit: pass the existing transaction onward.
Transactions roll back on error/panic and commit only on success.

WithinScope sets app.user_id, app.org_id, app.store_id and app.customer_id using
parameterized set_config(..., true). All four keys are set, including empty
values, so absent scopes cannot inherit earlier values. SQL policies should use
NULLIF(current_setting('app.store_id', true), '') before casting to UUID.

These settings do not validate membership or enforce RLS by themselves. Future
schema migrations must define policies, grants, and constraints; the runtime
role must not be superuser, have BYPASSRLS, or bypass policies as table owner.
Use FORCE ROW LEVEL SECURITY where applicable and verify with the runtime role.
No tenant policies are applied to the unrelated legacy listings table here.
