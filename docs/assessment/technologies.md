# Technology Stack

## Backend Services

### Core Runtime (Domain & Application Layers)
- **Language**: Go (for performance and concurrency in domain logic)
- **Framework**: Custom hexagonal architecture implementation
- **CQRS Pattern**: Separate command and query models for dashboard/admin activity, not for every runtime service interaction
- **Transactional Outbox**: Reliable event recording for flag change broadcasts

### Data Access (Infrastructure Layer)
- **Primary Database**: MySQL (relational data, ACID compliance for domain integrity)
- **Read Database**: MySQL read replicas are optional for dashboard/admin query-side workloads when needed
- **L1 In-memory Cache**: Ristretto per-process cache inside the flag-gen server for hot scoped flag read models, reconnect events, and recently loaded snapshot data; recommended TTL is 2 weeks
- **L2 Shared Cache**: Redis for shared cross-replica cache, dashboard read models, bootstrap/recovery reads, and distributed locks; recommended TTL is 1 year for scoped flag read models
- **Snapshot Fallback**: Pretty JSON files are a last-resort availability fallback, not a normal L3 cache, because they are refreshed only after committed mutation flows and may intentionally be stale
- **Message Queue**: Message broker for outbox-driven service-to-service broadcasts

### Message Broker Specifications
- **Protocol Support**: Start with one broker protocol that fits the selected broker; avoid multi-protocol requirements in the MVP
- **Message Persistence**: Durable queues with configurable retention policies
- **Delivery Guarantees**: At-least-once publishing paired with idempotent consumers
- **Routing Capabilities**: Direct, topic, headers, and fanout exchange types
- **Clustering & High Availability**: Multi-node clustering is a production hardening option, not an MVP requirement
- **Monitoring & Management**: Web-based management interface and metrics endpoints
- **Access Control**: Use deployment-appropriate transport security and lightweight credentials for broker access
- **Performance**: High-throughput with low latency for real-time event processing

### Recommended Message Brokers

#### Primary Recommendation: RabbitMQ
- **Why Recommended**: Mature ecosystem, excellent AMQP support, rich feature set for complex routing scenarios
- **Strengths**: Advanced routing, management UI, plugin ecosystem, proven runtime reliability
- **Use Case Fit**: Good fit for outbox dispatch, full-scope WebSocket initialization, flag change broadcasts, and local state update events
- **Community Support**: Large community with extensive documentation and integrations

#### Alternative: Apache Kafka
- **Why Recommended**: High-throughput event streaming, excellent for analytics and log aggregation
- **Strengths**: Scalable event streaming, retention policies, exactly-once semantics
- **Use Case Fit**: Better for future high-volume analytics events and long-term event storage
- **Considerations**: Higher complexity for simple messaging patterns

#### Alternative: NATS
- **Why Recommended**: Lightweight, high-performance messaging for distributed service communication
- **Strengths**: Lightweight operation, auto-discovery, multi-protocol support
- **Use Case Fit**: Good for lightweight service-to-service full-scope initialization and flag change broadcasts
- **Considerations**: Less feature-rich for complex routing compared to RabbitMQ

### Service Infrastructure
- **HTTP Router**: Presentation layer routing for dashboard REST, client-side HTTP flag reads, server-side fallback HTTP, and WebSocket endpoints
- **Middleware**: Authentication, authorization, correlation IDs, rate limiting, and structured logging
- **Background Workers**: Outbox dispatchers, projection updates, Redis reconnect workers, local state update broadcasts, and webhook delivery
- **Storage Adapters**: MySQL, Redis, Ristretto in-memory cache, file persistence, and message broker implementations behind domain/application interfaces
- **Dependency Injection**: Use Google Wire for compile-time dependency injection. Keep provider sets and injectors under `infrastructure/di/*` and generate `wire_gen.go` via `make wire`.
- **Migration Runtime**: Build two Go binaries from the same repository: `/app/server` for the main service and `/app/migrate` for schema migration execution, backed by a shared internal migrator package

