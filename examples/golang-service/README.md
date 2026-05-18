# Go service-to-service example

Two integration samples in one small HTTP server:

| Sample | Endpoint | Source |
|--------|----------|--------|
| **Flags** | `GET /flags`, `GET /flags/{key}` | Local cache fed by **flag-gen WebSocket broadcast** (`flags.snapshot` + live updates) |
| **Evaluate** | `POST /evaluate`, `GET /evaluate/{key}` | Remote flag-gen `POST .../flags/evaluate` (full response) |

**Broadcast is done by flag-gen**, not by this service polling. After a dashboard change, flag-gen publishes to RabbitMQ and pushes `flag.updated` to WebSocket subscribers. HTTP `Refresh` runs once at startup (before the first snapshot); optional `FLAG_GEN_POLL_FALLBACK=true` only if broker/WS is unavailable.

Uses [`sdk/flagclient`](../../sdk/flagclient/).

## Run

```bash
cd examples/golang-service
cp .env.example .env
export $(grep -v '^#' .env | xargs)
go run .
```

## 1) Sample: flags (WebSocket cache)

```bash
curl -s http://localhost:8080/flags | jq
curl -s http://localhost:8080/flags/seed_demo_toggle | jq
```

Response lists everything in local memory after connect:

```json
{
  "projectId": "...",
  "environmentKey": "development",
  "source": "local_cache",
  "flags": [
    { "key": "seed_demo_toggle", "enabled": true, "clientSide": false, "value": { "default": true }, "version": 1 }
  ]
}
```

## 2) Sample: evaluate (HTTP)

```bash
curl -s -X POST http://localhost:8080/evaluate \
  -H 'Content-Type: application/json' \
  -d '{
    "context": { "userId": "user-1", "attributes": { "country": "ID" } },
    "flags": ["seed_demo_rules"]
  }' | jq
```

Full flag-gen shape:

```json
{
  "results": {
    "seed_demo_rules": {
      "value": true,
      "version": 1,
      "reason": "rule_match",
      "matchedRuleId": "...",
      "source": "cache_l1"
    }
  },
  "timestamp": "...",
  "correlationId": "..."
}
```

## Auth

| Mode | Env |
|------|-----|
| WebSocket dev token | `BROADCAST_DEV_TOKEN` |
| Evaluate API key | `FLAG_GEN_API_KEY` (`flags:evaluate` scope in production) |
