---
name: logmanager-standards
description: Standardizes usage of github.com/SALT-Indonesia/salt-pkg/logmanager (and lmgin). Use when wiring Gin logging middleware, masking sensitive headers, creating API/DB/other segments, propagating trace IDs, or logging errors with context.
---

# Logmanager Standards

## App logging setup (masking + exposed headers)

Use a single `setupLogging()` that creates `*logmanager.Application` with:

- `WithAppName("flag-gen")`
- `WithExposeHeaders("Referer", "X-Forwarded-For", "User-Agent")`
- masking config for sensitive headers (Authorization, tokens, signatures, msisdn, etc.)

Then attach Gin middleware:

- `router.Use(lmgin.Middleware(appLog))`

## Context propagation rule

- Treat `context.Context` as the carrier of logmanager transaction.
- Always call `logmanager.FromContext(ctx)` in infra/usecase to get the current txn record when available.

## API segment (outbound HTTP calls)

When infra calls external HTTP services:

1. Build request with `http.NewRequestWithContext(ctx, ...)`
2. Start segment:
   - `logTx := logmanager.StartApiSegment(logmanager.ApiSegment{Request: req, Name: "<Client>.<Operation>"})`
   - `defer logTx.End()`
3. After `Do(req)`:
   - on error → `logTx.NoticeError(err)` then return an `apperror` (infra error)
4. On success:
   - `logTx.SetResponse(resp)`
   - read body; on failures use `logTx.NoticeError(err)`

## Database segment

Wrap significant DB queries/mutations:

- `txn := logmanager.StartDatabaseSegment(logmanager.FromContext(ctx), logmanager.DatabaseSegment{Query: q, Table: "<table>"})`
- `defer txn.End()`
- on error → `txn.NoticeError(err)`

## Other segment (background/side effects)

For “non-HTTP non-DB” operations (publish to queue, transformations, etc.):

- `txn := logmanager.StartOtherSegment(logmanager.FromContext(ctx), logmanager.OtherSegment{Name: "<Operation>"})`
- `defer txn.End()`
- on error → `txn.NoticeError(err)`

## Trace ID propagation to downstream services

Follow wec-order-omni pattern:

- If `logmanager.FromContext(ctx) != nil`, get trace id via `TraceID()`.
- Put it into outbound headers (example key used there: `x-trace-id`).

Use one consistent header key across this repo for downstream propagation.

## Error logging rule of thumb

- In handler/usecase/infra, log with context:
  - `logmanager.ErrorWithContext(ctx, err)` for immediate errors
- For segment-scoped operations:
  - prefer `txn.NoticeError(err)` so error attaches to the segment

Pair with `apperror` for classification (see `app-error-handling` skill):

- log detailed cause chain internally
- return safe error message outward

