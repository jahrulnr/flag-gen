# Flag Gen — Quickstart (local)

## Prerequisites

- Go 1.26+
- MySQL 8+
- Redis (optional for L2; service degrades without Redis for reads)
- RabbitMQ (optional; required for outbox dispatch + broadcast consumer)

## Environment

| Variable | Purpose |
|----------|---------|
| `MYSQL_DSN` | MySQL DSN (required for server) |
| `MIGRATIONS_DIR` | Path to `migrations/mysql` (default `migrations/mysql`) |
| `REDIS_ADDR` | Redis address (optional) |
| `RABBITMQ_AMQP_URL` | AMQP URL for outbox publisher / broadcast consumer |
| `RABBITMQ_EXCHANGE` | Topic exchange name (default `flaggen.events`) |
| `OUTBOX_WORKER_ENABLED` | `true` to run outbox dispatcher in-process |
| `BROADCAST_CONSUMER_ENABLED` | `true` to consume `flag.*` and fan out to WebSocket hub |
| `BROADCAST_DEV_TOKEN` | Shared token for `/v1/broadcast/flags` query `token=` |
| `ALLOW_DEV_MIGRATE` | Allow `POST /v1/system/migrate` outside dev when `true` |
| `PUBLIC_READ_API_KEY` | Required `X-Public-Api-Key` for `/v1/public/...` when set |
| `SNAPSHOT_DIR` | Snapshot file directory |
| `CLEANUP_INTERVAL_SECONDS` | Soft-delete purge interval (default daily) |
| `SOFT_DELETE_RETENTION_DAYS` | Retention before hard delete (default 90) |

## Migrate schema (CLI)

```bash
export MYSQL_DSN='user:pass@tcp(127.0.0.1:3306)/flaggen?parseTime=true'
export MIGRATIONS_DIR=./migrations/mysql
go run ./cmd/migrate
```

## Run API server

```bash
go run ./cmd/app
```

Health: `GET http://localhost:8080/health`  
Lifecycle: `GET http://localhost:8080/v1/system/status`

## Install (empty DB)

When `database.state` is `db_empty`, call:

`POST /v1/system/install` with admin + optional seed (see `docs/assessment/api-contract.md`).

## WebSocket broadcast (dev)

`GET ws://localhost:8080/v1/broadcast/flags?projectId=...&environmentKey=...&token=<BROADCAST_DEV_TOKEN>`

Requires `BROADCAST_DEV_TOKEN` to match when set.
