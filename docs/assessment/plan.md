# Feature Flagging Application Development Plan

## Overview

This plan outlines the development of a self-hosted feature flagging service inspired by GrowthBook and Unleash. The assessment focuses first on the core service: flag management, deterministic evaluation, auditability, API contracts, service-to-service broadcasts, and failover behavior.

## Objectives

- Create a self-hosted feature flagging service focused on core flag management
- Support environments, boolean/JSON flag values, simple targeting, and stable percentage rollouts
- Provide an Angular admin dashboard with GrowthBook-inspired interaction patterns
- Use HTTP as the primary dashboard/admin API and client-side/browser flag read API
- Use WebSocket for server-side service-to-service flag state synchronization: send all scoped flags on connect, then broadcast committed flag changes
- Implement robust failover using file-backed snapshots as read-through fallback
- Preserve data consistency for flag mutations with transactional outbox and idempotent consumers
- Provide comprehensive versioning and audit trails for operational tracking
- Implement rich API error tracking with detailed logging for easier debugging
- Design resource-based HTTP APIs for flag, environment, segment, and evaluation operations
- Guard database lifecycle from the application: install only on connected empty DB, detect schema compatibility, and block dashboard access when required migrations are pending
- Defer experimentation, advanced analytics, and AI-powered capabilities until the core flagging service is stable

## Failover and Resilience Mechanisms

To keep the MVP service reliable during runtime failures, the application implements failover strategies based on [sequence-diagram.puml](sequence-diagram.puml) and [technologies.md](technologies.md):

### Snapshot Fallback Strategy

- **Primary**: MySQL database for transactional integrity
- **L1 Cache**: Ristretto in-memory cache in each flag-gen server process for hot scoped flag read models and recently loaded snapshots; recommended TTL is 2 weeks
- **L2 Cache**: Redis shared cache improves cross-replica read performance and provides distributed locks; recommended TTL is 1 year for scoped flag read models, but Redis is not required for runtime availability
- **Fallback**: Pretty-printed JSON snapshot files with version, checksum, and timestamp metadata
- **Fallback Naming**: Snapshot files are not treated as L3 cache in normal operation; they are a last-resort availability fallback when DB/L2 cannot serve reads
- **Read Path**: WebSocket and HTTP runtime reads use L1 -> L2 -> DB -> snapshot fallback, with refill protected by per-scope singleflight so concurrent requests do not stampede Redis, DB, or the filesystem
- **Degraded Cache Fill (From File)**: When **DB is down**, **L2 is available but misses**, and **L1 misses** while a valid snapshot file exists, the service may fill caches from the file snapshot but must use a **short TTL (e.g., 1 hour)** for any cache entries derived from file fallback. This limits the blast radius of serving stale file-backed state as if it were hot cache.
- **Degraded TTL Scope**: The short TTL applies to both **L1** and **L2** entries when they are populated from snapshot fallback (if Redis is reachable). If Redis is unreachable, only L1 is filled.
- **Negative Cache**: HTTP flag reads that confirm `flag not found` cache a not-found marker in L1 for 5 minutes, not in L2 by default
- **Circuit Breaker**: If L1/L2 are empty, DB is down, and no valid snapshot can serve the read, return 5xx/503 and open a circuit breaker to avoid repeatedly hammering DB
- **Availability Purpose**: If MySQL or Redis is unavailable, dashboard recovery reads and runtime services can load the newest valid file snapshot into memory/local cache
- **Simple Mental Model**: A snapshot is the last known-good full flag state for a project/environment, written to a file so the app can still serve reads when DB or Redis is unavailable
- **Storage Layout**: Local deployments store snapshots under `/app/snapshot/*.json`, beside `/app/server` and `/app/migrate`
- **Synchronization**: Snapshot updates are generated from committed database state after create/update/delete workflows fully succeed; snapshots are not co-primary writes
- **Staleness Rule**: A snapshot remains stale by design until the complete mutation flow succeeds, including database commit, version/audit/outbox recording, and the post-commit snapshot refresh step
- **Consistency Check**: There is no routine mass rebuild; a scheduler checks snapshot/database consistency periodically and after database boot or reconnect recovery
- **Consistency Guard**: Snapshot consistency checks use local singleflight and a Redis distributed mutex so repeated triggers and horizontal replicas do not run overlapping checks

