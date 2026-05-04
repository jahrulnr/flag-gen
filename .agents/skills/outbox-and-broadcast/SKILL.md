---
name: outbox-and-broadcast
description: Implements transactional outbox and WebSocket broadcast flows per docs/assessment/sequence-diagram.puml and database-schema.md. Use when adding outbox tables, dispatcher polling/retry, idempotent consumers, and flags.snapshot/flag.updated/flag.deleted messaging.
---

# Transactional Outbox + Broadcast (MVP)

## Source of truth

- `docs/assessment/sequence-diagram.puml`
- `docs/assessment/database-schema.md` (`outbox_events`)
- `docs/assessment/api-contract.md` (WebSocket message shapes)

## Commit boundary

In one DB transaction for flag mutations, commit:

- flag state
- flag_versions
- audit_entries
- outbox_events (with full payload)

Publishing is async and retryable; do not roll back committed flag state if publish fails.

## Dispatcher rules

- Poll `outbox_events` pending/failed with backoff (`next_attempt_at`)
- Use row locking (`FOR UPDATE SKIP LOCKED`) when supported
- Mark published after broker/ws ack

## Consumer rules (idempotency)

- Deduplicate by `eventId`
- Ignore stale `aggregateVersion` / `scopeVersion`

## Snapshot refresh ordering

- Refresh snapshot **after commit** (never before)
- Snapshot failures do not invalidate the successful mutation; they leave snapshot stale

