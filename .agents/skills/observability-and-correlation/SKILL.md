---
name: observability-and-correlation
description: Standardizes correlation IDs, logging, and minimal observability signals across handler/usecase/infra. Use when implementing HTTP middleware, error responses, logging with request context, metrics naming, or when the user mentions correlation id, tracing, logging, monitoring, or observability.
---

# Observability + Correlation ID

## Goal

Make failures diagnosable quickly by ensuring every request and error is traceable end-to-end.

## Correlation ID rules

- Accept optional `X-Correlation-ID` from client.
- If missing, generate one at the edge (HTTP middleware).
- Propagate via `context.Context`:
  - handler → usecase → infrastructure
- Include correlation id in:
  - logs
  - error responses (API contract)
  - outbox events (optional but recommended)

## Logging rules

- Log at boundaries:
  - handler entry/exit (method, path, status, latency)
  - usecase errors (domain/app/infra classification)
  - infrastructure failures (DB/Redis/HTTP client)
- Never log secrets; respect masking rules.
- When using `apperror`, log `Caused()` (full chain) but expose safe `Error()` to clients.

## Minimal metrics (recommended)

Emit counters/timers (exact backend depends on chosen stack):

- request count + latency per endpoint
- cache hit/miss per tier (L1/L2)
- snapshot fallback activations
- circuit breaker transitions (open/half-open/closed)
- outbox pending/publish retries/failures
- redis reconnect attempts/success/fail

## Error envelope alignment

HTTP error responses should include:

- `correlationId`
- timestamp
- stable error code/message (per API contract)

Tie handler mapping to `apperror.IAppError.StatusCode()`.

