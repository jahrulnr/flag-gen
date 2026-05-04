# Database Schema Recommendation

## Scope

This document proposes the initial MySQL schema for the flag-gen MVP. The schema supports dashboard/admin workflows, deterministic evaluation, server-side WebSocket full-scope synchronization, client-side HTTP flag reads, transactional outbox, audit/version history, and snapshot fallback metadata.

The schema intentionally keeps experimentation, analytics events, advanced RBAC, SSO, and webhook delivery details out of the MVP.

## Design Principles

- **MySQL is the source of truth** for projects, environments, flags, rules, versions, audit entries, access keys, and outbox events.
- **Redis is an acceleration layer**, not required for availability.
- **Snapshot files are fallback state**, tracked by metadata in MySQL but stored outside the database.
- **Flag mutations are transactional**: flag state, version record, audit entry, idempotency record, and outbox event are committed together.
- **Server-side runtime services are read-local**: they receive all scoped flags over WebSocket and later apply `flag.updated` or `flag.deleted` messages.
- **Client-side runtime reads use HTTP** and use `client_side = true` as browser-facing visibility metadata.
- **Feature flags are not secret storage**: flag payloads must be safe to expose; `client_side` is browser-facing visibility metadata, not a secret-protection boundary.
- **IDs use UUID strings in the schema proposal** for readability and easier debugging. Implementation can switch to `BINARY(16)` later if storage efficiency becomes important.

## Table Overview

| Table | Purpose |
| --- | --- |
| `users` | Dashboard user login and actor attribution |
| `user_sessions` | Dashboard session tracking |
| `schema_migrations` | Applied database migration history and dirty-state tracking |
| `projects` | Product/application boundary |
| `environments` | Runtime environment under a project |
| `flags` | Main flag aggregate state |
| `flag_rules` | Ordered targeting rules for a flag |
| `segments` | Reusable targeting segment definitions |
| `flag_versions` | Immutable flag snapshots for version history and rollback |
| `audit_entries` | Actor-attributed mutation history |
| `api_keys` | Lightweight scoped keys for SDK/WebSocket/client-side access |
| `idempotency_keys` | Retry safety for dashboard/admin mutation commands |
| `outbox_events` | Transactional event records for WebSocket/broker publishing |
| `snapshot_metadata` | Metadata for file-backed fallback snapshots |

## Conventions

- `id`: `CHAR(36)` UUID.
- `key`: human-readable stable key, lowercase snake case where possible.
- `version`: integer aggregate version used for optimistic locking and stale broadcast rejection.
- `scope_version`: integer environment-level version used for full-scope WebSocket snapshots and snapshot file freshness.
- `*_json`: MySQL `JSON` column.
- Timestamps use `DATETIME(3)` in UTC.
- Soft delete uses `deleted_at`; rows with `deleted_at` older than 3 months are hard-deleted by application cleanup.
- Unique keys that must ignore soft-deleted rows use generated `active_key` columns in the baseline SQL migration so active entities stay unique while soft-deleted keys can be reused.

## Core Tables

### users

Dashboard users for MVP username/password login. 2FA is optional and can be added without changing flag semantics.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `CHAR(36)` | Primary key |
| `email` | `VARCHAR(255)` | Unique login identity |
| `name` | `VARCHAR(255)` | Display name |
| `password_hash` | `VARCHAR(255)` | Argon2id/bcrypt hash |
| `two_factor_enabled` | `BOOLEAN` | Default `false` |
| `two_factor_secret_encrypted` | `VARBINARY(512)` | Nullable |
| `status` | `VARCHAR(32)` | `active`, `disabled` |
| `created_at` | `DATETIME(3)` | |
| `updated_at` | `DATETIME(3)` | |

Recommended indexes:

- `UNIQUE (email)`
- `INDEX (status)`

### user_sessions

Tracks dashboard sessions so logout/revocation can invalidate active sessions.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `CHAR(36)` | Primary key |
| `user_id` | `CHAR(36)` | FK to `users.id` |
| `session_hash` | `CHAR(64)` | Hash of opaque session token |
| `user_agent` | `VARCHAR(512)` | Nullable |
| `ip_address` | `VARCHAR(64)` | Nullable |
| `expires_at` | `DATETIME(3)` | |
| `revoked_at` | `DATETIME(3)` | Nullable |
| `created_at` | `DATETIME(3)` | |

