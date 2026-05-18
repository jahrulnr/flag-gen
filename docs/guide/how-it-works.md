# How Flag Gen Works

This document explains runtime behavior for integrators. For API field-level detail, see [assessment/api-contract.md](assessment/api-contract.md). For evaluation rules, see [assessment/evaluation-semantics.md](assessment/evaluation-semantics.md).

## Two integration flows

Most applications need **both**, but they are separate:

| Flow | What you get | Typical transport | Cached per user? |
|------|----------------|-------------------|------------------|
| **Flag definitions** | `enabled`, `value`, rules, rollout, `version` | WebSocket (`flags.snapshot` + deltas) or `GET .../flags` | No — one definition set per project/environment |
| **Evaluate** | Resolved `value`, `reason`, `version`, `source` for a **context** | `POST .../flags/evaluate` (or GET single-flag) | No — computed per request from definitions + `userId` + `attributes` |

```text
Dashboard edit
      │
      ▼
 MySQL (source of truth) ──► L1/L2 cache refill
      │
      ├── outbox ──► RabbitMQ ──► WebSocket hub ──► backend SDK local cache
      │
      └── (optional) snapshot file refresh

Backend request with userId + attributes
      │
      ▼
 POST /flags/evaluate  ──► reads definitions from cache/DB ──► EvaluationResult
```

**Do not** use `Variation()` / local cache alone for per-user targeting — it only applies the flag default when `enabled` is true. Use **evaluate** when rules or rollout depend on user attributes.

## Flag definition sync (server-side)

### WebSocket (recommended for Go services)

1. Service dials `GET /v1/broadcast/flags?projectId=&environmentKey=&token=` (dev token or API key scope `broadcast:subscribe`).
2. On connect, flag-gen sends **one** message `flags.snapshot` with **all flags** for that scope (from scoped cache, not a full-table scan per client globally).
3. Later, only **deltas**: `flag.created`, `flag.updated`, `flag.deleted`.

Reconnect after a disconnect receives a fresh `flags.snapshot`.

### HTTP list (bootstrap or fallback)

`GET /v1/projects/{id}/environments/{key}/flags` loads the same definition set once. The Go example may call this at startup; with a working WebSocket, the snapshot message is usually enough.

Optional polling (`FLAG_GEN_POLL_FALLBACK=true` in examples) is only for when RabbitMQ or WebSocket is unavailable — not the primary design.

## Live updates

For dashboard changes to reach running services:

```text
PATCH flag (transaction)
  → outbox row
  → OUTBOX_WORKER publishes to RabbitMQ
  → BROADCAST_CONSUMER fans out to WebSocket hub for that scope
  → each subscribed connection gets flag.updated
```

Required env (local): `OUTBOX_WORKER_ENABLED=true`, `BROADCAST_CONSUMER_ENABLED=true`, RabbitMQ reachable, and matching `BROADCAST_DEV_TOKEN` on server and SDK when set.

## Evaluate path

1. Caller sends `context`: at least `userId`; optional `attributes` (plan, country, etc.).
2. Service loads flag definition (L1 Ristretto → L2 Redis → MySQL → snapshot file).
3. Evaluator applies: disabled → default; rules by priority; else rollout bucketing; else default.
4. Response includes `reason` (`disabled`, `rule_match`, `rollout`, `default`, …) and `source` (`cache_l1`, `database`, `snapshot`, …).

`source: cache_l1` means the **flag definition** was read from cache, not that the evaluate result is cached for that user.

## Read resilience (flag-gen server)

When serving definitions or evaluate:

```text
L1 (in-process) → L2 (Redis) → MySQL → validated snapshot file → 503 degraded
```

Snapshot files are last-known-good fallback, not a normal third cache tier. See [assessment/plan.md](assessment/plan.md) and [snapshot-format.md](assessment/snapshot-format.md).

## Public / client-side API

Browser and mobile should use `/v1/client/...` or `/v1/public/...` routes so only `clientSide` flags and reduced payloads are exposed. Server-side services use `/v1/projects/...` and WebSocket broadcast.

## Related docs

- SDK usage: [sdk/sdk-flagclient_id.md](sdk/sdk-flagclient_id.md)
- Examples: [../../examples/](../../examples/)
- Best practices: [concepts/feature-flags_id.md](concepts/feature-flags_id.md)
