# Agent Guide

This repo is a Go service for feature flag management. Keep this file short and treat it as universal guidance for every agent session.

## How To Work

- Read the relevant code and local docs before changing behavior. Prefer existing package patterns over new abstractions.
- Keep changes scoped to the requested task. Do not revert unrelated user changes.
- For Go changes, run `gofmt` on touched files and verify with `go test ./...` when practical.

## Skill Routing

Before changing code, route yourself to the relevant skill in `.agents/skills/` and follow it. Do not duplicate skill rules here; the skill file is the source of truth.

- Any new Go code, refactor, bug fix, or behavior change: read `unit-testing-standards/SKILL.md` first.
- Unit tests that need mocks, gomock, NFT data, or shared fixtures: read `unit-testing-standards/SKILL.md` and `testing-mocks-and-nft/SKILL.md`.
- New feature or package layout work: read `project-layout-clean-architecture/SKILL.md`, `domain-modeling/SKILL.md`, and `unit-testing-standards/SKILL.md`.
- Domain models, aggregates, entities, value objects, or DDD boundaries: read `domain-modeling/SKILL.md`.
- HTTP API routes, request/response DTOs, versioning, idempotency, or status behavior: read `flag-gen-api-contract-implementation/SKILL.md`, `go-gin-service-basics/SKILL.md`, and `app-error-handling/SKILL.md`.
- Gin server setup, middleware, CORS/security headers, request limits, or graceful shutdown: read `go-gin-service-basics/SKILL.md`.
- Returning, wrapping, translating, or mapping errors across handlers/usecases/infra: read `app-error-handling/SKILL.md`.
- Flag evaluation rules, rollout bucketing, evaluation metadata, not-found/degraded semantics: read `evaluator-semantics/SKILL.md`.
- Feature flag product guidance, naming, ownership, rollout/rollback, cleanup, or docs/guide/concepts work: read `feature-flag-wiki-best-practices/SKILL.md`.
- Cache refill, snapshot fallback, degraded TTL, snapshot validation, or circuit breaker behavior: read `cache-snapshot-fallback/SKILL.md`, `redis-locks-and-singleflight/SKILL.md`, and `observability-and-correlation/SKILL.md`.
- Redis locks, local singleflight, schedulers, cleanup jobs, migrations/install guards, or cross-replica concurrency: read `redis-locks-and-singleflight/SKILL.md`.
- DB lifecycle status/install/migrate endpoints, schema versioning, or advisory-lock flows: read `db-lifecycle-install-migrate/SKILL.md`, `flag-gen-api-contract-implementation/SKILL.md`, and `app-error-handling/SKILL.md`.
- Outbox tables, dispatcher polling/retry, RabbitMQ broadcast, WebSocket events, or idempotent consumers: read `outbox-and-broadcast/SKILL.md`, `redis-locks-and-singleflight/SKILL.md`, and `observability-and-correlation/SKILL.md`.
- Logging, metrics, correlation IDs, tracing, or runtime observability: read `observability-and-correlation/SKILL.md` and `logmanager-standards/SKILL.md`.
- Gin log middleware, masked headers, logmanager segments, or trace-aware logging: read `logmanager-standards/SKILL.md`.
- Wire providers, injectors, or dependency wiring: read `wire-dependency-injection/SKILL.md`.
- Manual or exploratory QA of the Angular admin (`web/`, base `/app/`): read `web-admin-dashboard-qa/SKILL.md` first (includes non-negotiable UX bar). Product-facing UX summary and requirements: `docs/ui-ux-qa-summary-and-requirements.md`; historical regression notes may live in `docs/ui-ux-regression-report-*.md`.

## Verification Notes

- Prefer focused unit tests for narrow changes, then `go test ./...` before finishing.
- If a task updates generated DI wiring, verify both normal and wire-inject builds if relevant.
- When verification cannot be run, say exactly what was not run and why.