### Redis Reconnect Strategy

- **Event-driven Recovery**: Redis connection errors publish an internal in-memory reconnect event instead of reconnecting inline on every failed API call
- **Mutex Guard**: Reconnect is guarded by a mutex/single-flight lock so duplicate events do not start overlapping reconnect attempts
- **Non-blocking APIs**: While reconnect is running, reads can bypass Redis and use DB or the newest valid snapshot; writes still rely on MySQL transaction boundaries
- **Backoff and Logging**: Reconnect attempts use retry backoff, structured logs, and metrics for disconnect/reconnect events
- **Recovery Refresh**: After Redis reconnects, the app can repopulate hot dashboard/read caches from DB or snapshot data

### Transactional Outbox and Idempotent Consumers

- **Atomic Writes**: Flag mutation, flag version record, audit entry, and outbox event are stored in one database transaction
- **Async Publishing**: A background worker publishes committed outbox events to the message broker
- **Retry Safety**: Consumers process events idempotently using event IDs and aggregate versions
- **Failure Behavior**: If message publishing fails, the flag state remains committed and the outbox event is retried

### Lazy File Creation and Migration Safety

- **Read Operations**: Check L1 Ristretto first, then L2 Redis, then DB; if DB is unavailable, load the newest valid snapshot instead of failing hard
- **Snapshot Refresh**: Missing snapshots can be created from committed DB state, but existing snapshots are only advanced after successful create/update/delete completion
- **Migration**: Ensures files exist when moving between self-hosted instances
- **Freshness Tracking**: Files include aggregate version, checksum, and modification timestamps for monitoring

### Safe Defaults and Recovery

- **File Corruption**: On invalid files, load safe defaults to prevent service disruption
- **DB or Redis Failure**: Automatic failover to valid snapshots/memory/local cache with resync on recovery
- **Unserved 5xx**: When all caches miss and DB/snapshot cannot serve the request, activate a circuit breaker for the failing dependency/scope
- **Concurrency Handling**: Optimistic locking prevents data corruption in concurrent updates
- **Soft Delete Retention**: Rows marked with `deleted_at` are retained for 3 months, then hard-deleted by application cleanup jobs
- **Cleanup Guard**: Cleanup and history/audit pruning jobs use local singleflight plus Redis distributed locks to stay safe in horizontally scaled deployments

### WebSocket Server Behavior

- **Server Role**: flag-gen provides the WebSocket server and does not run a reconnect loop for clients
- **Client Reconnect**: If a client disconnects, the client reconnects and the normal connection flow repeats
- **Initial Burst**: On every successful client connection, flag-gen sends the full scoped flag set to that client, then only later committed changes

### Database Lifecycle Guard

- **Install Mode**: Show install page only when MySQL is reachable and flag-gen tables are empty; DB connection config comes only from backend environment variables
- **Schema Status**: Backend reports database state, current schema version, target schema version, and compatibility before protected dashboard routes load
- **Migration Blocking**: Frontend shows a blocking migration page when migration is required; optional compatible drift shows a non-blocking notification
- **Go Migrator**: Schema migrations are executed by a Go-based migrator. The recommended runtime uses separate `/app/server` and `/app/migrate` binaries with one shared migrator package and registry.

### Error Tracking and Monitoring

- **Rich Logging**: Request context, user info, stack traces for debugging
- **Correlation IDs**: Trace requests across distributed components
- **Alerting**: Domain-specific alerts via Prometheus/ELK

These mechanisms aim to keep service availability and data consistency stable during runtime failures.

## Architecture Approach

### DDD + Clean Architecture

The application follows **Domain-Driven Design (DDD)** principles combined with **Clean Architecture** to ensure:

- **Domain-Centric Design**: Business logic at the core, technology-agnostic
- **Layered Architecture**: Clear separation of concerns with dependency inversion
- **Testability**: Independent layers that can be tested in isolation
- **Maintainability**: Easy to modify and extend without affecting other layers

#### Architecture Layers

```
┌─────────────────────────────────────┐
│         Presentation Layer          │
│  - HTTP REST Controllers            │
│  - WebSocket Broadcast Handlers     │
│  - DTOs & View Models              │
└─────────────────────────────────────┘
                    │
┌─────────────────────────────────────┐
│       Application Layer            │
│  - Use Cases / Application Services│
│  - Command & Query Handlers        │
│  - Outbox Dispatchers              │
│  - Event Handlers                  │
└─────────────────────────────────────┘
                    │
┌─────────────────────────────────────┐
│         Domain Layer                │
│  - Entities & Aggregates           │
│  - Value Objects                   │
│  - Domain Services                 │
│  - Domain Events                   │
│  - Business Rules & Invariants     │
└─────────────────────────────────────┘
                    │
┌─────────────────────────────────────┐
│     Infrastructure Layer           │
│  - Repositories (Implementations)  │
│  - External APIs                   │
│  - Message Queues                  │
│  - File Systems                    │
│  - Frameworks & Drivers            │
└─────────────────────────────────────┘
```

#### DDD Strategic Patterns

- **Bounded Contexts**: Separate MVP domains (Flag Management, Targeting, Evaluation, Integration)
- **Context Mapping**: Define relationships between bounded contexts
- **Ubiquitous Language**: Consistent terminology across team and code
- **Aggregate Roots**: Encapsulate business invariants
- **Domain Events**: Communicate changes between aggregates

#### Clean Architecture Principles

- **Dependency Inversion**: Inner layers don't depend on outer layers
- **Single Responsibility**: Each class has one reason to change
- **Open/Closed**: Open for extension, closed for modification
- **Interface Segregation**: Clients depend only on methods they use
- **Liskov Substitution**: Subtypes are substitutable for base types

### Detailed Technology Stack

Based on the comprehensive technology specifications in [technologies.md](technologies.md), the application leverages the following stack to align with Clean Architecture and DDD principles:

#### Backend Runtime (Domain & Application Layers)

- **Language**: Go (for high-performance domain logic, concurrency, and clean interfaces)
- **Framework**: Hexagonal architecture implementation with dashboard-focused CQRS-style command/query separation
- **CQRS Pattern**: Separate command and query models for dashboard/admin activity, especially flag mutations, audit history, list/detail reads, version diffs, and rollback workflows
- **Outbox Pattern**: Transactional event recording for reliable async broadcasts

#### Data Access (Infrastructure Layer)

- **Primary Database**: MySQL (relational integrity for domain aggregates, ACID compliance)
- **Read Database**: MySQL read replicas for dashboard/admin query optimization when needed
- **Cache**: Ristretto L1 for per-process hot scoped read models and Redis L2 for shared cross-replica cache/distributed locks
- **Analytics Storage**: Deferred until advanced analytics/experimentation enters scope
- **Message Broker**: Used for service-to-service flag change broadcasts
  - **Primary Recommendation**: RabbitMQ (mature, AMQP support, clustering, high availability)
  - **Alternatives**: Apache Kafka (high-throughput streaming), NATS (lightweight)

#### Service Runtime Concerns

- **Routing Boundary**: HTTP handlers expose dashboard/admin commands and query-side reads; WebSocket handlers deliver scoped flag state to runtime subscribers
- **Monitoring Hooks**: Service-level metrics, structured logs, and traces are emitted from the application and infrastructure layers
- **Security Boundary**: Dashboard authentication, lightweight runtime scope checks, and audit logging are enforced before domain mutations or full-scope flag delivery

#### Presentation Layer

