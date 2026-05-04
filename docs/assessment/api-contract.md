# API Contract for Feature Flagging Service

## Overview

This document defines the MVP API contract between the Angular admin dashboard, backend services, server-side SDKs, and the flag-gen feature flagging service.

HTTP is the primary protocol for dashboard/admin management, dashboard query-side reads, and client-side/browser flag reads. Server-side SDKs and backend services use WebSocket for runtime flag state: receive all scoped flags on connect, apply committed change broadcasts, and maintain local read state. The flag-gen server keeps hot scoped flag read models in a per-process Ristretto L1 cache, with Redis as L2 shared cache. Runtime reads use L1 -> L2 -> DB -> snapshot fallback. Server-side HTTP remains available for recovery, polling fallback, or explicit remote evaluation calls.

CQRS applies primarily to dashboard/admin workflows where user-driven commands and dashboard read models need separation for stability, auditability, and operational visibility. Server-side runtime consumers do not participate in command/query CQRS flows; they receive an initial scoped flag snapshot over WebSocket, wait for committed broadcasts, and update local cache or in-memory state. Client-side runtime consumers use HTTP public endpoints only.

## Base URL
- **Service API**: `/v1`
- **Local Example**: `http://localhost:8080/v1`

## Access Model
- **Management APIs**: Dashboard session from standard username/password login
- **Optional 2FA**: Can be enabled for dashboard users later without changing flag evaluation semantics
- **Deferred SSO**: OAuth 2.0, OpenID Connect, and SAML are not required for MVP
- **Server-side Runtime APIs**: Lightweight project/environment key when access needs to be scoped
- **Broadcast APIs**: WebSocket subscribers must be authorized with lightweight project/environment keys because the stream is internal runtime traffic
- **Client-side APIs**: Browser/client-side HTTP reads are read-only and may be public because flag payloads must not contain secrets or credentials
- **Header**: `Authorization: Bearer <token>` for token-based SDK or WebSocket access

### Permissions
- `flags:read`
- `flags:write`
- `flags:evaluate`
- `segments:read`
- `segments:write`
- `broadcast:subscribe`
- `admin`

Feature flag payloads must not contain credentials, secrets, or private configuration values. Client-side flags should be treated as public because browser applications can inspect their payloads.

## Common Headers
- `X-Correlation-ID`: Optional caller-provided request correlation ID
- `Idempotency-Key`: Required for create/update/delete commands that may be retried
- `If-Match`: Optional optimistic locking header using the current aggregate version

## System and Database Lifecycle

### GET /system/status
Return application and database lifecycle status. The frontend must call this before loading protected dashboard routes.

**Response (200):**
```json
{
  "appVersion": "1.1.0",
  "database": {
    "state": "migration_required",
    "connected": true,
    "empty": false,
    "currentSchemaVersion": 202605010001,
    "targetSchemaVersion": 202605010003,
    "minimumCompatibleSchemaVersion": 202605010001,
    "blockingBelowSchemaVersion": 202605010002,
    "dirty": false,
    "canInstall": false,
    "canMigrate": true,
    "runtimeReadCompatible": true
  },
  "message": "Database migration is required before dashboard management can continue.",
  "correlationId": "req-abc-123"
}
```

Expected `database.state` values:

- `db_unreachable`
- `db_empty`
- `db_ready`
- `migration_optional`
- `migration_required`
- `db_ahead`
- `db_invalid`

### POST /system/install
Bootstrap an empty database. This endpoint is available only when `GET /system/status` returns `db_empty`.

Database connection settings are never accepted from the frontend. They must come from backend environment variables or deployment secrets.

**Request:**
```json
{
  "admin": {
    "email": "admin@example.com",
    "name": "Admin",
    "password": "change-me"
  },
  "seed": {
    "projectKey": "default",
    "environmentKey": "production"
  }
}
```

**Response (201):**
```json
{
  "installed": true,
  "appVersion": "1.1.0",
  "schemaVersion": 202605010003,
  "correlationId": "req-abc-123"
}
```

