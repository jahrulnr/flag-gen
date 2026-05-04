---
name: db-lifecycle-install-migrate
description: Implements DB lifecycle guard (status/install/migrate) per docs/assessment/database-lifecycle.md and docs/assessment/api-contract.md. Use when building /v1/system/status, /v1/system/install, /v1/system/migrate, schema versioning rules, and advisory-lock guarded flows.
---

# DB Lifecycle / Install / Migrate Guard (MVP)

## Source of truth

- `docs/assessment/database-lifecycle.md`
- `docs/assessment/database-schema.md` (schema_migrations table)
- `docs/assessment/api-contract.md` (`/system/*` endpoints)

## Required endpoints (MVP)

- `GET /v1/system/status`
- `POST /v1/system/install`
- `POST /v1/system/migrate`

## Backend classification states

Implement `db_unreachable`, `db_empty`, `db_ready`, `migration_optional`, `migration_required`, `db_ahead`, `db_invalid`.

## Non-negotiable behavior

- Frontend must never send DB credentials.
- Install/migrate guarded with MySQL advisory locks:
  - `GET_LOCK('flag_gen_install', timeout)`
  - `GET_LOCK('flag_gen_migrate', timeout)`
- If migration is required:
  - keep `/system/status` available
  - block protected dashboard writes/incompatible reads

## Migrator shape

Prefer two binaries (server + migrate) sharing one migrator package, as described in docs.