- **APIs**: HTTP REST as the primary dashboard/admin API and client-side/browser flag read API, with server-side HTTP use for recovery, polling fallback, or explicit remote evaluation
- **Broadcasts**: WebSocket for server-side service-to-service flag state synchronization and committed flag change notifications
- **Web Dashboard**: Angular with TypeScript and GrowthBook-inspired UX patterns
- **SDKs**: SDKs are follow-up work after flag-gen API contracts stabilize; priority SDKs are Go for backend services and Angular/browser integration for frontend usage

#### Failover Caching Strategy

- **Persistent Snapshots**: Pretty-printed JSON files with version, checksum, and timestamps for flag data
- **Failover Mechanism**: Automatic fallback to valid snapshots on DB or Redis failure, lazy creation for migrations
- **Transaction Safety**: Flag mutations commit DB state and outbox events atomically
- **Snapshot Freshness Safety**: Snapshot files are refreshed only after create/update/delete workflows fully succeed, so runtime services never consume partially applied mutation state from files
- **Broadcast Consistency**: Background outbox dispatcher retries until events are published successfully

This stack ensures technology choices support domain independence while keeping the service reliable and observable.

### Workflow Patterns and Implementation Details

Drawing from the sequence diagram in [sequence-diagram.puml](sequence-diagram.puml), the application implements specific patterns for reliable operation:

#### CQRS Implementation

- **Dashboard/Admin Scope**: CQRS is applied primarily to user activity in the dashboard, where commands mutate flag data and query models support stable reads for list/detail/history/diff/rollback screens
- **Command Side**: Handles dashboard/admin writes (e.g., CreateFlagCommand, UpdateFlagCommand, RollbackFlagCommand) with domain validation, audit logging, optimistic locking, and outbox event recording
- **Query Side**: Handles dashboard/admin reads with read models optimized for navigation, filtering, audit history, and operational visibility
- **Server-side Runtime Reads**: Service-to-service consumers do not participate in command/query CQRS flows; they receive all scoped flags on WebSocket connect, apply later committed broadcasts, update local read state, and serve evaluations from memory/cache
- **Client-side Runtime Reads**: Browser/client-side consumers use read-only HTTP public endpoints for browser-facing flag reads
- **Event-Driven Communication**: Domain events (e.g., FlagCreated) published to message broker for async processing

#### Transactional Outbox for Flag Mutations

- **Commit Boundary**: Flag state, version history, audit entry, and outbox event are committed together in MySQL
- **Reliable Broadcast**: A dispatcher publishes outbox events to the broker/WebSocket broadcast path after commit
- **Idempotency**: Consumers deduplicate by event ID and ignore stale aggregate versions
- **Rich Error Tracking**: Correlation IDs, user context, retry count, and stack traces for debugging

#### Failover and Resilience

- **Snapshot Fallback**: DB is the source of truth; Redis is an acceleration layer; pretty JSON files are read-through availability snapshots for DB or Redis outages
- **Snapshot Purpose**: Snapshot files are plain fallback copies of the latest valid scoped flag state; they are not another database and they are not edited directly
- **Lazy File Creation**: Read operations can create missing snapshots from committed DB state for migration safety
- **Controlled Freshness**: Snapshot files are only advanced after create/update/delete operations fully succeed, preventing partially applied mutations from becoming fallback state
- **Safe Defaults**: On corruption/failure, use predefined defaults to maintain service availability
- **Edge Cases**: Handles concurrency (optimistic locking), file corruption, and concurrent operations

#### Real-Time Capabilities

- **HTTP API**: Dashboard/admin screens use HTTP for commands and query-side reads; browser/client-side consumers use HTTP for public flag reads; server-side SDKs may use HTTP for recovery or polling fallback
- **WebSocket Broadcast**: Server-side service subscribers receive the full scoped flag set on connect, then committed flag change events after outbox publication and update local read state
- **Event Consumption**: Application layer consumes broker events to update dashboard read models and notify runtime subscribers

These patterns ensure the architecture handles complex workflows while maintaining domain integrity.

## Key Components

