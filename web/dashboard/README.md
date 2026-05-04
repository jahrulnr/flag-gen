# Angular admin dashboard (scaffold)

This repository ships the **Go API** first. For the GrowthBook-style Angular UI referenced in `docs/plan.md`, scaffold a sibling app:

```bash
cd web/dashboard
npx -y @angular/cli@19 new flag-gen-admin --defaults --routing=true --style=scss --skip-git
```

Then:

1. Call `GET /v1/system/status` before protected routes (install / migrate gating per `docs/assessment/database-lifecycle.md`).
2. Point the Angular environment `apiBaseUrl` to `http://localhost:8080/v1`.
3. Forward `X-Correlation-ID` on all requests; display it on error surfaces.
4. After Wave 4, subscribe to `ws://.../v1/broadcast/flags` with `BROADCAST_DEV_TOKEN` for realtime updates.

Until the SPA is generated, this folder intentionally contains only this README.