### Redis Reconnect Handling
- **Disconnect Detection**: Redis adapter publishes an internal in-memory event when a command detects connection loss
- **Single Reconnect Owner**: Reconnect processing uses a local mutex/singleflight guard so concurrent API failures in one process do not start competing reconnect attempts
- **Read Fallback During Reconnect**: Reads bypass Redis and fall back to MySQL or the newest valid snapshot while reconnect is in progress
- **Cache Refill**: After reconnect succeeds, hot read models can be repopulated from committed DB state or snapshot data
- **Observability**: Emit metrics and logs for disconnect count, reconnect attempts, reconnect success, reconnect failure, and duration

### Runtime Read Cache Path
- **Read Order**: WebSocket and HTTP runtime reads check L1 Ristretto first, then L2 Redis, then MySQL, and only use snapshot files when database access is unavailable
- **L1 Hit**: Return the scoped flag state directly from Ristretto
- **L1 Miss, L2 Hit**: Return the L2 Redis value and refill L1 asynchronously or inline without delaying the response path unnecessarily
- **Singleflight Refill**: Refill is protected by per-scope singleflight so concurrent requests for the same project/environment do not stampede Redis, DB, or the filesystem
- **L2 Miss**: Load committed state from MySQL, then refill L2 and L1
- **Not-found Negative Cache**: If an HTTP flag read confirms a flag does not exist, cache a not-found marker in L1 for 5 minutes to avoid repeated DB lookups; do not store this marker in L2 by default
- **DB Unavailable**: Load the newest valid pretty JSON snapshot as last-resort fallback; snapshots may persist indefinitely when there is no update/delete activity
- **Degraded Cache Fill (From File)**: If **DB is down**, **L2 misses**, and **L1 misses** but a valid snapshot file exists, the service may fill caches from the snapshot but must use a **short TTL (e.g., 1 hour)** for entries derived from file fallback. If Redis is reachable, it may also backfill L2 using the same short TTL; otherwise only L1 is filled.
- **5xx Circuit Breaker**: If L1, L2, and snapshot fallback cannot serve data and DB is unavailable, return 5xx/503 and open a circuit breaker for the affected dependency/scope to prevent repeated DB pressure
- **No Silent Default on Remote Reads**: HTTP remote reads must not silently convert this failure into a successful default evaluation unless an endpoint explicitly supports degraded defaults
- **Circuit Breaker Recovery**: The breaker should use open, half-open, and closed states with backoff and health probes; successful DB recovery closes the breaker and allows normal L1/L2 refill

### Background Job Concurrency
- **Local Singleflight**: Cleanup jobs and snapshot consistency checks use per-process singleflight/mutex guards so repeated triggers inside one server instance are merged
- **Distributed Mutex**: Jobs that can run from multiple server replicas acquire a Redis lock before doing database/file work, for example `flag-gen:lock:cleanup` and `flag-gen:lock:snapshot-check`
- **Lock Safety**: Redis locks must include owner token, TTL, renewal for long jobs, and compare-and-delete unlock semantics so one replica cannot release another replica's lock
- **Degraded Behavior**: If Redis is unavailable, skip scheduled cleanup/snapshot check jobs and retry later; user-facing reads can still use DB or snapshot fallback

## Domain Layer Technologies

### Domain Modeling
- **Project and Environment**: Scope flags and API keys by product area and runtime environment
- **FlagKey and FlagValue**: Model stable flag keys and boolean/JSON values explicitly
- **EvaluationContext and EvaluationResult**: Represent user attributes, matched rules, returned values, and evaluation reasons
- **Value Objects**: Go structs with validation
- **Entities**: Domain entities with business methods
- **Aggregates**: Aggregate roots with consistency boundaries
- **Domain Events**: Event-driven communication between aggregates
- **Domain Services**: Stateless business logic

### Validation & Business Rules
- **Invariant Validation**: Domain-level validation
- **Business Rule Engine**: Start with a small explicit rule evaluator for simple targeting and percentage rollout
- **Specification Pattern**: Reusable business rules

## Application Layer Technologies

### Use Cases & Commands
- **Command Handlers**: Dashboard/admin command processing for flag, environment, segment, audit-relevant, and rollback workflows
- **Query Handlers**: Dashboard/admin query processing for list/detail/history/diff views and operational visibility
- **Application Services**: Orchestrate domain operations
- **Outbox Dispatcher**: Publishes committed flag change events and retries failed publishes
- **DTOs**: Data transfer between layers

