# Flag Gen admin dashboard

Single Angular SSR app for the Flag Gen admin UI. Source and tooling live at the repo root under `web/`.

## Runtime model

- Production: Go listens on `:6123`, owns `/api/*`, and proxies `/app/*` to the Angular SSR renderer.
- Development: Angular SSR dev server listens on `:6124/app/*` and calls Go at `:6123/api/*`.
- Do not use hash routing for SSR; `/app/*` is path-based so the SSR renderer can see the route.

## Local commands

From the **repository root**, API and dashboard together:

```bash
make dev
```

Open the app at **`http://localhost:6124/app/`** (dev server uses `--serve-path=/app/`). `make dev` runs **`dev-backend`** and **`dev-frontend`** in parallel (`make -j2`); stop with **Ctrl+C** (may need one press per job depending on your terminal).

Dashboard only (expects API already on `:6123`):

```bash
cd web
npm start
```

Build, SSR preview, and tests:

```bash
npm run build
npm run serve:ssr
npm test -- --watch=false
```

The Angular environment uses:

- Dev API: `http://localhost:6123/api`
- Prod API: `/api`

Configure the Node SSR server with `API_ORIGIN` when the API is not same-origin (see `src/server.ts`).
