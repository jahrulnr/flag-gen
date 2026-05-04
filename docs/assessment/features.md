# Feature Flagging Application Features

This document outlines the feature set for the self-hosted feature flagging service. The assessment prioritizes a focused MVP before experimentation, advanced analytics, and AI-driven capabilities.

## MVP Scope
- Core flag CRUD with version history and audit logs
- Project and environment management
- Boolean and JSON flag values
- Simple targeting rules and stable percentage rollouts
- Resource-based HTTP API for dashboard/admin workflows and client-side/browser flag reads
- WebSocket broadcast for server-side service-to-service full flag state sync and later flag change notifications
- Server-side SDK with WebSocket local state, caching, and snapshot fallback behavior
- Database install and migration guard driven by backend schema status
- Angular admin dashboard with GrowthBook-inspired UX patterns

## Core Features

### 1. Flag Management
- **Create, Read, Update, Delete (CRUD)**: Full lifecycle management of feature flags
- **Multi-environment Support**: Development, staging, production environments
- **Flag Archiving**: Clean up old flags without breaking applications
- **Flag Values**: Boolean and JSON values for release toggles and remote configuration-like use cases
- **Version History**: Complete audit trail of flag changes with user attribution
- **Change Tracking**: Version comparison and diff capabilities
- **Rollback to Version**: Restore previous flag versions when needed

### 2. Targeting & Segmentation
- **User Attributes**: Target based on custom user properties
- **Geographic Targeting**: Country, region, city-based rules
- **Percentage Rollouts**: Gradual release to percentage of users
- **Stable Bucketing**: Consistent user assignment based on flag key, environment, and user identifier
- **Rule Combinations**: Start with simple AND logic; expand only after MVP rules are stable
- **Segments**: Reusable static or rule-based segments for common audiences

### 3. Evaluation
- **Single Flag Evaluation**: Evaluate one flag by key and environment
- **Bulk Evaluation**: Evaluate multiple flags for the same context
- **Evaluation Context**: User ID and custom attributes used by targeting rules
- **Evaluation Result Metadata**: Return reason, matched rule, source, and flag version for debugging
- **Safe Defaults**: Return explicit fallback values when flag data is unavailable

### 4. Service-to-Service Broadcast
- **WebSocket Broadcast**: Send all authorized flags to a service on connection, then broadcast committed flag changes with flag value payloads
- **Outbox-backed Events**: Publish committed flag changes from transactional outbox records
- **Local State Updates**: Services and SDKs upsert/delete local flag state using project, environment, flag key, value, and version from broadcasts
- **Idempotent Consumers**: Ignore duplicate or stale events by event ID and aggregate version

## Analytics & Insights

### 5. Service Metrics
- **Evaluation Counters**: Track evaluation volume by project, environment, and flag
- **Latency Metrics**: Track flag evaluation latency and cache lookup latency
- **Fallback Metrics**: Track DB failover, snapshot fallback, and safe default usage
- **Outbox Metrics**: Track pending events, publish attempts, retries, and failures

### 6. Performance Monitoring
- **Flag Evaluation Latency**: Monitor response times
- **Cache Hit Rates**: Optimize caching strategies
- **Error Tracking**: Rich error logging with request context, user information, and detailed stack traces
- **Correlation ID Tracking**: End-to-end request tracing for debugging complex issues
- **Error Pattern Analysis**: Automated detection of recurring error patterns
- **Usage Analytics**: Track flag usage patterns

