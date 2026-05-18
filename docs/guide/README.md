# Guide

Everything you need to **use** flag-gen: concepts, integration, and day-to-day operations. For implementer specs, see [../assessment/README.md](../assessment/README.md).

## Getting started

| Document | Contents |
|----------|----------|
| [quickstart.md](quickstart.md) | Local setup, env vars, curl, WebSocket dev token |
| [how-it-works.md](how-it-works.md) | Flag definitions vs evaluate, broadcast, cache path |

## Concepts

Feature-flag ideas (not flag-gen–specific wire formats):

| Language | Document |
|----------|----------|
| Indonesian | [concepts/feature-flags_id.md](concepts/feature-flags_id.md) |
| English | [concepts/feature-flags.md](concepts/feature-flags.md) |

## SDK (Go)

| Language | Document |
|----------|----------|
| Indonesian (full) | [sdk/sdk-flagclient_id.md](sdk/sdk-flagclient_id.md) |
| English (summary) | [sdk/sdk-flagclient.md](sdk/sdk-flagclient.md) |

Package source: [`sdk/flagclient`](../../sdk/flagclient/) · Examples: [../../examples/](../../examples/)

## Operations

| Document | Contents |
|----------|----------|
| [metrics.md](metrics.md) | Metrics and alert hints |
| [transport-security.md](transport-security.md) | TLS and reverse proxy |
| [ui-ux-parity.md](ui-ux-parity.md) | Dashboard UX notes |
| [implementation-status.md](implementation-status.md) | Repo implementation checklist |

## Specs & API

- [../assessment/api-contract.md](../assessment/api-contract.md)
- [../assessment/evaluation-semantics.md](../assessment/evaluation-semantics.md)
- [../api/openapi.yaml](../api/openapi.yaml)
