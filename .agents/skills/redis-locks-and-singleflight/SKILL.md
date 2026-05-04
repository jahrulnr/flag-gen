---
name: redis-locks-and-singleflight
description: Defines per-scope singleflight keys and Redis distributed mutex rules (token, TTL, renewal, compare-and-delete unlock). Use when implementing cache refill, snapshot checks, cleanup jobs, migrations/install guards, or any cross-replica concurrency control using Redis.
---

# Redis Locks + Singleflight (Concurrency Standards)

## Scope key (canonical)

Use one consistent scope key string everywhere (cache keys, singleflight keys, lock keys):

- `scopeKey = "project:" + projectId + ":env:" + environmentKey`

## Local singleflight

Use **per-scope** singleflight for:

- L1/L2 refill
- snapshot file reads that can stampede
- snapshot refresh triggers

Avoid per-flag/per-key singleflight for “full scope read model” caching; use per-scope.

## Redis distributed mutex (cross-replica)

Use Redis locks when multiple replicas might run the same job/work concurrently:

- snapshot consistency check
- cleanup/pruning jobs
- any “rebuild” or “refresh” task that should be singleton

### Lock requirements

- **Owner token**: random unique value per lock holder.
- **TTL**: always set expiry; never infinite locks.
- **Renewal**: long jobs must extend TTL safely.
- **Compare-and-delete unlock**: only release if token matches (never delete another owner’s lock).

### Lock key convention

- `flag-gen:lock:<purpose>:{scopeKey}`
  - Examples:
    - `flag-gen:lock:snapshot-check:project:proj-1:env:production`
    - `flag-gen:lock:cleanup:global`

### Degraded behavior

If Redis is unavailable:

- Do not block runtime reads.
- Skip scheduled maintenance jobs and retry later.

## Relationship to DB advisory locks

Use MySQL advisory locks for install/migrate (per DB lifecycle docs), not Redis locks.

