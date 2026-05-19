# Flag Gen — Quickstart (local)

Documentation: [../README.md](../README.md) · runtime: [how-it-works.md](how-it-works.md)

## Prerequisites

- Go 1.26+
- MySQL 8+
- Redis (optional for L2; service degrades without Redis for reads)
- RabbitMQ (optional; required for outbox dispatch + broadcast consumer)

If you use the shared **`/apps/services/compose.yml`** stack: service **`database`** is MariaDB on `127.0.0.1:3306` (`root` / `root`); **`redis`** uses `requirepass guest` on `127.0.0.1:6379`; **`rabbitmq`** listens on `127.0.0.1:5672` (default `guest` / `guest` from localhost). Copy **`.env.example`** to **`.env`**, create database `flag_gen`, then `export $(grep -v '^#' .env | xargs)` (or your shell equivalent) before `go run`.

## Environment

| Variable | Purpose |
|----------|---------|
| `MYSQL_DSN` | MySQL DSN (required for server). In `loc=`, escape `/` as `%2F` (e.g. `loc=Asia%2FJakarta`); use `loc=UTC` or omit if unsure. Do not use `UTC+7` (`+` is treated as space; not a valid IANA zone in Go). |
| `MIGRATIONS_DIR` | Path to `migrations/mysql` (default `migrations/mysql`) |
| `REDIS_ADDR` | Redis address (optional) |
| `RABBITMQ_AMQP_URL` | AMQP URL for outbox publisher / broadcast consumer |
| `RABBITMQ_EXCHANGE` | Topic exchange name (default `flaggen.events`) |
| `OUTBOX_WORKER_ENABLED` | `true` to run outbox dispatcher in-process (publishes flag changes to RabbitMQ) |
| `BROADCAST_CONSUMER_ENABLED` | `true` to consume `flag.*` from RabbitMQ and fan out to WebSocket hub |
| `BROADCAST_DEV_TOKEN` | Shared token for `/v1/broadcast/flags` query `token=` (dev local) |

For live WebSocket updates after dashboard edits, set **`OUTBOX_WORKER_ENABLED=true`** and **`BROADCAST_CONSUMER_ENABLED=true`** (RabbitMQ on `RABBITMQ_AMQP_URL` must be running), then restart the server.
| `ALLOW_DEV_MIGRATE` | Allow `POST /v1/system/migrate` outside dev when `true` |
| `PUBLIC_READ_API_KEY` | Required `X-Public-Api-Key` for `/v1/public/...` when set |
| `SNAPSHOT_DIR` | Snapshot file directory |
| `CLEANUP_INTERVAL_SECONDS` | Soft-delete purge interval (default daily) |
| `SOFT_DELETE_RETENTION_DAYS` | Retention before hard delete (default 90) |

## Migrate schema (CLI)

From the repo root (loads **`.env`**):

```bash
make migrate
```

Equivalent without Make:

```bash
export MYSQL_DSN='user:pass@tcp(127.0.0.1:3306)/flaggen?parseTime=true&loc=UTC'
export MIGRATIONS_DIR=./migrations/mysql
go run ./cmd/migrate
```

## Run API server

```bash
go run ./cmd/app
```

Health: `GET http://localhost:<APP_PORT>/health` (default in `.env.example` is `6123`, aligned with `make dev` + Angular dev API URL)  
Lifecycle: `GET http://localhost:<APP_PORT>/v1/system/status`

## Dashboard API curl examples

These examples target the same `/v1` routes used by the Angular dashboard.

```bash
BASE=http://localhost:6123/v1

# List projects and copy the Project ID (UUID) from the row — not the project key.
curl "$BASE/system/status"
curl "$BASE/projects"
PROJECT_ID="<paste-project-uuid-from-dashboard>"

curl -X POST "$BASE/projects" \
  -H 'Content-Type: application/json' \
  -d '{"key":"default","name":"Default"}'

curl "$BASE/projects/$PROJECT_ID/environments"
curl -X POST "$BASE/projects/$PROJECT_ID/environments" \
  -H 'Content-Type: application/json' \
  -d '{"key":"production","name":"Production"}'

curl "$BASE/projects/$PROJECT_ID/environments/production/flags"
curl -X POST "$BASE/projects/$PROJECT_ID/environments/production/flags" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: demo-create-checkout' \
  -d '{"key":"checkout","name":"Checkout","enabled":true,"clientSide":true,"value":{"type":"boolean","default":false},"rules":[]}'

curl "$BASE/projects/$PROJECT_ID/environments/production/flags/evaluate?flagKey=checkout&userId=user-1"
curl "$BASE/public/projects/$PROJECT_ID/environments/production/flags" \
  -H 'X-Public-Api-Key: dev-public-key'
```

## Install (empty DB)

When `database.state` is `db_empty`, call:

`POST /v1/system/install` with admin + optional seed (see [../assessment/api-contract.md](../assessment/api-contract.md)).

## WebSocket broadcast (dev)

`GET ws://localhost:<APP_PORT>/v1/broadcast/flags?projectId=...&environmentKey=...&token=<BROADCAST_DEV_TOKEN>`

Requires `BROADCAST_DEV_TOKEN` to match when set.