### POST /system/migrate
Run pending migrations. This endpoint requires an authenticated dashboard admin session and is available only when the database state is `migration_optional` or `migration_required`.

**Response (200):**
```json
{
  "migrated": true,
  "previousSchemaVersion": 202605010001,
  "currentSchemaVersion": 202605010003,
  "applied": [
    {
      "version": 202605010002,
      "name": "add_client_side_flags"
    },
    {
      "version": 202605010003,
      "name": "add_snapshot_metadata_status"
    }
  ],
  "correlationId": "req-abc-123"
}
```

## Project and Environment Resources

### GET /projects
List projects visible to the current actor.

### POST /projects
Create a project.

### GET /projects/{projectId}/environments
List environments for a project.

### POST /projects/{projectId}/environments
Create an environment.

### DELETE /projects/{projectId}
Soft-delete a project. **409** if the project still has any environments or segments (remove those first). **204** on success.

### DELETE /projects/{projectId}/environments/{environmentKey}
Soft-delete an environment. **409** if the environment still has any flags. **204** on success.

## Flag Management

### GET /projects/{projectId}/environments/{environmentKey}/flags
List flags in an environment.

### POST /projects/{projectId}/environments/{environmentKey}/flags
Create a new feature flag.

**Request:**
```json
{
  "key": "checkout_v2",
  "name": "Checkout V2",
  "description": "New checkout flow",
  "enabled": false,
  "clientSide": false,
  "value": {
    "type": "boolean",
    "default": false
  },
  "rules": [
    {
      "attribute": "country",
      "operator": "equals",
      "value": "ID",
      "serve": true,
      "priority": 1
    }
  ],
  "rollout": {
    "percentage": 10,
    "stickiness": "userId"
  }
}
```

**Response (201):**
```json
{
  "id": "flag-123",
  "key": "checkout_v2",
  "version": 1,
  "createdAt": "2026-04-28T10:00:00Z",
  "correlationId": "req-abc-123"
}
```

### GET /projects/{projectId}/environments/{environmentKey}/flags/{flagKey}
Get a flag by key.

### PATCH /projects/{projectId}/environments/{environmentKey}/flags/{flagId}
Update an existing flag. The request must include the current version or `If-Match` header.

**Request:**
```json
{
  "version": 1,
  "enabled": true,
  "value": {
    "type": "json",
    "default": {
      "theme": "new-checkout"
    }
  },
  "rules": []
}
```

**Response (200):**
```json
{
  "id": "flag-123",
  "key": "checkout_v2",
  "version": 2,
  "updatedAt": "2026-04-28T10:05:00Z",
  "outboxEventId": "evt-789",
  "correlationId": "req-abc-123"
}
```

### DELETE /projects/{projectId}/environments/{environmentKey}/flags/{flagId}
Soft delete a flag. The request must include the current version or `If-Match` header.

**Response (204):** No content.

## Flag Evaluation

### POST /projects/{projectId}/environments/{environmentKey}/flags/evaluate
Evaluate one or more flags for a user or service context.

**Request:**
```json
{
  "context": {
    "userId": "user-123",
    "attributes": {
      "country": "ID",
      "subscription": "premium"
    }
  },
  "flags": ["checkout_v2", "new_ui"]
}
```

**Response (200):**
```json
{
  "results": {
    "checkout_v2": {
      "value": true,
      "version": 2,
      "reason": "rule_match",
      "matchedRuleId": "rule-456",
      "source": "database"
    },
    "new_ui": {
      "value": false,
      "version": 4,
      "reason": "default",
      "matchedRuleId": null,
      "source": "cache"
    }
  },
  "timestamp": "2026-04-28T10:06:00Z",
  "correlationId": "req-abc-123"
}
```

### GET /projects/{projectId}/environments/{environmentKey}/flags/{flagKey}/evaluate
Evaluate a single flag using query parameters for simple server-side SDK calls.

