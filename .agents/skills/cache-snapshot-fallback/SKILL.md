---
name: cache-snapshot-fallback
description: Implements L1/L2 cache + snapshot fallback behavior per docs/assessment/plan.md, technologies.md, snapshot-format.md, and sequence-diagram.puml. Use when implementing singleflight refill, degraded TTL cache fill from file, snapshot validation/atomic write, and circuit breaker behavior.
---

# Cache + Snapshot Fallback (MVP)

## Source of truth

- `docs/assessment/plan.md` (failover rules)
- `docs/assessment/technologies.md` (runtime read path)
- `docs/assessment/sequence-diagram.puml` (exact flows)
- `docs/assessment/snapshot-format.md` (file format + atomic write)

## Read path

L1 (Ristretto) → L2 (Redis) → DB (MySQL) → Snapshot file

## Singleflight rules

- Use **per-scope** singleflight keys (projectId + environmentKey) for refills.
- Protect against stampede to Redis, DB, and filesystem reads.

## Degraded cache fill from file

When **DB is down** + **L2 miss** + **L1 miss** and snapshot is valid:

- Fill **L1** from snapshot with **short TTL (e.g., 1 hour)**.
- If Redis reachable, optionally fill **L2** with the same short TTL.

## Snapshot safety

- Validate `formatVersion`, scope match, and checksum before use.
- Write snapshots atomically (temp file → fsync → rename).

## Circuit breaker

If caches miss and DB unavailable and snapshot missing/invalid:

- Return `503 service_degraded` for remote reads
- Open circuit breaker for the affected scope/dependency (per docs)

