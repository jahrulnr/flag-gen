---
name: go-gin-service-basics
description: Guides Go Gin service setup consistent with this repo (cmd/app/main.go, delivery/http/middleware). Use when adding routes, middleware, graceful shutdown, request size limits, CORS/helmet/gzip, or standardizing error responses in Gin.
---

# Go Gin Service Basics (repo-aligned)

## Repo anchors

- Entry point: `cmd/app/main.go`
- Middleware: `delivery/http/middleware/*`

## When adding new HTTP routes

- Register routes in one place (router wiring), keep handler logic in dedicated packages (evolve toward `delivery/http/handler/...`).
- Prefer route grouping under `/v1` for APIs and keep `/health` as simple liveness.
- Keep request body limits and common middleware applied globally (already done in `main.go`).

## Middleware rules

- Middleware should not embed business logic; it can enrich context (correlation id), enforce auth, validate headers, and map panics to JSON errors.
- Use one consistent JSON error envelope across the service. Avoid ad-hoc shapes per handler.

## Error response convention (recommended)

For non-2xx responses, return:

```json
{
  "error": "string",
  "message": "string",
  "details": {},
  "correlationId": "string",
  "timestamp": "datetime"
}
```

Align with `docs/assessment/api-contract.md` when implementing `/v1`.

## Graceful shutdown

- Keep signal handling and `srv.Shutdown(ctx)` pattern.
- Any background workers (outbox dispatcher, redis reconnect loop, etc.) should honor context cancellation and stop within timeout.

## Security/operational defaults

- Keep masking config for sensitive headers (already in `main.go`).
- Keep TLS config in server setup when `USE_SSL=true`.