### 6a. Redis Recovery
- **Event-driven Reconnect**: Redis disconnects publish an internal reconnect event
- **Mutex-protected Reconnect**: Only one reconnect attempt runs at a time even if many API calls detect the same outage
- **Fallback During Reconnect**: Reads can continue through DB or valid snapshot fallback while Redis reconnects
- **L1 Continuity**: The flag-gen server can keep serving hot read models from per-process Ristretto cache while Redis reconnects, bounded by version/TTL rules
- **Read Path**: Runtime WebSocket/HTTP reads check L1 Ristretto, then L2 Redis, then DB, then snapshot fallback when DB is unavailable
- **Singleflight Refill**: Cache refill is protected by per-scope singleflight so concurrent requests do not stampede Redis, MySQL, or snapshot file reads
- **Cache TTL Recommendation**: L1 scoped flag read models use 2-week TTL; L2 scoped flag read models use 1-year TTL
- **Degraded Cache Fill (From File)**: If **DB is down**, **L2 misses**, and **L1 misses** but a valid snapshot file exists, the service fills cache from file snapshot with a **short TTL (e.g., 1 hour)**. If Redis is reachable, it may also backfill L2 using the same short TTL; otherwise only L1 is filled.
- **Negative Cache**: HTTP `flag not found` results are cached as L1-only not-found markers for 5 minutes
- **Circuit Breaker**: 5xx/503 caused by empty caches plus unavailable DB/snapshot fallback opens a circuit breaker to protect the database path

### 6b. Background Job Safety
- **Local Singleflight**: Cleanup, audit/history pruning, and snapshot checks merge duplicate triggers inside one process
- **Redis Distributed Mutex**: Horizontally scaled replicas use Redis locks so only one replica performs a cleanup or snapshot check job at a time
- **Skip on Lock Failure**: If the lock is held or Redis is unavailable, scheduled maintenance jobs skip and retry later instead of blocking runtime traffic

## Integration & Extensibility

### 7. SDK Ecosystem
- **SDK Timing**: SDKs are follow-up work after the flag-gen API contract is stable
- **Priority SDKs**: Go SDK for backend/service-to-service use and Angular/browser integration for frontend use
- **Server-side SDK First**: Backend/server SDKs use authorized WebSocket for full scoped flag state and committed change messages
- **Client-side HTTP Reads**: Browser/client-side SDKs use read-only HTTP public endpoints; flag payloads must be safe to expose
- **Broadcast-first Server Reads**: Server-side SDKs and backend services receive the full scoped flag set on WebSocket connect, then receive committed flag changes through broadcasts and update local read state
- **HTTP Server Fallback**: Server-side SDKs can use HTTP for recovery, polling fallback, or explicit remote evaluation calls
- **Local Caching**: Backend/server SDKs may use Ristretto L1 cache for flag definitions/evaluation data with TTL and version awareness
- **Snapshot Fallback**: Server-side SDKs can use service-provided snapshots for degraded reads

### 8. API, WebSocket, and Webhooks
- **Resource-based REST API**: Endpoints such as `POST /projects/{projectId}/environments/{environmentKey}/flags`, `PATCH /projects/{projectId}/environments/{environmentKey}/flags/{flagId}`, `POST /projects/{projectId}/environments/{environmentKey}/flags/evaluate`, and `GET /projects/{projectId}/environments/{environmentKey}/flags/{flagKey}`
- **HTTP Dashboard API**: Dashboard/admin workflows use HTTP for command and query operations
- **HTTP Client-side Runtime**: Browser/client-side SDKs use HTTP public endpoints for client-side flags
- **HTTP Server Fallback**: Server-side SDKs and backend services use HTTP for recovery, polling fallback, or explicit remote evaluation calls
- **Dedicated Resources**: Explicit resources for projects, environments, flags, segments, and evaluation
- **WebSocket Broadcast**: Server-side service-to-service subscriptions for full initial flag state and later flag change events
- **Webhook Support**: Real-time notifications for flag changes
- **API Rate Limiting**: Prevent abuse with configurable limits
- **API Versioning**: Backward-compatible API evolution

### 9. Service Integrations
- **Monitoring Hooks**: Metrics, logs, and traces exposed for external observability systems
- **Event Export**: Stream flag change events to downstream service consumers
- **Webhook Support**: Notify downstream services when flags or segments change
- **API Clients**: Service-to-service integration through lightweight scoped keys and SDKs

## Access Control and Administration

