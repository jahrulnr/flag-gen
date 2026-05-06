# Flag Gen

Flag Gen is a self-hosted feature flag management service built for backend and platform teams that need controlled rollouts, fast rollback, runtime flag evaluation, and operationally safe degraded reads.

The service combines a Go API, MySQL persistence, Redis-backed read caching, RabbitMQ outbox dispatch, file-backed snapshots, a Go SDK, and an Angular admin dashboard.

## What It Provides

- Project and environment management for scoped flag ownership.
- Boolean and JSON feature flags with version history, audit log, rollback, and soft delete retention.
- Targeting rules and deterministic percentage rollouts.
- Runtime evaluation over HTTP with correlation IDs and structured error envelopes.
- Browser-safe public flag reads that only expose `clientSide` flags and a reduced payload.
- WebSocket broadcast for service-to-service subscribers.
- Dashboard username/password session auth, user management, and role-based permissions.
- API key generation, rotation, revocation, scopes, and per-key evaluation rate limiting.
- Optional signed webhook delivery from committed flag events.
- Transactional outbox for committed flag change events.
- L1 Ristretto cache, L2 Redis cache, file snapshot fallback, singleflight refill, and degraded circuit breaking.
- Snapshot generation from committed DB state and consistency checks guarded by Redis locks.
- Angular admin dashboard for projects, environments, flags, segments, history, audit, migration status, and realtime events.
- Lightweight Go client and server-side polling SDK.

## Architecture

```text
Angular dashboard (/app)
        |
        | /api and /v1 HTTP
        v
Go service: Gin handlers -> usecases -> domain repositories
        |
        +-- MySQL: projects, environments, flags, rules, versions, audit, outbox, snapshots
        +-- Redis: shared scoped flag cache and distributed locks
        +-- Ristretto: per-process hot scoped flag cache
        +-- Snapshot files: last-known-good fallback state
        +-- RabbitMQ: outbox publish and broadcast fan-out
```

Runtime read path:

```text
L1 Ristretto -> L2 Redis -> MySQL -> validated snapshot file -> 503 service_degraded
```

Snapshot files are fallback data, not a normal third cache tier. When snapshot data is used during degraded reads, cache refill uses a shorter TTL.

## Repository Layout

| Path | Purpose |
|------|---------|
| `cmd/app` | Main API/dashboard server |
| `cmd/migrate` | CLI migrator |
| `cmd/seed` | Idempotent local demo seed |
| `application` | Usecases and background jobs |
| `domain` | Entities, value objects, and repository ports |
| `delivery/http` | Gin handlers, middleware, router, dashboard proxy |
| `infrastructure` | MySQL, Redis/cache, RabbitMQ, snapshots, DI, metrics |
| `migrations/mysql` | SQL migrations |
| `sdk/flagclient` | Go HTTP client and server-side polling SDK |
| `web` | Angular SSR admin dashboard |
| `docs` | API, quickstart, metrics, security, and assessment docs |

## Requirements

- Go 1.26+
- Node.js and npm matching `web/package.json`
- MySQL or MariaDB
- Redis, optional but recommended for L2 cache and distributed locks
- RabbitMQ, optional unless outbox dispatch or broadcast is enabled

The local `.env.example` targets the shared services stack:

- MySQL/MariaDB: `127.0.0.1:3306`, user `root`, password `root`
- Redis: `127.0.0.1:6379`, password `guest`
- RabbitMQ: `127.0.0.1:5672`, user `guest`, password `guest`

## Local Setup

Create the database once:

```bash
mysql -h127.0.0.1 -uroot -proot \
  -e "CREATE DATABASE IF NOT EXISTS flag_gen CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

Copy and review environment variables:

```bash
cp .env.example .env
```

Apply migrations:

```bash
make migrate
```

Seed demo data:

```bash
make seed
```

Run API and dashboard together:

```bash
make dev
```

Open:

- Dashboard dev server: `http://localhost:6124/app/`
- API health: `http://localhost:6123/health`
- Lifecycle status: `http://localhost:6123/v1/system/status`
- Metrics: `http://localhost:6123/v1/system/metrics`

## Demo Credentials and Seed Data

The migration `202605040002_bootstrap_dev_admin` inserts the local demo user only when the `users` table is empty:

| Field | Value |
|-------|-------|
| Email | `admin@local` |
| Password | `admin` |

The current dashboard includes a local dev bypass for the UI. Use the credentials above for demos and documentation; the full password-backed dashboard session flow remains a roadmap item.

`make seed` is idempotent. With the default `.env`, it creates or reuses:

- Project key: `development`
- Environment key: `development`
- Segments: `seed_demo_segment`, `seed_demo_cohort`
- Flags: `seed_demo_toggle`, `seed_demo_json`, `seed_demo_rollout`, `seed_demo_rules`

## Dashboard

The Angular dashboard is served under `/app/` and uses `/api` for same-origin dashboard calls.

Development commands:

```bash
make dev-web      # Angular only on :6124/app
make dev-api      # Go API only on :6123
make dev          # API + dashboard
```

Dashboard views currently include:

- Login/dev bypass
- Database lifecycle/install/migration status
- Projects
- Environments
- Flags with create/edit/delete, rule builder, rollout editor, and history link
- Segments
- Audit log
- User management
- Role management with read-only, editor, business, admin, and superadmin presets
- Realtime broadcast status
- API key management endpoints for admin/API workflows

