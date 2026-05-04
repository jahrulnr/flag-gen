# Implementation Checklist (Flag Gen)

Dokumen ini merangkum status implementasi berdasarkan pekerjaan yang sudah dibuat di repo ini. Format checklist:
- `[x]` = sudah ada / sudah jalan
- `[]` = belum

## 1) Core HTTP Service (MVP)
- [x] Gin server bootstrap + middleware dasar (CORS, gzip, helmet, request size limit)
- [x] `X-Correlation-ID` middleware (generate/propagate + header response)
- [x] Error response mapping konsisten (`error`, `message`, `details?`, `correlationId`, `timestamp`)
- [x] Panic recovery middleware mengikuti format error baru

## 2) Domain + Value Objects (MVP)
- [x] `Scope` (`projectId`, `environmentKey`)
- [x] `FlagValue` boolean/json, `FlagRule` operator equals/not_equals/in/not_in, `RolloutConfig`
- [x] `EvaluationContext` + `EvaluationResult`
- [x] Deterministic rollout bucketing (stable percentage rollout)

## 3) MySQL Persistence (MVP)
- [x] Migrations schema inti (projects, environments, segments, flags, flag_rules, versions, audit, idempotency, outbox, snapshot_metadata)
- [x] MySQL repositories: projects, environments, segments
- [x] MySQL flag repository: list/get/create/update/soft-delete + rules load/replace
- [x] Snapshot metadata repository (get/upsert)
- [x] Outbox repository (list, claim/lock, mark published/failed)
- [x] Idempotency repository (begin processing + mark completed/failed)

## 4) Application Usecases (MVP)
- [x] Projects: list, create
- [x] Environments: list, create
- [x] Segments: list, create, update, delete
- [x] Flags: list, get, create, update, delete
- [x] Evaluate flags: cache → DB → snapshot fallback + rule/rollout evaluation

## 5) HTTP Handlers + Router
- [x] Handlers: projects, environments, segments, flags, evaluate
- [x] Router `/v1` registration
- [x] Routes aktif:
  - [x] `GET/POST /v1/projects`
  - [x] `GET/POST /v1/projects/:projectId/environments`
  - [x] `GET/POST/PATCH/DELETE /v1/projects/:projectId/segments`
  - [x] `GET/POST/GET/PATCH/DELETE /v1/projects/:projectId/environments/:environmentKey/flags`
  - [x] `POST /v1/projects/:projectId/environments/:environmentKey/flags/evaluate`

## 6) Audit + Versioning + Outbox Insert (Atomik)
- [x] Flag create: insert `flag_versions`, `audit_entries`, `outbox_events` dalam 1 transaction
- [x] Flag update: insert `flag_versions`, `audit_entries`, `outbox_events` dalam 1 transaction
- [x] Flag delete: insert `audit_entries`, `outbox_events` dalam 1 transaction
- [x] Correlation ID masuk ke `context.Context` dan dicatat di audit/outbox payload

## 7) Outbox Dispatcher Worker (RabbitMQ)
- [x] Config RabbitMQ + env loader (`RABBITMQ_AMQP_URL`, `RABBITMQ_EXCHANGE`, timeout)
- [x] Config outbox worker (`OUTBOX_WORKER_ENABLED`, interval, batch size, lock-for)
- [x] AMQP publisher (declare exchange topic + publish persistent)
- [x] Claim/lock semantic (multi-instance safe dengan lock expiry)
- [x] Dispatcher loop publish + retry/backoff + mark published/failed
- [x] Wiring DI + start goroutine di `cmd/app/main.go` saat enabled

## 8) Tests
- [x] `go test ./...` passing
- [x] Update mocks/tests yang terdampak perubahan interface
- [x] Dispatcher unit tests (success/failure + backoff)

## 9) Local Services (/apps/services)
- [x] RabbitMQ sudah tersedia di `/apps/services/compose.yml` (service `rabbitmq`, ports 5672/15672, volumes, network `services`)

## 10) Backlog / Next Steps
- [] Database lifecycle guard: detect DB empty vs installed, schema compatibility check, block dashboard access when migrations pending
- [] Install flow: backend-only DB config, show install page only on connected empty DB
- [] Migration blocking page + non-blocking drift notification (frontend)
- [] Go migrator shared package/registry + separate `/app/server` and `/app/migrate` binary approach
- [] Expose schema status endpoint (current schema version, target schema version, compatibility) for frontend gating

