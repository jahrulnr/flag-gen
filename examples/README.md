# Flag Gen — Integration Examples

Docs: [guide](../docs/guide/README.md) · [assessment](../docs/assessment/README.md)

Each example demonstrates **two flows**:

| Flow | Go (backend) | Angular (browser) |
|------|----------------|-------------------|
| **Flags** | WebSocket → local cache → `GET /flags` | `GET /v1/client/.../flags` |
| **Evaluate** | `POST /evaluate` → flag-gen evaluate API | `POST /v1/client/.../flags/evaluate` |

| Project | Path |
|---------|------|
| Go service-to-service | [golang-service](./golang-service/) |
| Angular client-side | [angular-client](./angular-client/) |

## Quick start

```bash
# flag-gen must be running (make dev)
cd examples/golang-service && go run .
cd examples/angular-client && npm start
```

See each project's README for curl examples and response shapes.