### 10. MVP Access Control
- **Dashboard Login**: Standard username/password login for MVP dashboard access
- **Optional 2FA**: Two-factor authentication can be enabled later without blocking the MVP
- **Runtime Scoped Access**: WebSocket subscribers use lightweight project/environment authorization because WebSocket is internal runtime traffic
- **Client-side Flags**: Browser/client-side integrations may read flags through read-only public HTTP endpoints; feature flags must not be used as a secret or credential store
- **Audit Logs**: Activity tracking with user attribution and change history, capped at the latest 5,000 audit rows
- **Rich Error Logging**: Detailed error context with correlation IDs for troubleshooting
- **Security Event Logging**: Login and token usage events for operational troubleshooting
- **Version Audit Trail**: Recent version history of who changed what and when, capped at the latest 5,000 version rows
- **Transport Encryption**: Use TLS in deployed environments

### 11. Deferred Enterprise Access
- **Single Sign-On (SSO)**: SAML, OAuth 2.0, and OpenID Connect are deferred until required
- **Role-Based Access Control (RBAC)**: Start minimal; expand to granular permissions when dashboard roles require it
- **Compliance Reporting**: Audit reports for regulatory requirements
- **Advanced Data Encryption**: At-rest encryption requirements are deferred until needed

### 12. Administration
- **User Management**: Multi-user support with teams
- **Organization Structure**: Hierarchical team management
- **API Key Management**: Secure API access tokens
- **Usage Quotas**: Prevent resource abuse
- **Backup & Recovery**: Automated data backups
- **Install Flow**: Show install page only when the configured database is reachable and empty
- **Migration Guard**: Notify when schema migration is optional; block dashboard access when migration is required

### 13. Service Resilience
- **Snapshot Fallback**: Intelligent failover to file-backed snapshots when MySQL and Redis L2 cannot serve reads
- **Snapshot Meaning**: Snapshot files are last known-good full flag state for a scope, used only as fallback when DB/Redis reads are unavailable
- **Snapshot Persistence**: Snapshot files may remain permanently available while no update/delete activity advances the scoped flag state
- **Snapshot Integrity**: Pretty JSON snapshots include version, checksum, and timestamp metadata
- **Lazy Snapshot Creation**: Automatic snapshot creation when database data exists but files are missing
- **Controlled Snapshot Freshness**: Snapshot files are updated only after create/update/delete workflows fully succeed, preventing partial mutation state from becoming fallback data
- **Snapshot Check Guard**: Snapshot consistency checks use local singleflight and Redis distributed mutexes
- **Migration Safety**: Ensures complete service data availability when moving between self-hosted instances
- **Transactional Outbox**: Record flag change events in the same transaction as the flag mutation
- **Idempotent Broadcasts**: Consumers deduplicate repeated broadcasts and ignore stale flag versions
- **Offline Evaluation Mode**: Limited flag evaluation using cached or file-backed data
- **L1 In-memory Cache**: Ristretto per-process cache for hot scoped flag state and recently loaded snapshots
- **Snapshot Fallback Is Not L3 Cache**: File snapshots are treated as last-resort availability fallback, not a normal cache tier
- **Circuit Breaker Recovery**: Breakers recover through half-open probes after DB or snapshot recovery signals
- **Soft Delete Cleanup**: Rows with `deleted_at` are hard-deleted automatically after 3 months
- **Cleanup Guard**: Cleanup jobs and audit/history pruning use local singleflight and Redis distributed mutexes

## Developer Experience

### 14. Developer Tools
- **Debugging Tools**: Flag evaluation simulator
- **Code Generation**: Generate flag usage code snippets
- **Documentation**: Auto-generated API docs and guides
- **CLI Tools**: Command-line interface for automation

### 15. Quality Assurance
- **Automated Testing**: Flag behavior validation
- **Isolated Test Environments**: Validate flag behavior without affecting active users
- **Rollback Automation**: One-click emergency rollbacks
- **Change Approval**: Review process for flag changes

## Roadmap Features

### Deferred Enhancements
- **Experimentation & A/B Testing**: Exposure tracking, statistics, and winner determination
- **Advanced Analytics**: Historical reports, exports, and richer dashboards
- **Client-side and Mobile SDKs**: Browser/mobile SDKs after server-side SDK behavior is stable
- **AI-Powered Targeting**: Smart segmentation and predictive optimization after core analytics mature

This feature set positions the platform as a service-focused feature flagging system with strong runtime reliability, evaluation safety, and extensibility.