### Cross-Cutting Concerns
- **Logging**: Structured logging with correlation IDs and rich context
- **Rich API Error Tracking**: Comprehensive error logging with request context, user info, and stack traces
- **Distributed Tracing**: Request tracing across all layers with unique correlation IDs
- **Error Handling**: Domain-specific error types with detailed error information
- **Transactions**: Unit of work pattern with transaction logging
- **Access Control**: Dashboard authentication, lightweight runtime scope checks, and audit logging

## Presentation Layer Technologies

### API Design
- **Resource-based HTTP API**: Endpoints such as `POST /projects/{projectId}/environments/{environmentKey}/flags`, `PATCH /projects/{projectId}/environments/{environmentKey}/flags/{flagId}`, `POST /projects/{projectId}/environments/{environmentKey}/flags/evaluate`, and `GET /projects/{projectId}/environments/{environmentKey}/flags/{flagKey}`
- **HTTP Dashboard Path**: Dashboard/admin workflows use HTTP for commands and query-side reads
- **HTTP Client-side Runtime Path**: Browser/client-side SDKs use read-only public HTTP endpoints for browser-facing flag reads
- **HTTP Server Fallback Path**: Backend services and server-side SDKs may use HTTP for recovery, polling fallback, or explicit evaluation calls
- **WebSocket Server-side Runtime Path**: Service-to-service subscribers receive all scoped flags on connect, then committed flag change payloads to update local read state
- **REST API**: Resource-based endpoints with proper HTTP methods
- **OpenAPI/Swagger**: API documentation

### Web Dashboard
- **Framework**: Angular with TypeScript
- **UX Direction**: GrowthBook-inspired layout and interaction patterns to reduce relearning for existing GrowthBook users
- **UI Library**: Angular Material or a compatible component system
- **Charts**: Keep charts minimal for MVP; advanced analytics visualizations are deferred

## Infrastructure Layer Technologies

### Persistence
- **Repository Pattern**: Domain interfaces with infrastructure implementations
- **Unit of Work**: Transaction management
- **Outbox Store**: Persist flag change events in the same database transaction as aggregate mutations
- **Read Models**: CQRS read-side projections for dashboard/admin screens; runtime services keep local read state from the initial WebSocket snapshot and later broadcasts

### External Integrations
- **HTTP Clients**: REST client libraries
- **Message Brokers**: Message broker producers/consumers with protocol abstraction
- **File Storage**: Local filesystem adapter for file-backed failover snapshots under `/app/snapshot/*.json`
- **Third-party APIs**: SDK integrations

### Snapshot Fallback Strategy
- **Persistent Snapshots**: Pretty-printed JSON files for feature flag data with version, checksum, and timestamps
- **Storage Location**: Store local snapshots beside the application runtime, for example `/app/snapshot/*.json`
- **Purpose**: Snapshot files are last known-good scoped flag state for availability during DB/Redis problems, not a co-primary datastore
- **Cache Terminology**: Do not treat snapshot files as L3 cache in normal operation; they are a fallback source used when database/shared cache access is unavailable or during recovery
- **Failover Mechanism**: Automatic fallback to valid snapshots on database or Redis/cache connection errors
- **Data Synchronization**: Snapshot generation follows committed database state and is triggered only after create/update/delete workflows fully succeed
- **Staleness Model**: Existing snapshots are allowed to remain stale until a mutation completes successfully; failed or partial mutation flows must not advance snapshot files
- **Lazy Snapshot Creation**: Read operations check cache/database first when available, create missing snapshots from committed database state, and use the newest valid snapshot when cache or database is unavailable
- **No Mass Rebuild**: Do not run a full snapshot rebuild as a normal operation; use a scheduler to check snapshot/database consistency periodically and after database boot/reconnect events
- **Concurrency Guard**: Snapshot consistency checks are singleflight locally and protected by a Redis distributed mutex across replicas
- **Migration Safety**: Ensures files exist when migrating between self-hosted instances
- **Transaction Safety**: Database changes and outbox records commit atomically
- **Snapshot Write Safety**: Snapshot refresh happens after the successful mutation flow, so file-backed fallback never exposes partially applied create/update/delete state
- **Broadcast Safety**: Failed message publishes stay pending in the outbox and are retried
- **Idempotent Consumers**: Consumers deduplicate by event ID and aggregate version
- **Versioning System**: Track complete history of flag changes with user attribution
- **Audit Trail**: Created/updated by information with timestamps for compliance
- **Change Tracking**: Version comparison and diff capabilities
- **Timestamp Tracking**: Each file includes last modified timestamp for monitoring
- **Memory Loading**: Failed database connections trigger loading of JSON snapshots into memory
- **Recovery Process**: Automatic resync with database or cache refill once connection is restored

