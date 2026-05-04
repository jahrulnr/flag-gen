# Snapshot File Format and Semantics

## Purpose

Snapshot files are **last-known-good** scoped flag state used as a **read-through availability fallback** when MySQL is unavailable (and optionally when Redis is unavailable or misses). They are **not** co-primary persistence and are **not** treated as a normal cache tier.

Snapshot files are generated only from **committed database state** (post-commit). Failed or partial mutations must not advance snapshot files.

## Scope Key and Naming

A snapshot represents exactly one scope:

- **projectId**
- **environmentKey** (or environmentId; docs currently use `environmentKey` in APIs)

Recommended canonical scope key string (used for file naming, singleflight keys, etc.):

- `scopeKey = "project:" + projectId + ":env:" + environmentKey`

Recommended file name:

- `{projectId}__{environmentKey}.snapshot.json`

Recommended directory:

- `/app/snapshot/` for self-hosted runtime

## Canonical JSON Structure (v1)

The snapshot JSON is pretty-printed for operability. It includes a format version so future schema changes are safe.

```json
{
  "formatVersion": 1,
  "projectId": "proj-1",
  "environmentKey": "production",
  "scopeVersion": 42,
  "generatedAt": "2026-04-28T10:05:00Z",
  "checksum": {
    "algo": "sha256",
    "hex": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "covers": "canonicalPayload"
  },
  "flags": [
    {
      "id": "flag-123",
      "key": "checkout_v2",
      "enabled": true,
      "clientSide": false,
      "value": { "type": "boolean", "default": true },
      "rules": [],
      "rollout": { "percentage": 10, "stickiness": "userId" },
      "version": 2,
      "deletedAt": null
    }
  ]
}
```

### Field Rules

- **`formatVersion`**: integer, required. Starts at `1`.
- **`projectId` / `environmentKey`**: required; must match the requested scope.
- **`scopeVersion`**: required; must match the environment scope version in MySQL at the time of generation.
- **`generatedAt`**: required RFC3339 UTC timestamp.
- **`flags`**: required array of **active** flag payloads for the scope.
  - Soft-deleted flags should not appear in `flags` (or must appear with `deletedAt` set and be excluded by consumers). Prefer exclusion for simplicity.
- **`checksum`**: required.
  - `covers = "canonicalPayload"` means the checksum is computed over the JSON payload excluding the `checksum` field itself (and excluding whitespace/pretty formatting).

## Checksum Algorithm

To keep checksum stable across pretty-print differences:

- Build a **canonical JSON byte sequence** by serializing the snapshot object **without** the `checksum` field using:
  - stable key ordering
  - no insignificant whitespace
- Compute SHA-256 over that canonical byte sequence.
- Store as lowercase hex string in `checksum.hex`.

If canonicalization is too heavy for MVP, acceptable alternative:

- Compute SHA-256 over the raw bytes of the file **excluding** the `checksum` field (still requires canonical exclusion).

## Read Semantics

### When the Service Reads Snapshot Files

Snapshot fallback is used only when **DB reads cannot serve** (e.g., DB down). The standard runtime read order remains:

- L1 (Ristretto) → L2 (Redis) → DB (MySQL) → **Snapshot file**

### Validity Checks on Read

When reading a snapshot file, treat it as **valid** only if:

- JSON parses successfully
- `formatVersion` is supported
- `projectId` / `environmentKey` match requested scope
- `checksum` verifies

If any check fails, the snapshot is **invalid** and the system should follow the degraded behavior described in the sequence diagram and API contract (potentially returning `503 service_degraded` for remote reads).

## Write Semantics (Generation and Refresh)

### When to Generate/Refresh

1. **After commit** of `create/update/delete flag` flows, generate a new snapshot from committed DB state.
2. **Lazy creation**: when DB is available and snapshot file is missing, the service may create the snapshot (migration safety).

### Atomic Write Strategy

To avoid partially written snapshots:

1. Write to a temp file in the same directory:
   - `{projectId}__{environmentKey}.snapshot.json.tmp`
2. `fsync` temp file
3. Atomic rename to final file name
4. Optionally `fsync` the directory

### Concurrency Rules

- **Local singleflight**: Only one snapshot refresh per scope runs concurrently in a process.
- **Cross-replica lock (optional but recommended)**: When multiple replicas can write to the same filesystem/object store location, guard with a Redis distributed mutex per scope.

## Cache Fill From Snapshot (Degraded TTL)

When a snapshot is used because **DB is down** and both caches miss:

- Fill **L1** from snapshot data using a **short TTL** (e.g., **1 hour**).
- If Redis is reachable, optionally fill **L2** with the same short TTL.

This prevents long-lived cache entries sourced from potentially stale file-backed state.

## Relationship to `snapshot_metadata`

MySQL table `snapshot_metadata` tracks the most recent known snapshot for a scope:

- `scope_version`, `checksum`, `file_path`, `generated_at`, `status`

The file is the availability artifact; DB metadata is for observability and operational checks. If metadata update fails post-commit, the snapshot file may still be valid; treat metadata as advisory.

