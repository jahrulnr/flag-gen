---
name: flag-gen-api-contract-implementation
description: Implements the MVP HTTP and WebSocket API contract from docs/assessment/api-contract.md in Go. Use when creating /v1 routes, request/response DTOs, idempotency/versioning handling, and standardized error/status behavior.
---

# Flag-gen API Contract Implementation (MVP)

## Source of truth

- `docs/assessment/api-contract.md`
- `docs/assessment/sequence-diagram.puml` (runtime read path + degraded behavior)

## Implementation workflow

1. **Create delivery layer routing**
   - Group under `/v1`.
   - Keep `/health` separate.
2. **Define DTOs in delivery layer**
   - DTOs may use JSON tags.
   - Do not reuse domain structs as DTOs (domain is struct-only, no JSON aliasing).
3. **Map DTO → application commands/queries**
   - Validation of transport shape in delivery.
   - Business invariants in domain.
4. **Return contract-shaped responses**
   - Include `correlationId` consistently.
   - Use required status codes (`400/401/403/404/409/422/429/500/503`).

## Cross-cutting rules

- **Idempotency**: create/update/delete commands require `Idempotency-Key`.
- **Optimistic locking**: accept `version` or `If-Match`.
- **No silent default**: if L1/L2 miss + DB unavailable + snapshot invalid/missing, remote reads must return `503 service_degraded` unless explicitly documented otherwise.

## WebSocket broadcast endpoints

- Implement `/v1/broadcast/flags` subscription with:
  - initial `flags.snapshot`
  - later `flag.updated` / `flag.deleted`
- Authorization is mandatory (internal runtime traffic).

## Suggested packages (evolve from current repo)

- `delivery/http/handler/...` for controllers
- `delivery/http/dto/...` for request/response shapes
- `application/...` for usecases

