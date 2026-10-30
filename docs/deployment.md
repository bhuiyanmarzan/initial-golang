# Deployment

Build with `make build` or the supplied Dockerfile. Docker runs as non-root and
excludes environment/secrets files from build context. Docker image building was
not verified in the restricted development environment.

For production configure ENV=production, RATE_LIMIT_BACKEND=redis, REDIS_URL,
DATABASE_URL, rate policies, and trusted proxy CIDRs. Migration jobs receive a
separate MIGRATION_DATABASE_URL and execute before deploying the API; do not run
migrations automatically on API startup. Do not give migration credentials to
the API process. API config does not require them.

The API listens on PORT on all interfaces. Block direct public access using the
host firewall or publish the container port on 127.0.0.1 only. Terminate TLS at
your proxy; restrict PostgreSQL and Redis to private interfaces. Use rediss://
with certificate verification when Redis crosses an untrusted network. The
Redis adapter supports a single endpoint, not Redis Cluster/Sentinel routing.
Give its Redis ACL access to PING, EVAL, TIME, HMGET, HSET, PEXPIRE and its key
prefix, plus SELECT if a nondefault database is used. Use a dedicated Redis
instance or deliberate memory sizing/noeviction: key eviction resets limits.

`deploy/nginx.conf.example` is a direct-edge template, not a complete TLS setup.
For local nginx use AUTH_TRUSTED_PROXY_CIDRS=127.0.0.1/32,::1/128. The edge
replaces X-Forwarded-For. When Cloudflare is added, configure nginx real-IP
handling for maintained Cloudflare network ranges and reject direct origin
traffic; do not trust a client-provided CF-Connecting-IP unconditionally.
Do not copy stale Cloudflare network ranges into application code.

Keep readiness restricted to infrastructure. Liveness says the process is
running; readiness checks database/Redis reachability, not business-schema or
migration compatibility. Operational probes are deliberately exempt from limits.

`deploy/eghissab.service.example` assumes binaries in /opt/eghissab and a protected
/etc/eghissab/api.env file. Create the service user and configure paths before
installing. Shutdown drains HTTP requests for SHUTDOWN_TIMEOUT, then closes
active connections on timeout and releases resources. Stop timeouts at the
service/container layer must exceed SHUTDOWN_TIMEOUT.

Operational signals: JSON request logs with server-generated request IDs,
status and latency; startup policy configuration; throttled limiter-outage logs;
readiness failures. Application logs never record request bodies, query strings,
Redis URLs, or raw limiter keys. Central log/metrics collection, TLS provisioning,
backups and restore drills remain deployment responsibilities.