Based on the domain model in [domain-diagram.puml](domain-diagram.puml), the bounded contexts include concrete examples for clarity:

1. **Flag Management Bounded Context**
  - Domain: `Project`, `Environment`, `Flag` aggregate (e.g., id="flag-123", key="checkout_v2", value=true), `FlagValue`, `FlagRule`
  - Application: CreateFlag, UpdateFlag, EvaluateFlag use cases
  - Infrastructure: MySQL repositories, Ristretto L1 cache, Redis L2 cache
  - Events: `FlagCreated` (e.g., flagId="flag-123", occurredAt="2026-04-28T10:00:00Z")
2. **Targeting Engine Bounded Context**
  - Domain: `User` entity (e.g., attributes={"country": "ID"}), `Segment` aggregate (e.g., name="Premium Users"), `TargetingRule` value object (e.g., attribute="subscription", operator="EQUALS")
  - Application: EvaluateTargeting, UpdateSegments use cases
  - Infrastructure: Rule engine, user data providers
  - Events: `UserTargeted` (e.g., userId="user-001", segmentId="seg-101")
3. **Evaluation Bounded Context**
  - Domain: `EvaluationContext`, `EvaluationResult`, `RolloutBucket`, `FlagValue`
  - Application: EvaluateFlags, EvaluateFlag use cases
  - Infrastructure: Ristretto L1 cache, Redis L2 cache, and snapshot fallback loader
  - Events: Optional evaluation metrics emitted after the core service is stable
4. **Integration Bounded Context**
  - Domain: SDK, Webhook, APIKey, OutboxEvent entities
  - Application: RegisterSDK, SendWebhook, DispatchOutboxEvents use cases
  - Infrastructure: SDK generators, webhook clients, message broker producers

## Development Phases

1. **Phase 1**: MVP domain modeling and core dashboard management (2-3 months)
  - Deliverables: project/environment model, flag CRUD, boolean/JSON values, dashboard login, install flow, schema lifecycle guard, version history, audit log, MySQL setup
  - Risks: Domain complexity; Mitigation: Event storming sessions
2. **Phase 2**: Evaluation, targeting, and dashboard query API (2 months)
  - Deliverables: resource-based dashboard/admin HTTP APIs, simple targeting, stable percentage rollout, deterministic evaluator, L1/L2 read cache path
  - Risks: Rule complexity; Mitigation: Start with a small explicit rule model before adding richer expressions
3. **Phase 3**: Outbox, WebSocket synchronization, SDK, and snapshot fallback (2-3 months)
  - Deliverables: transactional outbox, idempotent consumers, WebSocket initial full-scope snapshot, committed change broadcasts, server-side SDK local read state, snapshot fallback for DB/Redis outage
  - Risks: Event retry and stale broadcasts; Mitigation: Event IDs, aggregate versions, retry policies, and consumer idempotency tests
4. **Phase 4**: Angular admin dashboard and service hardening (2 months)
  - Deliverables: GrowthBook-inspired Angular dashboard, observability, rate limiting, error handling, access checks, and documentation
  - Risks: UI/UX alignment; Mitigation: User feedback loops

Total timeline: 8-10 months with iterative validation against diagrams.

### Deferred Roadmap

- **Experimentation**: A/B tests, exposure tracking, statistical analysis, and winner determination
- **Advanced Analytics**: Real-time dashboards, long-term reporting, and data warehouse exports
- **AI-Powered Features**: Smart targeting, predictive analytics, and automated optimization
- **Advanced Client SDKs**: Mobile/client-side SDKs after server-side SDK semantics are stable

## Service Observability and Runtime Considerations

Aligned with [technologies.md](technologies.md), the assessment focuses on service behavior, runtime reliability, and debuggability:

### Service Observability

