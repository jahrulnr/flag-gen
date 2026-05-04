---
name: web-admin-dashboard-qa
description: >-
  End-to-end and exploratory QA for the Angular Flag Gen Admin dashboard (base
  `/app/`). Defines verified URL flows, a non-negotiable UX quality bar
  (destructive confirmations, human-readable errors, no blocking overlays), and
  regression checks. Use when manually testing the admin UI, validating
  dashboard PRs, or when the user mentions UI regression, E2E admin flows,
  browser verification of `web/`, UX requirements for flags/segments/audit,
  read-model tables, or human-friendly identifiers vs raw UUIDs.
---

# Web admin dashboard QA (Flag Gen)

## UX bar (non-negotiable)

These are **release blockers** for admin UX, not polish tickets. Fail the QA pass if any are violated.

1. **No decorative layer may intercept clicks** on primary or destructive controls (e.g. floating owl / Lottie canvas over row **Delete**). Floating help must sit in a **non-overlapping** corner (product default: **bottom-left**) or use a hit target that does not cover list row actions; use `pointer-events` and `z-index` so list actions always win.
2. **No destructive delete without a serious confirmation:** modal (not a one-line native `confirm` alone for high-risk resources), explicit consequence copy, and **type-to-confirm** by re-entering the resource **`key`** before enabling **Permanent delete**. Apply consistently to **flag**, **segment**, **environment**, and **project** (same or stricter bar everywhere).
3. **No raw API error codes** as the only user-visible message (e.g. `VALIDATION_ERROR`). Map codes + optional `details` to **actionable human language**; keep `correlationId` secondary (e.g. footer / “report to admin”).
4. **Edit vs create** must be obvious at a glance on every entity page that reuses a top form (banner with resource key/name, or equivalent), including **Segments** parity with **Flags**.
5. **UUID / technical inputs** (e.g. Audit Project ID): disable browser spellcheck artifacts (`spellcheck="false"`, etc.) so the field does not look “broken”.
6. **Operator-readable tables:** for any admin **list or audit table**, the **primary** cell text must not be a naked UUID (or `type / uuid` only) when the product also has **key** or **display name** for that entity. Prefer **name or key as headline**, UUID (or id) **secondary** (smaller, monospace, truncated, expandable, or “copy id”). Mismatch across pages (e.g. **Flags** show flag **key**, **Audit** shows only flag **UUID** for the same row concept) is a **consistency fail** unless explicitly documented as API limitation with mitigation (tooltip, link to entity, etc.).

Authoritative write-up for product language and priorities: **`docs/ui-ux-qa-summary-and-requirements.md`**.

## Preconditions

- API + admin UI running so the SPA resolves (typical dev: Angular dev server proxying to the Go API; session used `http://localhost:6124` as origin).
- App is served with **`baseHref` `/app/`** (`web/angular.json`). All paths below are **relative to origin** (e.g. `http://localhost:6124/app/dashboard/projects`).

## Route map (source of truth)

From `web/src/app/app.routes.ts`:

| User-facing flow | Path pattern |
|------------------|--------------|
| DB lifecycle / install gate | `/app/lifecycle` |
| Default dashboard entry | `/app/dashboard` → redirects to **`/app/dashboard/projects`** |
| Projects | `/app/dashboard/projects` |
| Environments for a project | `/app/dashboard/projects/:projectId/environments` |
| Flags for an environment | `/app/dashboard/projects/:projectId/environments/:environmentKey/flags` |
| Flag change history | `/app/dashboard/projects/:projectId/environments/:environmentKey/flags/:flagId/history` |
| Segments (project scope, not nested under an environment route) | `/app/dashboard/projects/:projectId/segments` |
| Audit (read model search) | `/app/dashboard/audit` |

Wild route **`/**`** redirects to **`/app/lifecycle`**.

**Lifecycle redirect (expected):** when the backend reports DB ready / optional migration, `LifecyclePage` navigates to **`/dashboard/projects`** (in-app), i.e. user ends on `/app/dashboard/projects` without seeing install UI. Do not file that as a broken route when the stack is already bootstrapped.

## How to obtain IDs (do not guess UUIDs)

- **`projectId`:** Open Projects; use the **Environments** (or equivalent) navigation target from a project row, or copy from the URL after navigation (UUID in path).
- **`environmentKey`:** URL segment after `.../environments/` (e.g. `development`), not necessarily the display name casing.
- **`flagId`:** From Flags list, follow **History** on a row; `flagId` is the UUID in the resulting URL.

