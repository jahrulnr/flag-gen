# Go SDK (`flagclient`) — Complete Guide

Official Go package: [`sdk/flagclient`](../../../sdk/flagclient/). Beginner-friendly reference for all SDK features.

Related docs:

- [Guide index](../README.md) · [How it works](../how-it-works.md)
- [Feature flags (EN)](../concepts/feature-flags.md)
- [Evaluation semantics](../../assessment/evaluation-semantics.md)
- [API contract](../../assessment/api-contract.md)
- [Examples](../../../examples/)

Indonesian version: [sdk-flagclient_id.md](sdk-flagclient_id.md)

---

## Table of contents

1. [What is this SDK?](#1-what-is-this-sdk)
2. [Two parts: Client vs ServerSDK](#2-two-parts-client-vs-serversdk)
3. [Core concepts](#3-core-concepts)
4. [Installation](#4-installation)
5. [`Client` configuration](#5-client-configuration)
6. [HTTP: list flags](#6-http-list-flags)
7. [HTTP: evaluate](#7-http-evaluate)
8. [ServerSDK: local cache + WebSocket](#8-serversdk-local-cache--websocket)
9. [WebSocket message types](#9-websocket-message-types)
10. [When to use which feature](#10-when-to-use-which-feature)
11. [Multi-user in one service](#11-multi-user-in-one-service)
12. [Authentication](#12-authentication)
13. [Full example](#13-full-example)
14. [Anti-patterns](#14-anti-patterns)
15. [MVP limitations](#15-mvp-limitations)
16. [API quick reference](#16-api-quick-reference)

---

## 1. What is this SDK?

The `flagclient` package wraps flag-gen HTTP and WebSocket APIs so Go services do not hand-roll URLs, headers, and JSON parsing.

**Goals:**

- Fast **backend** integration
- **Service-to-service** pattern: WebSocket for flag definitions, HTTP for evaluation
- Optional **public/client** HTTP reads for BFFs

There is no separate Angular SDK yet — use HTTP ([angular-client example](../../../examples/angular-client/)).

---

## 2. Two parts: Client vs ServerSDK

| Component | Transport | Data |
|-----------|-----------|------|
| **`Client`** | HTTP | API requests/responses |
| **`ServerSDK`** | WebSocket + HTTP | In-process **flag definitions** cache |

`ServerSDK` does **not** replace per-user evaluation. Use `Client.EvaluateFlags` for rules, rollout, and attributes.

---

## 3. Core concepts

### Flags vs evaluate

| | **Flags (definitions)** | **Evaluate (decision)** |
|---|-------------------------|-------------------------|
| What | enabled, default, rules, rollout | Result for **one request context** |
| Cached globally? | **Yes** | **No** (computed each call) |
| Same for all users? | Same rule book | Different outcomes per context |

### Evaluation context

Sent by **your service** on each evaluate call — flag-gen does not load your user DB automatically:

```json
{
  "context": { "userId": "u-1", "attributes": { "country": "ID" } },
  "flags": ["feature_debt"]
}
```

### `source` in results

`source: cache_l1` means the **flag definition** was read from cache — not that the user result was cached.

### `Variation()` vs `EvaluateFlags()`

| | `Variation()` | `EvaluateFlags()` |
|---|---------------|-------------------|
| Rules / rollout | No | Yes |
| Returns | Default from local cache | Full server evaluation |

---

## 4. Installation

```go
require flag-gen v0.0.0
replace flag-gen => ../path/to/flag-gen

import "flag-gen/sdk/flagclient"
```

Example: [examples/golang-service](../../../examples/golang-service/).

---

## 5. `Client` configuration

| Field | Purpose |
|-------|---------|
| `BaseURL` | e.g. `https://flags.example.com/v1` |
| `RuntimeKey` | Server API key (`X-API-Key`) |
| `PublicKey` | `X-Public-Api-Key` for `/public` and `/client` |
| `BroadcastDevToken` | WebSocket `token=` query (dev) |

---

## 6. HTTP: list flags

### `ListFlags`

`GET /v1/projects/{id}/environments/{env}/flags` — all flag definitions (server).

### `ListPublicFlags`

`GET /v1/public/.../flags` — only `clientSide: true` flags (browser-safe payloads).

---

## 7. HTTP: evaluate

### Types

- `EvalContext` — `UserID`, `Attributes`
- `EvaluationResult` — `Value`, `Version`, `Reason`, `MatchedRuleID`, `Source`
- `EvaluateResponse` — `Results`, `Timestamp`, `CorrelationID`

### Functions

| Function | Endpoint | Use when |
|----------|----------|----------|
| `EvaluateFlags` | `POST .../flags/evaluate` | Per-request decisions (rules, rollout) |
| `EvaluateFlag` | same | Single flag convenience |
| `EvaluatePublicFlags` | `POST .../client/.../evaluate` | Client-side flags from BFF |

See [evaluation-semantics.md](../../assessment/evaluation-semantics.md) for `reason` values.

---

## 8. ServerSDK: local cache + WebSocket

| Method | Purpose | When |
|--------|---------|------|
| `NewServerSDK` | Create cache for a scope | Startup |
| `Refresh` | HTTP load into cache | Startup / fallback |
| `StartWebSocket` | Live sync (`flags.snapshot`, updates) | Backend services |
| `StartPolling` | Periodic HTTP refresh | With WebSocket (recommended) |
| `Snapshot` | Copy local cache | Debug / health |
| `Variation` | Read default only | Global on/off, no per-user rules |

---

## 9. WebSocket message types

| `type` | Effect |
|--------|--------|
| `flags.snapshot` | Replace full local map |
| `flag.updated` / `flag.created` | Upsert if version ≥ current |
| `flag.deleted` | Remove by key |

Requires `BROADCAST_CONSUMER_ENABLED` + RabbitMQ for live dashboard changes.

---

## 10. When to use which feature

- **Per-user / per-country** → `EvaluateFlags` + attributes  
- **Global kill switch** → `Variation()` or evaluate  
- **Keep definitions fresh** → `StartWebSocket` + `StartPolling`  
- **Browser flags** → `ListPublicFlags` / `EvaluatePublicFlags`

---

## 11. Multi-user in one service

Safe when definitions are shared (WebSocket cache) and **evaluate is called per request** with the correct context. flag-gen does not store per-user evaluation results.

---

## 12. Authentication

| | Dev | Production |
|---|-----|------------|
| WebSocket | `BroadcastDevToken` | API key `broadcast:subscribe` |
| Evaluate | optional | `flags:evaluate` |
| Public | `PUBLIC_READ_API_KEY` | same |

---

## 13. Full example

[examples/golang-service](../../../examples/golang-service/) — `GET /flags` (cache) and `POST /evaluate` (full API response).

---

## 14. Anti-patterns

- Global map of evaluate results shared across users  
- `Variation()` for country/plan targeting  
- Empty `attributes` when rules need them  
- Secrets in flag values  

---

## 15. MVP limitations

- No local rule evaluation in SDK — use HTTP evaluate  
- Go only for SDK; Angular uses HTTP  
- `Variation()` is default-only  

---

## 16. API quick reference

See [sdk-flagclient_id.md](sdk-flagclient_id.md#16-referensi-cepat-api) for the full function table (bilingual maintenance: keep both in sync).