- [] WebSocket server: send full scoped flag set on connect (initial burst), then committed updates only
- [] WebSocket runtime scope checks (project/environment keys) — do not rely on port separation
- [] Idempotent consumers for broadcasts (dedupe by event ID + aggregate version) on subscriber side

- [] Cache behavior: negative cache `flag not found` marker in L1 (5 minutes)
- [] Circuit breaker when L1/L2 empty + DB down + snapshot invalid (return 503, avoid hammering)
- [] Redis reconnect strategy: event-driven reconnect, mutex/singleflight guard, backoff + metrics
- [] Snapshot scheduler: periodic DB/snapshot consistency checks + after DB reconnect recovery refresh
- [] Snapshot consistency guard: local singleflight + Redis distributed mutex

- [] Soft delete retention cleanup job (retain 3 months, then hard delete)
- [] Cleanup + pruning job guards (singleflight + Redis distributed lock)
- [] Cap history/audit tables (target 5,000 rows each) + pruning strategy

- [] Dashboard query/read models (CQRS-ish): list/detail/history/diff/rollback optimized queries
- [] Endpoint: flag version history + diff/compare support
- [] Endpoint: rollback flag to a prior version

- [] Angular admin dashboard (GrowthBook-inspired) — scaffold app + layout utama
- [] Angular auth pages (login/session) atau dev bypass untuk local
- [] Angular views: Projects (list/create)
- [] Angular views: Environments (list/create per project)
- [] Angular views: Flags (list/create/edit/delete) + rule builder minimal + rollout editor
- [] Angular views: Segments (list/create/edit/delete) + segment rule JSON editor minimal
- [] Angular integration: panggil API `/v1/...` + error handling + correlation-id display
- [] Angular realtime: subscribe perubahan flag via WebSocket (optional; setelah broadcaster siap)
- [] Angular UX parity: naming/structure mengikuti GrowthBook (minimalkan relearning)

- [] WebSocket broadcaster untuk service-to-service subscribe (consume RabbitMQ → push ke WS clients)
- [] WebSocket endpoint `/v1/broadcast/flags` + auth token (broadcast scope)
- [] RabbitMQ consumer untuk broadcast worker (bind ke exchange `flaggen.events`, routingKey `flag.*`)
- [] Event contract stabilization: header `x-event-id` + payload minimal untuk invalidation cache

- [] Endpoint single-flag evaluate (`GET .../flags/:flagKey/evaluate?userId=...`)
- [] Endpoint list/get flag versions (`GET /v1/.../flags/:flagId/versions`)
- [] Endpoint audit log query (`GET /v1/.../audit?entityType=flag&entityId=...`)
- [] Browser/client-side read-only endpoints (public flag reads) + exposure rules
- [] SDK support: Go server-side SDK (local read state + reconnect/poll fallback)
- [] Webhook support: delivery worker + retry policy + signature (deferred until needed)

- [] OpenAPI/Swagger untuk semua endpoint `/v1` + contoh curl untuk Angular dashboard
- [] Documentation: quickstart local (env vars, migrate, run app, run dispatcher)

- [] Auth (dev-mode → API key / OAuth) + scope enforcement
- [] API keys: generate/revoke/rotate + scope enforcement (`flags:read/write/evaluate`, `segments:*`, `broadcast:subscribe`)
- [] Rate limiting per API key (evaluation endpoints)
- [] RBAC user management (deferred; kalau dashboard butuh multi user)
- [] Dashboard authentication: username/password + secure sessions (+ optional 2FA)
- [] Transport security: TLS configuration + deployment docs

- [] Snapshot writer + snapshot metadata update otomatis (generate snapshot dari state committed, bukan manual)
- [] Snapshot generation trigger: on outbox published (or periodic) + checksum + validate
- [] Snapshot read-through hardening: mark invalid snapshot_metadata saat checksum mismatch

- [] Optimize N+1 queries untuk load rules (flags list)
- [] Optimize DB queries: join flags + rules batch load per scope
- [] Cache invalidation: invalidate `ScopedFlagCache` by scope on outbox events

- [] Observability metrics (outbox lag, publish retry count, evaluation latency)
- [] Metrics: outbox pending count, retry count, publish latency, dispatcher loop lag
- [] Tracing/logging: include correlationId in dispatcher logs + publish errors
- [] Alert signals: DB failover activation, cache degradation, file corruption, broker publish failures