Example: `GET /projects/proj-1/environments/production/flags/checkout_v2/evaluate?userId=user-123`

If the flag key is confirmed missing, the service returns `404` and stores an L1-only not-found marker for 5 minutes. This negative cache is intended mainly for HTTP single-flag reads and must not be stored in L2 by default.

## Client-side HTTP Reads

Client-side/browser integrations use HTTP only. These endpoints are read-only and can be public because feature flag payloads must be safe to expose. The service may still use `clientSide`/visibility metadata to decide which flags are returned to browser-facing endpoints, but this is product scoping rather than secret protection.

### GET /client/projects/{projectId}/environments/{environmentKey}/flags
Return public/client-side flags for a project/environment.

**Response (200):**
```json
{
  "projectId": "proj-1",
  "environmentKey": "production",
  "scopeVersion": 42,
  "flags": {
    "checkout_banner": {
      "value": true,
      "version": 3,
      "reason": "default"
    }
  },
  "timestamp": "2026-04-28T10:06:00Z",
  "correlationId": "req-abc-123"
}
```

### GET /client/projects/{projectId}/environments/{environmentKey}/flags/{flagKey}/evaluate
Evaluate one public/client-side flag over HTTP.

Example: `GET /client/projects/proj-1/environments/production/flags/checkout_banner/evaluate?userId=user-123`

If the flag key is confirmed missing, the service returns `404` and stores an L1-only not-found marker for 5 minutes.

## Segments

### GET /projects/{projectId}/segments
List reusable user segments.

### POST /projects/{projectId}/segments
Create a segment.

### PATCH /projects/{projectId}/segments/{segmentId}
Update a segment.

### DELETE /projects/{projectId}/segments/{segmentId}
Delete a segment if no active flag rule depends on it.

## WebSocket Broadcast

### Connection: /v1/broadcast/flags
Subscribe backend services to flag state for a project/environment. The scope is selected by the token/key and may also be passed explicitly as query parameters, for example `/v1/broadcast/flags?projectId=proj-1&environmentKey=production`.

On connection, the server sends the full flag set for the authorized scope so the service can build local read state without telling flag-gen which individual flags it will use. After that, broadcasts are emitted only when committed flag changes are published from the transactional outbox.

The WebSocket stream is internal runtime traffic and must be protected by authorization. Do not rely on public/internal port separation as the only control. The stream is scoped not because feature flags are secrets, but to avoid arbitrary consumers subscribing to every project/environment. Flags must never contain secrets. Public client-side integrations use separate read-only HTTP endpoints.

This intentionally favors a simple full-scope subscription over per-flag subscription negotiation. Feature flag data is expected to be small enough that sending all flags on connect is acceptable, while avoiding extra complexity for tracking which service needs which flag.

**Initial Snapshot Message:**
```json
{
  "type": "flags.snapshot",
  "projectId": "proj-1",
  "environmentKey": "production",
  "scopeVersion": 42,
  "flags": [
    {
      "id": "flag-123",
      "key": "checkout_v2",
      "enabled": true,
      "clientSide": false,
      "value": {
        "type": "boolean",
        "default": true
      },
      "rules": [],
      "rollout": {
        "percentage": 10,
        "stickiness": "userId"
      },
      "version": 2
    }
  ],
  "occurredAt": "2026-04-28T10:04:59Z"
}
```

**Change Message:**
```json
{
  "type": "flag.updated",
  "eventId": "evt-789",
  "projectId": "proj-1",
  "environmentKey": "production",
  "flagId": "flag-123",
  "flagKey": "checkout_v2",
  "scopeVersion": 43,
  "enabled": true,
  "clientSide": false,
  "value": {
    "type": "boolean",
    "default": true
  },
  "rules": [],
  "rollout": {
    "percentage": 10,
    "stickiness": "userId"
  },
  "version": 2,
  "occurredAt": "2026-04-28T10:05:00Z"
}
```