Recommended indexes:

- `UNIQUE (session_hash)`
- `INDEX (user_id, expires_at)`

### schema_migrations

Tracks applied database migrations. This table is used by the backend to decide whether the database is empty, ready, outdated but compatible, requires migration, or is newer than the running app.

| Column | Type | Notes |
| --- | --- | --- |
| `version` | `BIGINT` | Primary key; timestamp-style migration version |
| `name` | `VARCHAR(255)` | Human-readable migration name |
| `checksum` | `CHAR(64)` | Checksum of migration content |
| `execution_ms` | `INT` | Migration runtime |
| `applied_by_app_version` | `VARCHAR(64)` | App version that applied the migration |
| `applied_at` | `DATETIME(3)` | UTC |
| `dirty` | `BOOLEAN` | True when a migration failed mid-run |

Recommended indexes:

- `PRIMARY KEY (version)`
- `INDEX (dirty)`
- `INDEX (applied_at)`

### projects

Top-level product/application boundary.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `CHAR(36)` | Primary key |
| `key` | `VARCHAR(128)` | Stable project key |
| `name` | `VARCHAR(255)` | |
| `description` | `TEXT` | Nullable |
| `created_by` | `CHAR(36)` | FK to `users.id`, nullable for bootstrap/system |
| `created_at` | `DATETIME(3)` | |
| `updated_at` | `DATETIME(3)` | |
| `deleted_at` | `DATETIME(3)` | Nullable |

Recommended indexes:

- `UNIQUE (key)`
- `INDEX (deleted_at)`

### environments

Environment under a project, such as `development`, `staging`, or `production`.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `CHAR(36)` | Primary key |
| `project_id` | `CHAR(36)` | FK to `projects.id` |
| `key` | `VARCHAR(128)` | Stable environment key |
| `name` | `VARCHAR(255)` | |
| `sort_order` | `INT` | Default `0` |
| `version` | `INT` | Environment scope version, increments on scoped flag changes |
| `created_at` | `DATETIME(3)` | |
| `updated_at` | `DATETIME(3)` | |
| `deleted_at` | `DATETIME(3)` | Nullable |

Recommended indexes:

- `UNIQUE (project_id, key)`
- `INDEX (project_id, deleted_at)`

### flags

Main aggregate state. This row is the optimistic-locking boundary.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `CHAR(36)` | Primary key |
| `project_id` | `CHAR(36)` | FK to `projects.id` |
| `environment_id` | `CHAR(36)` | FK to `environments.id` |
| `key` | `VARCHAR(255)` | Stable flag key |
| `name` | `VARCHAR(255)` | |
| `description` | `TEXT` | Nullable |
| `enabled` | `BOOLEAN` | Default `false` |
| `client_side` | `BOOLEAN` | Default `false`; include in browser/client-side HTTP reads |
| `value_type` | `VARCHAR(32)` | `boolean`, `json` |
| `default_value_json` | `JSON` | Stores boolean or JSON default |
| `rollout_percentage` | `DECIMAL(5,2)` | Nullable; `0.00` to `100.00` |
| `rollout_stickiness` | `VARCHAR(64)` | Default `userId` |
| `version` | `INT` | Starts at `1`, increments on mutation |
| `created_by` | `CHAR(36)` | FK to `users.id`, nullable |
| `updated_by` | `CHAR(36)` | FK to `users.id`, nullable |
| `created_at` | `DATETIME(3)` | |
| `updated_at` | `DATETIME(3)` | |
| `deleted_at` | `DATETIME(3)` | Nullable |

Recommended indexes:

- `UNIQUE (project_id, environment_id, key)`
- `INDEX (project_id, environment_id, deleted_at)`
- `INDEX (project_id, environment_id, client_side, deleted_at)`
- `INDEX (updated_at)`

Optimistic update shape:

```sql
UPDATE flags
SET enabled = ?, version = version + 1, updated_by = ?, updated_at = ?
WHERE id = ? AND version = ? AND deleted_at IS NULL;
```

### flag_rules