## Verified primary E2E chain (happy path)

Execute in order; each step assumes the previous succeeded.

1. **`GET` UI `/app/dashboard/projects`** — Projects list and create-project form visible; at least one project row if seeded.
2. **Drilldown:** From a project row, open **Environments** → URL matches `.../projects/{projectId}/environments`.
3. **Drilldown:** From an environment row, open **Manage flags** (or equivalent) → URL matches `.../environments/{environmentKey}/flags`.
4. **Flags row actions:** **History** on a flag → URL matches `.../flags/{flagId}/history`; page title/heading indicates flag history.
5. **Segments (same project):** Navigate to **`/app/dashboard/projects/{projectId}/segments`** (sidebar or direct URL) — segment list / create form loads for that `projectId`.
6. **Audit:** Open **`/app/dashboard/audit`** — exercise **product-intended** scope: (a) if the spec is **“full log for the workspace”**, assert rows load **without** forcing an extra “search” step beyond picking scope (or document gap); (b) if the spec is **“search-gated”**, assert empty vs results states and that **first successful search** returns the **full** window the API promises (e.g. limit, pagination). Always verify table updates after filter change.
7. **Wiki:** On a page that exposes it, open **Wiki** — in-page help content appears; close control works.

## Audit & read-model pages (sharp checks)

Use this block so Audit is not reduced to “form renders + Search works”.

- **Scope vs empty state:** Distinguish **(1)** “no query yet” placeholder, **(2)** “queried, zero rows”, and **(3)** “data present”. Mislabeling (e.g. implying user error when the product should have auto-loaded scope) is a **copy/IA fail**.
- **“Show all audit” expectation:** If stakeholders expect **every audit row** for a project (or tenant) to be visible **by default** after choosing that scope, then **requiring a UUID paste + Search as the only path to see any rows** is a **product/UX gap** — log it with: expected behavior, actual behavior, and whether backend supports list-without-filter or needs default `projectId` from shell context.
- **Entity column semantics:** Cells that render only `entityType / entityId` with **entityId = UUID** are weak for operators. Sharp pass = at least one of: **key**, **name**, **link to the entity screen**, or **progressive disclosure** (expand row / tooltip) populated when the read model or join allows it.
- **Correlation column:** Long correlation UUID as the **only** readable token is heavy. Pass with truncation + copy, or secondary styling; fail if it dominates the row without affordance.
- **Created column:** Raw ISO vs localized time — note mismatch with **Flag history** formatting for consistency across read models.
- **Cross-surface identity:** Same logical object (project, flag, segment) should use the **same primary label** (key or name) wherever the operator compares screens (Audit vs Flags vs URL). Flagging **UUID here / key there** sharpens regression reports.

## Secondary checks (regression-focused)

- **Tables (Audit, Flag history):** Header and cell text must stay readable; see `docs/ui-ux-regression-report-2026-05-04.md` if present. Apply **Audit & read-model pages** and **UX bar** #6 for column content, not only contrast.
- **Breadcrumbs:** Human labels; avoid meaningless duplicate segments; UUID-only crumbs without labels are a fail for orientation.
- **Information architecture:** Audit under **System** in the shell must match breadcrumb (`SYSTEM > Audit` or equivalent).
- **Top-of-page create forms:** After **Edit**, confirm **UX bar** #4 (clear editing state).
- **Destructive flows:** Verify **UX bar** #2 (modal + type key + disabled until match) for every deletable resource the page exposes.
- **Floating wiki / owl:** Verify **UX bar** #1 on short viewports and last list rows.
- **Connectivity:** Header shows expected connected/degraded state if the shell exposes it.
- **Console:** For production-readiness passes, note unexpected errors; dev-only Angular/Vite messages are informational.

## Deliverables after a QA pass

- Short note: routes exercised, build/commit, environment, and pass/fail per step above, explicitly calling out **UX bar** violations.
- Product-facing UX backlog and rationale: **`docs/ui-ux-qa-summary-and-requirements.md`**.

## Automation note

Automated Playwright/Cypress should use the same path templates and resolve `projectId` / `flagId` from API or UI navigation, not hard-coded demo UUIDs. Assert **no element** with decorative animation sits above `[aria-label*="Delete"]` row actions in hit-test order. For **Audit**, assert after a successful load that **entity** cells are not exclusively bare UUIDs when product rules require key/name (or assert documented exception).
