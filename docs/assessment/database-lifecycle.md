# Database Lifecycle, Install, and Migration Guard

## Scope

This document defines how the application guards database setup and schema compatibility from the application side.

The frontend never receives or edits database connection configuration. Database host, port, user, password, database name, and TLS settings are provided through backend environment variables or deployment secrets.

## Startup Classification

On startup and on system status checks, the backend classifies database state into one of these states:

| State | Meaning | Backend behavior | Frontend behavior |
| --- | --- | --- | --- |
| `db_unreachable` | Config exists, but the backend cannot connect to MySQL | API returns degraded/system error; no install page | Show database connection error page, no DB config form |
| `db_empty` | Backend can connect, but required flag-gen tables do not exist | Allow install/bootstrap flow | Show install page |
| `db_ready` | Schema exists and is compatible with current app version | Serve normal API | Allow normal dashboard access |
| `migration_optional` | DB schema is behind but still compatible | Serve normal API with warning metadata | Show non-blocking migration notification |
| `migration_required` | DB schema is behind and current app requires migration before use | Block protected dashboard/API writes and incompatible reads | Show blocking migration popup/page |
| `db_ahead` | DB schema version is newer than this app binary supports | Block app access to avoid data corruption | Show incompatible app/database version error |
| `db_invalid` | Schema tables exist but metadata is missing/corrupted | Block app access except diagnostics | Show repair/manual intervention message |

## Install Flow

The install page is shown only when all conditions are true:

1. Backend can connect to MySQL using environment-provided DB config.
2. No required flag-gen tables exist yet.
3. Backend reports `db_empty` from the system status endpoint.

The install page must not ask for database credentials. It can collect only application bootstrap data, such as:

- Initial admin email
- Initial admin name
- Initial admin password
- Optional project/environment seed values

Install action should:

1. Acquire a database advisory lock, for example `GET_LOCK('flag_gen_install', timeout)`.
2. Re-check that the database is still empty.
3. Run baseline schema creation.
4. Insert schema metadata into `schema_migrations`.
5. Create the initial admin user.
6. Insert audit entry for installation.
7. Release the lock.

If two browser sessions attempt install concurrently, only one should succeed. The other should reload status and continue to login/dashboard.

## Schema Version Metadata

The application must know:

- Current app version, e.g. `1.1.0`
- Current supported schema version range
- Latest schema version required by this app
- Current database schema version
- Whether the current database version is compatible, optionally compatible, or blocked

Recommended backend constants:

```text
APP_VERSION=1.1.0
SCHEMA_MIN_COMPATIBLE_VERSION=202605010001
SCHEMA_TARGET_VERSION=202605010003
SCHEMA_BLOCKING_BELOW_VERSION=202605010002
```

Meaning:

- If DB version is below `SCHEMA_BLOCKING_BELOW_VERSION`, migration is required.
- If DB version is below `SCHEMA_TARGET_VERSION` but at or above `SCHEMA_BLOCKING_BELOW_VERSION`, migration is optional but recommended.
- If DB version is above `SCHEMA_TARGET_VERSION`, the app is older than the database and must block access.

## Migration Metadata Table

`schema_migrations` stores applied migration history and is the first table checked after connectivity.

| Column | Type | Notes |
| --- | --- | --- |
| `version` | `BIGINT` | Primary key; timestamp-style migration version |
| `name` | `VARCHAR(255)` | Human-readable migration name |
| `checksum` | `CHAR(64)` | Checksum of migration file/content |
| `execution_ms` | `INT` | Runtime duration |
| `applied_by_app_version` | `VARCHAR(64)` | App version that applied it |
| `applied_at` | `DATETIME(3)` | UTC |
| `dirty` | `BOOLEAN` | True only if a migration failed mid-run |

Recommended indexes:

- `PRIMARY KEY (version)`
- `INDEX (dirty)`
- `INDEX (applied_at)`

If `dirty = true`, backend reports `db_invalid` and blocks normal access.

