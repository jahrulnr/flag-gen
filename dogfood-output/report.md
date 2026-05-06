# Flag Gen Dashboard E2E Regression Report

Date: 2026-05-07
Target: http://localhost:6124/app/
Browser: agent-browser headed Chrome
Credentials: admin@local / admin

## Summary

In progress.

## Coverage Log

- Login with seed admin: PASS (`01-projects.png`).
- Projects list loaded with sidebar, breadcrumb, seeded `development` project, no page errors.
- Project create workflow: PASS (created e2e_014210 and auto-opened Environments, `02-project-created.png`).
- Environment create workflow: PASS (`03-environment-created.png`).
- Flags empty-state and create form opened: PASS (`04-flags-empty.png`, `05-flag-create-form.png`).
- Flag create workflow: PASS (`06-flag-created.png`).
- Flag history read model: PASS (`07-flag-history.png`).
- Segments empty-state and create workflow: PASS (`08-segments-empty.png`, `09-segment-created.png`).
- Roles page and built-in role presets: PASS (`10-roles-page.png`).
- Custom role create workflow: PASS (`11-role-created.png`).
- Users page with role dropdown: PASS (`12-users-page.png`).
- User create workflow with read-only role: PASS (`13-user-created.png`).
- Read-only login and forbidden mutation enforcement: PASS (`14-readonly-login.png`, `15-readonly-forbidden.png`).
- UX note: read-only users still see mutation/admin controls; backend correctly rejects them, but UI should ideally hide or disable controls by permission.
- Audit query/read model: PASS (`16-audit-page.png`).
- Runtime flag list/evaluate via browser: PASS when using project UUID (`19-runtime-evaluate-project-id-json.png`).
- API/docs note: using the project key in the runtime URL returned no flags; local docs currently show key-style examples, while the API path behaved as project UUID in this test (`17-runtime-evaluate-json.png`, `18-runtime-list-json.png`).
- Lifecycle route sanity: OBSERVED redirect to `/app/login` when opening `/app/lifecycle` directly (`20-lifecycle.png`).
- Cleanup: PASS. Temporary project/flag/segment soft-deleted, temporary read-only user disabled, temporary custom role deleted.

## Findings

1. Medium: Runtime API works with project UUID, but the README/OpenAPI examples use project keys. Browser sanity with project key returned an empty flag list/not_found evaluation; project UUID returned the created flag and correct evaluation result.
2. Low: Read-only users still see mutation/admin controls in the dashboard. Backend RBAC rejects write/admin actions with 403, but UI should hide or disable those controls based on permissions.
3. Low: Direct `/app/lifecycle` navigation redirected to `/app/login` in the dev browser session, so lifecycle status was not reachable as a standalone page during this run.

## Final Status

Regression/sanity completed with headed agent-browser. Core create/read flows and RBAC enforcement passed; findings above are follow-up UX/docs/routing issues.
