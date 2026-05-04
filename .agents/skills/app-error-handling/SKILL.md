---
name: app-error-handling
description: Standardizes error handling using pkg/error (apperror) across usecase, handler, and infra (non-domain) so errors are recognizable without brittle Go error mapping. Use when returning errors from usecases, mapping infra failures, translating errors to HTTP responses, or when the user mentions apperror, pkg/error, status code, error mapping, or error types.
---

# App Error Handling (using `pkg/error` / `apperror`)

## Goal

Use `pkg/error/error.go` (`package apperror`) for **usecase**, **handler**, and **infrastructure** (non-domain) so errors:

- carry an HTTP-ish `StatusCode()`,
- preserve a `Cause` for debugging,
- are easy to classify using `errors.As` (vs native Go errors that need extra mapping).

## Source of truth

- `pkg/error/error.go`

## Non-domain usage rule

- **Domain** stays pure business rules (see domain modeling skill).
- **Usecase/handler/infra** may use `apperror` to classify and transport error semantics.

If domain returns typed domain errors, map them in usecase to `apperror.NewDomainError(...)` or `apperror.NewValidationError(...)`.

## Creating errors (preferred helpers)

- Validation/input problems (usecase/handler): `apperror.NewValidationError(apperror.ValidationError, msg, cause)`
- Domain/business rule failures (usecase mapping): `apperror.NewDomainError(apperror.DomainError, msg, cause)` (422)
- Infrastructure failures (DB/Redis/HTTP client): `apperror.NewInfrastructureError(apperror.InfrastructureError, msg, cause)` (502)
- Internal/unexpected: `apperror.NewApplicationError(apperror.ApplicationError, msg, cause)` (500)
- Not found: `apperror.NewNotFoundError(resource, cause)` (404)
- Unauthorized/Forbidden: `apperror.NewUnauthorizedError(...)`, `apperror.NewForbiddenError(...)`

## Wrapping errors (keep the cause)

When you catch an underlying error and want to keep original context:

- Use `CausedBy(err)` to attach/replace the cause while preserving status+message.

Example pattern:

```go
appErr := apperror.NewInfrastructureError(apperror.InfrastructureError, "redis get failed", nil)
return appErr.CausedBy(err)
```

## Detecting app errors (mapping in handler)

In handlers/middleware, map any error to response by:

1. `errors.As(err, &appErr)` to detect `apperror.IAppError`
2. If not an `IAppError`, wrap as `NewApplicationError(...)` (500)

Pseudo:

```go
var appErr apperror.IAppError
if errors.As(err, &appErr) {
  code := appErr.StatusCode()
  // respond with code + safe message, log appErr.Caused() for details
} else {
  appErr = apperror.NewApplicationError(apperror.ApplicationError, "internal error", err)
}
```

## Layer-specific guidance

### Infrastructure layer (non-domain)

- Never return raw driver/client errors upward.
- Convert to `apperror.NewInfrastructureError(...)` (or `NewNotFoundError` when appropriate) and keep the original cause.

### Usecase / application layer

- Validate inputs: return `NewValidationError(...)` or `NewDomainError(...)`.
- Map domain errors → `apperror` at the boundary.
- Do not leak infra-specific details in public message; keep details in `Cause`.

### Handler / delivery layer

- Translate `IAppError.StatusCode()` to HTTP status.
- Public response uses `Error()` (safe message), logging uses `Caused()` for full chain.
- Ensure correlation id / error envelope matches API contract (when implemented).

## Anti-patterns

- Returning `fmt.Errorf(...)` from infra/usecase and expecting handler to guess status code.
- Using domain structs as transport error DTOs.
- Putting `apperror` in domain methods (unless you intentionally treat it as a boundary concern; default: keep domain independent).