## Migration Framework Runtime

The migration framework is implemented in Go and lives in the same repository as the main application, but it is exposed through a separate migrator binary.

Recommended deployment layout:

```text
/app/server             # main flag-gen HTTP/WebSocket server
/app/migrate            # Go-based schema migrator binary
/app/snapshot/*.json    # file-backed flag snapshots
```

The server and migrator should share one internal migration package so compatibility rules, migration discovery, checksum validation, and execution behavior cannot drift between binaries.

Recommended shape:

- **Main server binary**: owns HTTP/WebSocket runtime, system status endpoint, install page support, and dashboard-triggered migration action.
- **Migrator binary**: owns direct schema migration execution for CI, local development, container jobs, and operational recovery.
- **Shared Go migrator package**: owns migration registry, optional/required compatibility rules, advisory locking, dirty-state handling, checksum validation, and migration execution.

This keeps runtime serving and schema mutation operationally separate while still allowing the dashboard to trigger the same migrator logic through `POST /v1/system/migrate`.

Schema compatibility is defined in the Go migrator registry for each application version. The registry decides whether a pending migration is optional or required for the currently running server.

## Migration Flow

Migration can be initiated from the dashboard only when backend reports `migration_optional` or `migration_required`.

Migration action should:

1. Require authenticated dashboard admin session.
2. Acquire a database advisory lock, for example `GET_LOCK('flag_gen_migrate', timeout)`.
3. Re-check current schema state.
4. Run pending migrations in order.
5. Record each migration in `schema_migrations`.
6. Clear dirty state on success.
7. Release the lock.
8. Return new system status.

If migration fails:

- Record failed/dirty state when the migration framework supports it.
- Return a clear error and correlation ID.
- Keep protected UI blocked if the migration is required.
- Do not attempt automatic destructive rollback unless the migration explicitly supports it.

## Frontend Behavior

The frontend should call a system status endpoint before loading protected dashboard routes.

Recommended route handling:

- `db_unreachable`: show database unavailable page.
- `db_empty`: redirect to `/install`.
- `db_ready`: allow dashboard.
- `migration_optional`: allow dashboard and show notification/banner.
- `migration_required`: show blocking migration modal/page; hide or disable dashboard menus.
- `db_ahead`: show app/database version mismatch page.
- `db_invalid`: show repair/manual intervention page.

The frontend should not decide compatibility itself. It only follows backend status fields.

## API Behavior During Migration States

When migration is required:

- `GET /v1/system/status` remains available.
- `POST /v1/system/migrate` remains available to authorized admins.
- Login/session endpoints remain available if their tables are compatible.
- Protected dashboard menus and mutation APIs should return `423 Locked` or `503 Service Unavailable` with status details.
- Runtime WebSocket/SDK reads may continue only if backend marks the schema as compatible for runtime reads.

When migration is optional:

- APIs continue to work.
- Responses may include headers such as `X-Schema-Migration-State: optional`.
- Frontend shows a non-blocking migration notification.

## Compatibility Policy

Each app version should define which features require which schema version inside the Go migrator registry.

Example:

| Feature | Required schema version | Blocking |
| --- | --- | --- |
| Core flag CRUD | `202605010001` | Yes |
| `client_side` flag exposure | `202605010002` | Yes if client-side API enabled |
| Snapshot metadata status | `202605010003` | No if runtime can use previous snapshot metadata shape |

This lets the app show a warning for compatible drift while blocking only when data safety or required runtime behavior depends on the new schema.

The optional/required decision is not inferred by the frontend and is not decided from SQL filenames alone. It is part of the migrator integration contract.

## Operational Rules

- Never auto-run migrations silently on normal app startup in production.
- Allow explicit migration from dashboard/admin only after status check.
- Allow auto-migration only in local development or test mode if configured.
- Migrations must be forward-only by default.
- Destructive migrations require explicit review and backup guidance.
- Back up production databases before required migrations.