Ordered targeting rules for a flag. MVP starts with simple comparisons and optional segment references.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `CHAR(36)` | Primary key |
| `flag_id` | `CHAR(36)` | FK to `flags.id` |
| `priority` | `INT` | Lower number evaluated first |
| `attribute` | `VARCHAR(255)` | Nullable when using `segment_id` |
| `operator` | `VARCHAR(32)` | `equals`, `not_equals`, `in`, `not_in` |
| `comparison_value_json` | `JSON` | Value to compare against |
| `serve_value_json` | `JSON` | Value returned when rule matches |
| `segment_id` | `CHAR(36)` | Nullable FK to `segments.id` |
| `created_at` | `DATETIME(3)` | |
| `updated_at` | `DATETIME(3)` | |

Recommended indexes:

- `UNIQUE (flag_id, priority)`
- `INDEX (flag_id)`
- `INDEX (segment_id)`

### segments

Reusable targeting definitions. Rules are stored as JSON for MVP to avoid designing a full expression engine too early.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `CHAR(36)` | Primary key |
| `project_id` | `CHAR(36)` | FK to `projects.id` |
| `key` | `VARCHAR(128)` | Stable segment key |
| `name` | `VARCHAR(255)` | |
| `rules_json` | `JSON` | Explicit rule list |
| `version` | `INT` | Starts at `1` |
| `created_by` | `CHAR(36)` | Nullable FK to `users.id` |
| `updated_by` | `CHAR(36)` | Nullable FK to `users.id` |
| `created_at` | `DATETIME(3)` | |
| `updated_at` | `DATETIME(3)` | |
| `deleted_at` | `DATETIME(3)` | Nullable |

Recommended indexes:

- `UNIQUE (project_id, key)`
- `INDEX (project_id, deleted_at)`

## Soft Delete Retention Policy

All tables with a `deleted_at` column are retained for **3 months** after soft deletion. After that, the application cleanup job hard-deletes the oldest eligible rows. The expectation is that after 3 months, deleted entities are no longer needed for restore or normal operation.

MVP tables covered by this policy:

- `projects`
- `environments`
- `flags`
- `segments`

Cleanup should:

1. Run as a background job, not inside user-facing mutation transactions.
2. Delete only rows where `deleted_at < NOW() - INTERVAL 3 MONTH`.
3. Process in small batches to avoid long locks.
4. Delete children before parents where foreign keys require it.
5. Use local singleflight/mutex so repeated triggers inside one process do not run overlapping cleanup jobs.
6. Acquire a Redis distributed mutex before deleting rows so only one replica performs cleanup in horizontally scaled deployments.
7. Log cleanup counts and errors.

The Redis cleanup lock should use an owner token, TTL, renewal for long batches, and compare-and-delete unlock semantics. If Redis is unavailable, skip cleanup and retry later; cleanup is maintenance work and must not block runtime reads or flag mutations.

Recommended hard-delete order:

1. For deleted `flags`: delete related `flag_rules`, then `flags`.
2. For deleted `segments`: ensure no active `flag_rules.segment_id` depends on them, then delete `segments`.
3. For deleted `environments`: delete already-deleted flags and related rules first, then `environments`.
4. For deleted `projects`: delete already-deleted child environments, flags, segments, and related rules first, then `projects`.

History/audit rows are governed by the separate 5,000-row cap below, not by `deleted_at`.

## History and Audit Tables

### Retention Policy

`flag_versions` and `audit_entries` are capped at **5,000 rows per table** for the MVP. After inserting new history/audit rows, the application prunes the oldest rows beyond the cap. This keeps the self-hosted database small and predictable.

Pruning should run after the mutation transaction commits or in a short background job. It must delete oldest rows by `created_at` and must not block the flag mutation result. The pruning job follows the same local singleflight and Redis distributed mutex rules as soft-delete cleanup. If pruning fails, log the error and retry later.

### flag_versions