**Delete Message:**
```json
{
  "type": "flag.deleted",
  "eventId": "evt-790",
  "projectId": "proj-1",
  "environmentKey": "production",
  "flagId": "flag-123",
  "flagKey": "checkout_v2",
  "scopeVersion": 44,
  "version": 3,
  "occurredAt": "2026-04-28T10:10:00Z"
}
```

Consumers must handle duplicate messages, ignore stale versions, and upsert/delete local flag state from the broadcast payload.

## Data Types

### Flag
```json
{
  "id": "string",
  "projectId": "string",
  "environmentKey": "string",
  "key": "string",
  "name": "string",
  "description": "string",
  "enabled": "boolean",
  "clientSide": "boolean",
  "value": "FlagValue",
  "rules": "array[FlagRule]",
  "rollout": "RolloutConfig",
  "version": "integer",
  "createdAt": "datetime",
  "updatedAt": "datetime",
  "deletedAt": "datetime|null"
}
```

### FlagValue
```json
{
  "type": "boolean|json",
  "default": "boolean|object|array|string|number|null"
}
```

### FlagRule
```json
{
  "id": "string",
  "attribute": "string",
  "operator": "equals|not_equals|in|not_in",
  "value": "boolean|object|array|string|number|null",
  "serve": "boolean|object|array|string|number|null",
  "priority": "integer"
}
```

### RolloutConfig
```json
{
  "percentage": "number",
  "stickiness": "userId|sessionId|custom"
}
```

## Error Handling

All errors include:
```json
{
  "error": "string",
  "message": "string",
  "details": "object|null",
  "correlationId": "string",
  "timestamp": "datetime"
}
```

### Expected Status Codes
- `400`: Invalid request
- `401`: Unauthorized
- `403`: Forbidden
- `404`: Resource not found
- `409`: Version conflict or idempotency conflict
- `422`: Domain validation failed
- `429`: Rate limit exceeded
- `500`: Internal server error
- `503`: Service degraded; response may include fallback source details or circuit breaker state

### Negative Cache and Circuit Breaker
- `404 flag_not_found`: Cache an L1-only not-found marker for 5 minutes for the affected flag key/scope. Do not store negative-cache markers in L2 by default.
- `5xx/503 degraded`: If L1, L2, and snapshot fallback cannot serve data and DB is unavailable, open a circuit breaker for the affected dependency/scope.
- `no_silent_default`: HTTP remote reads must not silently convert this failure into a successful default evaluation unless the endpoint explicitly supports degraded defaults.
- `circuit_open`: While the breaker is open, fail fast or serve valid fallback data instead of repeatedly hitting DB.
- `half_open`: Probe DB recovery with limited requests. On success, close breaker and refill L2/L1; on failure, reopen with backoff.

## Versioning
- API version in URL path
- Flag aggregate version for optimistic locking
- Broadcast messages include `scopeVersion` and aggregate `version` for idempotent consumers
- Deprecation notices via headers when APIs evolve

## Rate Limiting
- Management APIs: configurable per actor/API key
- Evaluation APIs: configurable per SDK key and environment
- Headers: `X-RateLimit-Remaining`, `X-RateLimit-Reset`

## SDK Integration
SDK implementation is follow-up work after flag-gen stabilizes its API contract. Priority SDK targets are Go for backend service-to-service use and Angular/browser integration for frontend use. Backend/server SDKs may use Ristretto as L1 in-memory cache. Browser/client-side SDKs do not need in-memory cache unless a future frontend server-side runtime requires it.

Server-side SDKs should maintain local cache/read state with TTL and version awareness. They should subscribe to authorized WebSocket broadcasts to apply committed flag changes, use polling or HTTP reads as fallback, and wrap evaluation endpoints only when remote evaluation is explicitly required. Client-side/browser SDKs should use read-only HTTP public/client-side endpoints. If L1/L2 cache and database reads are unavailable, server-side SDKs/services may use the newest valid service-provided snapshot; snapshots are refreshed only after create/update/delete workflows fully succeed.