- **Metrics**: Application/domain metrics such as flag evaluation count, evaluation latency, cache hit rate, and failover activation count
- **Logging**: Structured service logs with correlation IDs, request context, actor information, and domain-specific error details
- **Tracing**: Request tracing across presentation, application, domain, and infrastructure layers
- **Alert Signals**: Domain-specific signals for DB failover activation, cache degradation, file corruption, and broker publish failures
- **Circuit Breaker Signals**: Track breaker open/half-open/closed transitions, rejected requests, and recovery probes

### Service Security

- **Dashboard Authentication**: MVP starts with standard username/password login and secure sessions; 2FA is optional
- **Deferred SSO**: OAuth 2.0, OpenID Connect, and SAML are deferred until there is a concrete integration need
- **Runtime Scope Control**: WebSocket access is internal runtime traffic and must be authorized by lightweight project/environment keys; do not rely on port separation alone
- **Flag Payload Rule**: Feature flags must not contain credentials or secrets; server-side and client-side flag payloads are treated as safe-to-expose read-only runtime configuration
- **Transport Security**: TLS for HTTP/WebSocket communication in deployed environments
- **Auditability**: Recent version history and user attribution for flag mutations, with history/audit tables capped at 5,000 rows each

### Service Performance

- **Caching**: Ristretto L1 cache for hot scoped read models, Redis L2 cache for shared dashboard/query caching across replicas, and SDK-side/local service caching for high-volume evaluations
- **Read Model Optimization**: Query-side projections optimized for dashboard reads; runtime services evaluate from local read state
- **Evaluation Latency**: Target low-latency evaluation by keeping rule execution deterministic and minimizing synchronous dependencies
- **Backpressure**: Rate limiting and broker retry policies to protect the service under load

This keeps the assessment centered on service readiness rather than runtime hosting choices.

## Testing and Validation Strategy

To validate the architecture and handle edge cases from [sequence-diagram.puml](sequence-diagram.puml), a comprehensive testing strategy is implemented:

### Unit Testing

- **Domain Logic**: Test aggregates, entities, and business rules (e.g., flag evaluation, targeting rules)
- **Application Services**: Test use cases and command/query handlers
- **Coverage**: 80%+ for domain and application layers

### Integration Testing

- **Layer Interactions**: Test repository implementations, message broker publishing/consuming
- **Bounded Contexts**: Contract tests for context interfaces
- **Infrastructure**: DB, cache, and file persistence validation

### End-to-End Testing

- **User Journeys**: Full workflows (flag creation to evaluation) with real dependencies
- **Edge Cases**: Concurrent updates, DB failures, file corruption (from diagram)

### Performance and Load Testing

- **Domain Scenarios**: Simulate high-volume flag evaluations, local state updates, initial WebSocket snapshots, and service-to-service broadcasts
- **Failover Validation**: Test automatic fallback and recovery processes

### Security and Access Testing

- **Dashboard Login**: Authentication tests for password login, session handling, logout, and optional 2FA when enabled
- **Runtime Scope Checks**: Verify WebSocket/SDK keys only receive the authorized project/environment flag set
- **Client-side Exposure**: Validate that browser/client-side endpoints only expose flags intended for public clients
- **Audit Trails**: Versioning and logging integrity

Testing ensures the service handles all scenarios reliably through automated validation at each layer.

## Files in This Assessment

- [domain-diagram.puml](domain-diagram.puml): DDD domain model with bounded contexts
- [sequence-diagram.puml](sequence-diagram.puml): Application workflow with clean layers
- [technologies.md](technologies.md): Technology stack aligned with clean architecture
- [features.md](features.md): Detailed feature list
- [api-contract.md](api-contract.md): API contract between dashboard, SDKs, backend services, and the flag-gen service
- [evaluation-semantics.md](evaluation-semantics.md): Deterministic MVP evaluator and result metadata semantics
- [database-schema.md](database-schema.md): Recommended MySQL schema for MVP persistence, outbox, audit, access keys, and snapshot metadata
- [database-lifecycle.md](database-lifecycle.md): Install, migration, and schema compatibility guard behavior
- [snapshot-format.md](snapshot-format.md): Snapshot JSON file format, checksum, atomic write rules, and degraded cache fill TTL