Immutable per-version snapshot of a flag aggregate. Used for diff, rollback, audit investigation, and rebuilding snapshot files from committed state. Retained up to the latest 5,000 rows.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `CHAR(36)` | Primary key |
| `flag_id` | `CHAR(36)` | FK to `flags.id` |
| `project_id` | `CHAR(36)` | Denormalized for query |
| `environment_id` | `CHAR(36)` | Denormalized for query |
| `version` | `INT` | Aggregate version |
| `change_type` | `VARCHAR(32)` | `created`, `updated`, `deleted`, `rollback` |
| `snapshot_json` | `JSON` | Full flag payload including rules and rollout |
| `created_by` | `CHAR(36)` | Nullable FK to `users.id` |
| `created_at` | `DATETIME(3)` | |

Recommended indexes:

- `UNIQUE (flag_id, version)`
- `INDEX (project_id, environment_id, created_at)`
- `INDEX (created_by, created_at)`

### audit_entries

Generic actor-attributed mutation log. It can reference flags, segments, environments, projects, or access keys. Retained up to the latest 5,000 rows.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `CHAR(36)` | Primary key |
| `project_id` | `CHAR(36)` | Nullable for global/user events |
| `environment_id` | `CHAR(36)` | Nullable |
| `entity_type` | `VARCHAR(64)` | `flag`, `segment`, `environment`, `project`, `api_key`, `user` |
| `entity_id` | `CHAR(36)` | |
| `action` | `VARCHAR(64)` | `created`, `updated`, `deleted`, `login`, etc. |
| `actor_id` | `CHAR(36)` | Nullable FK to `users.id` |
| `correlation_id` | `VARCHAR(128)` | Nullable |
| `old_value_json` | `JSON` | Nullable |
| `new_value_json` | `JSON` | Nullable |
| `created_at` | `DATETIME(3)` | |

Recommended indexes:

- `INDEX (project_id, environment_id, created_at)`
- `INDEX (entity_type, entity_id, created_at)`
- `INDEX (actor_id, created_at)`
- `INDEX (correlation_id)`

## Runtime and Integration Tables

### api_keys

Lightweight scoped keys for server-side SDKs, WebSocket subscribers, and public/client-side access.

Only store a hash of the key. `key_prefix` is for display and lookup diagnostics.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `CHAR(36)` | Primary key |
| `project_id` | `CHAR(36)` | FK to `projects.id` |
| `environment_id` | `CHAR(36)` | Nullable FK to `environments.id` |
| `name` | `VARCHAR(255)` | |
| `key_prefix` | `VARCHAR(32)` | Non-secret prefix |
| `key_hash` | `CHAR(64)` | SHA-256/HMAC hash |
| `key_type` | `VARCHAR(32)` | `server`, `broadcast`, `public` |
| `scopes_json` | `JSON` | e.g. `["flags:read", "broadcast:subscribe"]` |
| `created_by` | `CHAR(36)` | Nullable FK to `users.id` |
| `last_used_at` | `DATETIME(3)` | Nullable |
| `expires_at` | `DATETIME(3)` | Nullable |
| `revoked_at` | `DATETIME(3)` | Nullable |
| `created_at` | `DATETIME(3)` | |
| `updated_at` | `DATETIME(3)` | |

Recommended indexes:

- `UNIQUE (key_hash)`
- `INDEX (project_id, environment_id, key_type, revoked_at)`
- `INDEX (key_prefix)`

### idempotency_keys

Prevents duplicate effects when dashboard/admin commands are retried.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `CHAR(36)` | Primary key |
| `idempotency_key` | `VARCHAR(128)` | Caller-provided key |
| `actor_id` | `CHAR(36)` | Nullable FK to `users.id` |
| `project_id` | `CHAR(36)` | Nullable |
| `request_hash` | `CHAR(64)` | Hash of method/path/body |
| `status` | `VARCHAR(32)` | `processing`, `completed`, `failed` |
| `response_code` | `INT` | Nullable |
| `response_json` | `JSON` | Nullable |
| `locked_until` | `DATETIME(3)` | Nullable |
| `created_at` | `DATETIME(3)` | |
| `updated_at` | `DATETIME(3)` | |

Recommended indexes:

- `UNIQUE (idempotency_key, actor_id)`
- `INDEX (project_id, created_at)`
- `INDEX (status, locked_until)`

### outbox_events

