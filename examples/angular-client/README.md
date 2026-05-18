# Angular client-side example

Two HTTP samples in one small app:

| Sample | API | Purpose |
|--------|-----|---------|
| **List flags** | `GET /v1/client/.../flags` | Read client-side flag definitions (`clientSide: true`) |
| **Evaluate** | `POST /v1/client/.../flags/evaluate` | Run targeting/rollout; full flag-gen evaluation response |

No WebSocket — browser-safe HTTP only.

## Setup

1. flag-gen running (`make dev`).
2. Set `projectId` in `src/environments/environment.ts`.
3. Flags must have `clientSide: true` for this example.
4. Restart flag-gen after pulling (needs `POST .../client/.../flags/evaluate` route).

## Run

```bash
cd examples/angular-client
npm install
npm start
```

Open http://localhost:4200

## Sample 1 — list flags

```http
GET /v1/client/projects/{projectId}/environments/{environmentKey}/flags
X-Public-Api-Key: dev-public-key
```

```json
{
  "flags": [
    { "key": "my_flag", "enabled": true, "value": true, "version": 2 }
  ],
  "correlationId": "..."
}
```

## Sample 2 — evaluate

```http
POST /v1/client/projects/{projectId}/environments/{environmentKey}/flags/evaluate
```

```json
{
  "context": { "userId": "user-1", "attributes": { "country": "ID" } },
  "flags": ["my_flag"]
}
```

```json
{
  "results": {
    "my_flag": {
      "value": true,
      "version": 2,
      "reason": "rule_match",
      "matchedRuleId": null,
      "source": "cache_l1"
    }
  },
  "timestamp": "...",
  "correlationId": "..."
}
```
