# Flag Gen Dashboard — Regression & UX Report

| Field | Value |
|-------|--------|
| Date | 2026-05-19 (initial QA) · **fix pass 2026-05-19** |
| Target | http://localhost:6124/app/ |
| API | http://localhost:6123/api/ |
| Browser | Cursor IDE browser + prior agent-browser headed Chrome |
| Credentials | `admin@local` / `admin` |
| Reference | [docs/guide/ui-ux-parity.md](../../docs/guide/ui-ux-parity.md) |

## Summary

Dashboard MVP meets core parity when using **real login**. A fix pass on 2026-05-19 addressed release blockers from QA: Dev bypass removed, evaluate result layout, debounced list search (250ms / min 3 chars), audit `q` filter with LIKE on flag key/name/entity id, flag history rollback UI, environment create guard + validation messages, and docs clarifying Project ID (UUID) vs project key.

**Remaining gaps (deferred):** no Update UI for projects/environments; read-only users still see mutation controls (server 403); UX polish (empty states, form placement, i18n mix).

---

## Fix pass (2026-05-19)

| Item | Status |
|------|--------|
| Remove Dev bypass (login-only session) | **Fixed** |
| `.evaluate-result` layout CSS | **Fixed** (global `styles.scss`) |
| List search debounce 250ms + min 3 chars | **Fixed** (`core/list-search.ts`, projects/envs/flags/segments) |
| Audit search `q` + LIKE (BE + FE) | **Fixed** |
| Flag history rollback button + API (`FindByKey`) | **Fixed** (first E2E hit 404; patched [`rollback_flag.go`](application/usecase/rollback_flag.go)) |
| Environment create race / validation copy | **Fixed** |
| Docs: UUID in quickstart/README | **Fixed** |

Verification: `go test ./...` pass; `npm run build` (web) pass.

### Browser retest (2026-05-19, post-fix)

| Check | Result |
|-------|--------|
| Login page: no Dev bypass button | **PASS** |
| Real login → projects API 200 | **PASS** |
| Projects search `qa` (2 chars) → `q=` empty, both projects | **PASS** |
| Projects search `qa_ui` (5 chars) → single `GET …?q=qa_ui`, one row | **PASS** |
| Audit search `qa_crud_flag` → `GET …/audit?q=qa_crud_flag`, rows returned | **PASS** |
| Evaluate modal: VALUE / REASON / SOURCE / VERSION separated | **PASS** (see `screenshots/browser-flags-evaluate-result-fixed.png`) |
| Flag history: "Rollback to this version" button visible | **PASS** |
| Rollback E2E: v1 → current v2, confirm `qa_crud_flag` | **PASS** (`POST …/rollback` **200**, history refreshed) |
| Environment create button enabled when project loaded | **PASS** |

---

## Test environment

- DB state: `db_ready` (`GET /v1/system/status`)
- Seeded project: `development`
- Scratch data from prior session may still exist: `qa_ui_901`, etc.

---

## CRUD coverage (post-fix)

| Area | Create | Read/List | Update | Delete | Notes |
|------|--------|-----------|--------|--------|-------|
| Projects | PASS | PASS | **No UI** | PASS | Search debounced |
| Environments | PASS | PASS | **No UI** | Not re-tested | Create disabled until projectId loaded |
| Flags | PASS | PASS | PASS | Not re-tested | Evaluate result panel styled |
| Segments | PASS | PASS | PASS | Not re-tested | Search debounced |
| Flag history | N/A | PASS | N/A | Rollback | Rollback to version + confirm |
| Audit | N/A | PASS | N/A | N/A | Toolbar `q` + debounce |
| Login | PASS | — | — | — | Dev bypass removed |

---

## Search behavior (post-fix)

| Surface | API | Debounce | Min chars | Backend filter |
|---------|-----|----------|-----------|----------------|
| Projects / Environments / Flags / Segments | `GET …?q=` | **250ms** | **3** (shorter → full list) | `LIKE %q%` on key + name |
| Audit toolbar | `GET …/audit?q=` | **250ms** | **3** | `entity_id` exact or LIKE on id/key/name |
| Audit project picker | `GET /projects?q=` | **250ms** | none | `LIKE %q%` on key + name |

---

## Bugs (original QA → resolution)

| Sev | Issue | Resolution |
|-----|--------|------------|
| **High** | Dev bypass → API 401 | Removed bypass; stale `dev` marker cleared on load |
| **High** | Evaluate result labels merge | Global `.evaluate-result` grid styles |
| **Medium** | Audit search exact `entity_id` only | Added `q` param + LIKE; FE uses debounced `q` |
| **Medium** | List search no debounce/min length | `list-search.ts` + wired on list pages |
| **Medium** | Docs use project key in paths | quickstart/README use `PROJECT_ID` UUID |
| **Low** | Vague environment validation | Field-specific messages (`projectId is required`, etc.) |
| **Low** | No project/env update UI | **Open** |
| **Low** | Read-only UI not gated | **Open** (server 403 only) |

---

## Final status

| Gate | Result |
|------|--------|
| Core CRUD (authenticated) | **PASS** (no project/env update UI) |
| Search | **PASS** (debounced, min 3, BE-backed) |
| Dev bypass | **N/A** (removed) |
| Release-ready polish | **Mostly yes** — optional RBAC UI + CRUD update remain |

### Optional follow-ups

1. Permission-aware UI for read-only roles.
2. PATCH + edit UI for projects/environments.
3. Manual re-screenshot evaluate panel after deploy.
4. Cleanup scratch QA entities (`qa_ui_901`, etc.).