## SDKs & Client Libraries

### Core Features (All Layers)
- **SDK Timing**: Implement SDKs after the flag-gen API contract stabilizes
- **Priority SDKs**: Prioritize Go for backend/service-to-service use and Angular/browser integration for frontend use
- **Layer Communication**: Clean interfaces between SDK layers
- **Caching Strategy**: Backend/server SDKs may use Ristretto as L1 cache; client-side/browser SDKs do not need in-memory cache unless a future frontend server-side rendering/runtime mode requires it
- **Error Handling**: Domain-specific error propagation

### SDK Architecture
- **Presentation Layer**: Server-side SDK uses WebSocket for initial full-scope flag state and committed change messages; client-side SDK uses HTTP public endpoints; server-side HTTP remains for fallback and explicit remote evaluation
- **Application Layer**: Request coordination, server-side SDK WebSocket reconnect expectations, polling fallback, and version-aware local state updates
- **Domain Layer**: Limited deterministic flag evaluation from local read state
- **Infrastructure Layer**: Local storage, in-memory cache, file snapshot support, and connection management

## Service Observability

### Monitoring & Observability
- **Metrics**: Prometheus (application and domain metrics)
- **Logging**: ELK Stack with rich structured logging and correlation IDs
- **Rich Error Tracking**: Comprehensive error context with request details, user info, and stack traces
- **Tracing**: Jaeger (distributed tracing across layers with detailed span information)
- **Alerting**: Alertmanager with domain-specific alerts and error pattern detection

### Access Control (Cross-Cutting)
- **Dashboard Authentication**: Standard username/password login with secure sessions for MVP
- **Optional 2FA**: Can be added for dashboard users without changing runtime flag delivery
- **Deferred SSO**: OAuth 2.0, OpenID Connect, and SAML are deferred until required by integrations
- **WebSocket Internal Authorization**: WebSocket broadcast is internal runtime traffic and must be protected by authorization; do not rely on port separation alone to define internal access
- **Runtime Scope Keys**: Server-side SDK and WebSocket access use lightweight project/environment keys to limit full-scope flag delivery
- **Client-side Exposure**: Browser/client-side integrations use read-only HTTP endpoints; these endpoints may be public because flag payloads must not contain credentials or private config values
- **Flag Payload Rule**: Feature flags are safe-to-expose runtime configuration, not a secret store
- **Secrets Handling**: Injected service secrets for database, cache, broker, and signing keys

## Development Tools

### Domain-Driven Development
- **Ubiquitous Language**: Consistent terminology across layers
- **Bounded Context Mapping**: Context map documentation
- **Event Storming**: Collaborative domain modeling
- **Domain Storytelling**: User story mapping

### Testing Strategy (All Layers)
- **Unit Tests**: Domain logic and application services
- **Integration Tests**: Layer interactions
- **Contract Tests**: Bounded context interfaces
- **E2E Tests**: Full user journeys
- **Performance Tests**: Load testing with domain scenarios

## Architecture Benefits

### Clean Architecture Alignment
- **Dependency Inversion**: Domain doesn't depend on infrastructure
- **Testability**: Each layer can be tested independently
- **Maintainability**: Changes in one layer don't affect others
- **Flexibility**: Technology choices can change without affecting domain

### DDD Benefits
- **Business Alignment**: Code reflects business domain
- **Evolutionary Architecture**: Easy to adapt to changing requirements
- **Team Communication**: Shared understanding through ubiquitous language
- **Complex Domain Handling**: Suitable for complex business logic

## Service Performance Considerations

### Runtime Optimization
- **Database Partitioning**: Defer partitioning until analytics or event volume requires it
- **Caching Layers**: Ristretto L1 for process-local hot reads, Redis L2 for shared cross-replica cache, and snapshot files as last-resort fallback rather than L3 cache
- **Read Models**: Precomputed projections for fast flag evaluation and dashboard queries
- **Backpressure**: Rate limiting, circuit breakers, queue retry policies, and idempotent consumers for burst protection