## Screenshots

The screenshots below use the local seed scope (`development` / `development`) and the demo credentials from the bootstrap migration.

| View | Screenshot |
|------|------------|
| Login | ![Login](docs/screenshots/login.png) |
| Projects | ![Projects](docs/screenshots/projects.png) |
| Environments | ![Environments](docs/screenshots/environments.png) |
| Flags | ![Flags](docs/screenshots/flags.png) |
| Segments | ![Segments](docs/screenshots/segments.png) |
| Audit | ![Audit](docs/screenshots/audit.png) |

## HTTP API

Core API routes are under `/v1`. Dashboard proxy routes are under `/api` and share the same handlers with CSRF protection.

Common examples:

```bash
BASE=http://localhost:6123/v1

curl "$BASE/system/status"
curl "$BASE/projects"
curl "$BASE/projects/development/environments"
curl "$BASE/projects/development/environments/development/flags"

curl "$BASE/projects/development/environments/development/flags/evaluate?flagKey=seed_demo_toggle&userId=user-1"

curl "$BASE/public/projects/development/environments/development/flags" \
  -H 'X-Public-Api-Key: dev-public-key'
```

OpenAPI lives at:

- `docs/openapi.yaml`

## API Keys and Auth

Dashboard users authenticate through `/api/auth/login`, which sets the `flag_gen_session` HTTP-only cookie. The seed/bootstrap admin is:

- Email: `admin@local`
- Password: `admin`

API keys are scoped to a project/environment:

```bash
curl -X POST "$BASE/projects/development/environments/development/api-keys" \
  -H 'Content-Type: application/json' \
  -d '{"name":"server sdk","keyType":"server","scopes":["flags:read","flags:evaluate"]}'
```

Use the returned one-time `token` with:

```bash
curl "$BASE/projects/development/environments/development/flags/evaluate?flagKey=seed_demo_toggle&userId=user-1" \
  -H "X-API-Key: <token>"
```

Supported scopes include `flags:read`, `flags:write`, `flags:evaluate`, `segments:*`, and `broadcast:subscribe`.

## Webhooks

Set these variables to enable signed delivery of committed flag events:

```bash
WEBHOOK_WORKER_ENABLED=true
WEBHOOK_URLS=https://example.internal/flag-events
WEBHOOK_SECRET=change-me
```

Each delivery includes `X-Flag-Gen-Event-ID`, `X-Flag-Gen-Event-Type`, and `X-Flag-Gen-Signature: sha256=<hmac>` when a secret is configured. Delivery retries three times with short backoff.

## SDK Usage

Minimal HTTP client:

```go
client := flagclient.New("http://localhost:6123/v1")
result, err := client.Evaluate(ctx, "development", "development", "user-1", "seed_demo_toggle")
```

Server-side local state with polling fallback:

```go
client := flagclient.New("http://localhost:6123/v1")
sdk := flagclient.NewServerSDK(client, "development", "development", 30*time.Second)

if err := sdk.Refresh(ctx); err != nil {
    return err
}

sdk.StartPolling(ctx)
enabled := sdk.Variation("seed_demo_toggle", false).(bool)
```

## Operational Behavior

- Flag mutations write flag state, version history, audit entries, and outbox events transactionally.
- Outbox dispatch publishes committed events and retries failures with backoff.
- Successful outbox publish triggers snapshot refresh for the affected flag scope.
- Cache invalidation runs after local mutations and on outbox/broadcast paths.
- Redis reconnect is signalled from retryable L2 errors and handled by a guarded background probe.
- Snapshot checksum or format failures mark metadata invalid and emit corruption metrics.
- Cleanup jobs prune soft-deleted data and cap audit/history rows with Redis lock guards.

## Metrics and Alerts

Metrics are exposed as plain text:

```bash
curl http://localhost:6123/v1/system/metrics
```

Important implemented signals:

- `outbox_publish_attempts_total`
- `outbox_publish_success_total`
- `outbox_publish_failures_total`
- `evaluate_service_degraded_total`
- `scoped_cache_service_degraded_total`
- `redis_reconnect_attempts_total`
- `redis_reconnect_success_total`
- `redis_reconnect_failures_total`
- `snapshot_file_corruption_total`
- `snapshot_outbox_triggers_total`

See `docs/metrics.md` for alert suggestions.

## Transport Security

For direct TLS:

```bash
USE_SSL=true
APP_BASE_DIR=/app
PATH_SSL_CERT=/certs/fullchain.pem
PATH_SSL_KEY=/certs/privkey.pem
```

For production deployments, prefer TLS termination at a managed load balancer or reverse proxy and pin `CORS_ALLOWED_ORIGINS` to the dashboard origin. See `docs/transport-security.md`.

## Development Commands

```bash
make help
make migrate
make seed
make dev
make build
make unit-test-coverage
go test ./...

cd web
npm run build
```

If Wire providers change:

```bash
make wire
```

## Status

The implemented MVP covers core flag management, runtime evaluation, dashboard workflows, cache/snapshot resilience, outbox broadcast plumbing, metrics, and local demo flows.

Remaining larger roadmap items include:

- API key lifecycle with scoped permissions and per-key rate limiting
- Password-backed dashboard sessions and RBAC
- Webhook delivery worker with retry and signatures
- Full GrowthBook-level UX parity review
- Production observability export beyond the current in-process text registry
