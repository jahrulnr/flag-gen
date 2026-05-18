# Assessment — Technical specifications

Design-time specs for implementing and reviewing flag-gen. For usage and integration, see [../guide/README.md](../guide/README.md).

## Core specs

| Document | Contents |
|----------|----------|
| [features.md](features.md) | MVP feature list |
| [api-contract.md](api-contract.md) | HTTP routes, WebSocket messages, errors, idempotency |
| [evaluation-semantics.md](evaluation-semantics.md) | Deterministic evaluate: rules, rollout, `reason`, `source` |
| [database-schema.md](database-schema.md) | MySQL tables |
| [database-lifecycle.md](database-lifecycle.md) | Install, migrate, schema compatibility guard |
| [snapshot-format.md](snapshot-format.md) | Snapshot file format and degraded TTL rules |

## Architecture

| Document | Contents |
|----------|----------|
| [plan.md](plan.md) | Development plan, failover, outbox, cache strategy |
| [technologies.md](technologies.md) | Stack and runtime read path |
| [sequence-diagram.puml](sequence-diagram.puml) | flag-gen request/mutation/broadcast sequences |
| [domain-diagram.puml](domain-diagram.puml) | Domain model |

## OpenAPI

[../api/openapi.yaml](../api/openapi.yaml)

## Implementation tracking

[../guide/implementation-status.md](../guide/implementation-status.md)