Transactional event table. Inserted in the same DB transaction as flag mutations.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `CHAR(36)` | Primary key/event ID |
| `aggregate_type` | `VARCHAR(64)` | `flag`, `segment` |
| `aggregate_id` | `CHAR(36)` | |
| `aggregate_version` | `INT` | Version after mutation |
| `scope_version` | `INT` | Environment version after mutation |
| `project_id` | `CHAR(36)` | Denormalized routing key |
| `environment_id` | `CHAR(36)` | Nullable for non-environment events |
| `event_type` | `VARCHAR(128)` | `flag.created`, `flag.updated`, `flag.deleted` |
| `payload_json` | `JSON` | Full broadcast payload needed by consumers |
| `status` | `VARCHAR(32)` | `pending`, `publishing`, `published`, `failed` |
| `retry_count` | `INT` | Default `0` |
| `next_attempt_at` | `DATETIME(3)` | For retry backoff |
| `locked_by` | `VARCHAR(128)` | Nullable dispatcher ID |
| `locked_at` | `DATETIME(3)` | Nullable |
| `last_error` | `TEXT` | Nullable |
| `created_at` | `DATETIME(3)` | |
| `published_at` | `DATETIME(3)` | Nullable |

Recommended indexes:

- `INDEX (status, next_attempt_at, created_at)`
- `INDEX (aggregate_type, aggregate_id, aggregate_version)`
- `INDEX (project_id, environment_id, created_at)`

Dispatcher selection shape:

```sql
SELECT id
FROM outbox_events
WHERE status IN ('pending', 'failed')
  AND (next_attempt_at IS NULL OR next_attempt_at <= ?)
ORDER BY created_at
LIMIT 100
FOR UPDATE SKIP LOCKED;
```

### snapshot_metadata

Tracks generated fallback files. The actual snapshot body lives in pretty-printed JSON files, typically under `/app/snapshot/*.json` for local/self-hosted deployments.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `CHAR(36)` | Primary key |
| `project_id` | `CHAR(36)` | FK to `projects.id` |
| `environment_id` | `CHAR(36)` | FK to `environments.id` |
| `scope_version` | `INT` | Environment version represented by the snapshot |
| `checksum` | `CHAR(64)` | SHA-256 of snapshot file |
| `file_path` | `VARCHAR(1024)` | Local path or object storage key |
| `source_outbox_event_id` | `CHAR(36)` | Nullable FK to `outbox_events.id` |
| `generated_at` | `DATETIME(3)` | |
| `last_validated_at` | `DATETIME(3)` | Nullable |
| `status` | `VARCHAR(32)` | `valid`, `invalid` |

Recommended indexes:

- `UNIQUE (project_id, environment_id)`
- `INDEX (status, generated_at)`
- `INDEX (source_outbox_event_id)`

## Mutation Transaction Pattern

For `create/update/delete flag`, the application should commit the following in one MySQL transaction:

1. Validate actor/session and idempotency key.
2. Lock or optimistic-check the target flag aggregate.
3. Insert/update `flags`.
4. Replace `flag_rules` for the new version, if rules changed.
5. Increment `environments.version` for the scoped project/environment.
6. Insert `flag_versions`.
7. Insert `audit_entries`.
8. Insert `outbox_events` with the full WebSocket payload and new `scope_version`.
9. Mark `idempotency_keys` as completed.
10. Commit.

After commit:

1. Refresh snapshot file from committed state.
2. Update `snapshot_metadata`.
3. Prune `flag_versions` and `audit_entries` rows beyond the latest 5,000 rows per table using the cleanup job concurrency guard.
4. Let the outbox dispatcher publish the event.

If snapshot refresh fails after commit, the flag mutation remains successful. The old snapshot stays valid but stale until the next successful refresh.

## Recommended MVP DDL Order

1. `users`
2. `user_sessions`
3. `schema_migrations`
4. `projects`
5. `environments`
6. `segments`
7. `flags`
8. `flag_rules`
9. `flag_versions`
10. `audit_entries`
11. `api_keys`
12. `idempotency_keys`
13. `outbox_events`
14. `snapshot_metadata`

## Deferred Tables

These should be added later only when the product needs them:

- `experiments`
- `experiment_variants`
- `exposure_events`
- `webhook_endpoints`
- `webhook_deliveries`
- `teams`
- `roles`
- `user_project_roles`
- `approval_requests`
- `flag_owners`
