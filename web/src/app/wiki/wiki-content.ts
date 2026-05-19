export type WikiId =
  | 'lifecycle'
  | 'projects'
  | 'environments'
  | 'segments'
  | 'flags'
  | 'flag-history'
  | 'audit'
  | 'users'
  | 'roles'
  | 'wiki-guide'
  | 'wiki-flag-gen'
  | 'developer-wiki';

export interface WikiSection {
  title: string;
  bullets: string[];
}

export interface WikiContent {
  id: WikiId;
  eyebrow: string;
  title: string;
  summary: string;
  sections: WikiSection[];
}

export const WIKI_CONTENT: Record<WikiId, WikiContent> = {
  lifecycle: {
    id: 'lifecycle',
    eyebrow: 'Before you enter',
    title: 'Database lifecycle & admin access',
    summary:
      'This page gates the dashboard. It checks whether MySQL is reachable, empty, migrated, or blocked — and offers install or migrate actions when allowed.',
    sections: [
      {
        title: 'What this page does',
        bullets: [
          'Calls GET /v1/system/status and shows database.state (db_empty, db_ready, migration_required, etc.).',
          'Blocks other dashboard routes until the database is in a safe state for management.',
          'Also applies to Users and Roles: you need a ready database before managing access control.',
        ],
      },
      {
        title: 'How to: first-time local setup',
        bullets: [
          '1. Start MySQL and create the flag_gen database (see repo quickstart).',
          '2. Open this page — if state is db_empty, fill Install (admin email, password, optional seed project/environment keys).',
          '3. Submit Install, wait for success, then confirm state becomes db_ready.',
          '4. Log in to the dashboard and open Projects.',
        ],
      },
      {
        title: 'How to: after pulling new migrations',
        bullets: [
          '1. Refresh status — if migration_required appears, dashboard writes are blocked.',
          '2. Click Run migration during a maintenance window.',
          '3. Refresh again until db_ready (or migration_optional).',
          '4. Verify flags and environments still load as expected.',
        ],
      },
      {
        title: 'Install form fields',
        bullets: [
          'Email / Name / Password — first dashboard admin user.',
          'Project key / Environment key — optional seed scope (e.g. development / development) created during install.',
          'These keys are human labels; integrations still need the Project ID UUID from the Projects page later.',
        ],
      },
      {
        title: 'When status is not db_ready',
        bullets: [
          'db_unreachable — fix MYSQL_DSN, network, or credentials on the API server.',
          'db_invalid / db_ahead — schema mismatch; align migrations with the deployed app version.',
          'Do not run Install on a database that already has data — use migrate instead.',
        ],
      },
      {
        title: 'Dokumentasi Wiki',
        bullets: ['Setelah db_ready: Wiki › Panduan untuk tutorial dashboard.'],
      },
    ],
  },
  projects: {
    id: 'projects',
    eyebrow: 'Workspace setup',
    title: 'Projects — organize flags by product',
    summary:
      'A project is the top-level scope for environments, segments, and flags. Each project has a human key and a stable Project ID (UUID) for APIs.',
    sections: [
      {
        title: 'What this page does',
        bullets: [
          'Lists all projects with search and pagination.',
          'Creates projects (key, name, description).',
          'Links to Environments and Segments for each project.',
          'Shows Project ID on every row — required for SDK, WebSocket, and HTTP integrations.',
        ],
      },
      {
        title: 'How to: create a project',
        bullets: [
          '1. Enter Key — short stable identifier (e.g. checkout, mobile_app). Lowercase with underscores is typical.',
          '2. Enter Name — display name for operators.',
          '3. Optional Description — ownership or purpose for your team.',
          '4. Click Create project — you are redirected to Environments for that project.',
          '5. Copy the Project ID from the list before configuring external services.',
        ],
      },
      {
        title: 'API & SDK identifiers (read this for integrations)',
        bullets: [
          'Project key — short label you choose (e.g. development). Shown in this list; for humans only, not in API paths today.',
          'Project ID — UUID on each row under “Project ID”. Required for /v1/projects/:projectId/..., WebSocket projectId=, and FLAG_GEN_PROJECT_ID.',
          'Environment integrations use environment key (e.g. production) in the path — see Environments wiki (? on that page).',
          'Dokumentasi lengkap: sidebar Wiki › Panduan / Konsep Flag Gen / Developer.',
          'SDK: docs/guide/sdk/sdk-flagclient_id.md di repo.',
        ],
      },
      {
        title: 'Field guide',
        bullets: [
          'Key — immutable in practice; pick carefully. Used in communication, not in API paths today.',
          'Name — can be renamed later for clarity.',
          'Description — optional; helps new operators understand scope.',
        ],
      },
      {
        title: 'Before deleting a project',
        bullets: [
          'Delete all flags in every environment, then environments, then segments.',
          'Confirm no running service still uses the Project ID in config.',
          'Deletion is permanent after the confirm dialog (type the project key to confirm).',
        ],
      },
      {
        title: 'Dokumentasi Wiki',
        bullets: [
          'Wiki › Panduan — tutorial dashboard dan penjelasan menu.',
          'Wiki › Konsep Flag Gen — konsep feature flag.',
          'Wiki › Developer — integrasi SDK.',
        ],
      },
    ],
  },
  environments: {
    id: 'environments',
    eyebrow: 'Release contexts',
    title: 'Environments — separate flag state per stage',
    summary:
      'Environments (production, staging, development) hold independent flag definitions under one project. Open this wiki (?) for API identifiers needed by SDKs.',
    sections: [
      {
        title: 'What this page does',
        bullets: [
          'Lists environments for the current project (from the URL).',
          'Creates environments with key, name, and sort order.',
          'Opens Manage flags for each environment key.',
        ],
      },
      {
        title: 'API & SDK identifiers for this project',
        bullets: [
          'Project key — human label; find it on Projects list or in the browser URL context (not used in API paths).',
          'Project ID — UUID from Projects page (each row shows “Project ID”). Paste into FLAG_GEN_PROJECT_ID or SDK constructor.',
          'Environment key — the Key column on this page (e.g. development). Used in .../environments/{environmentKey}/flags.',
          'Example: GET /v1/projects/{project-uuid}/environments/staging/flags',
          'WebSocket: /v1/broadcast/flags?projectId={uuid}&environmentKey=staging&token=...',
        ],
      },
      {
        title: 'How to: set up environments',
        bullets: [
          '1. Start with production (or your primary live key).',
          '2. Add staging / qa / development with distinct keys.',
          '3. Use Sort order to control list order (lower numbers first).',
          '4. Click Manage flags to define flags only in that environment.',
          '5. Note Project ID from Projects page before wiring services.',
        ],
      },
      {
        title: 'Field guide',
        bullets: [
          'Key — appears in API paths: .../environments/{environmentKey}/flags. Use short lowercase names.',
          'Name — operator-friendly label (e.g. Production).',
          'Sort order — numeric; optional ordering in lists.',
        ],
      },
      {
        title: 'How environments relate to integrations',
        bullets: [
          'Example API path: GET /v1/projects/{project-id}/environments/staging/flags',
          'WebSocket: .../broadcast/flags?projectId={uuid}&environmentKey=staging',
          'Same flag key can exist in multiple environments with different enabled/value/rules.',
        ],
      },
      {
        title: 'Before deleting',
        bullets: [
          'Remove every flag in the environment first.',
          'Update SDKs and services so they no longer call that environment key.',
          'Confirm with the team — deletes are destructive.',
        ],
      },
      {
        title: 'Dokumentasi Wiki',
        bullets: ['Wiki › Panduan · Konsep Flag Gen · Developer di sidebar.'],
      },
    ],
  },
  segments: {
    id: 'segments',
    eyebrow: 'Targeting',
    title: 'Segments — reusable audience definitions',
    summary:
      'Segments store named rule sets at project level. Use them to document audiences; flag targeting rules on the Flags page reference attributes directly.',
    sections: [
      {
        title: 'What this page does',
        bullets: [
          'CRUD segments under the current project (key, name, rules JSON).',
          'Segments are project-scoped, not per-environment.',
        ],
      },
      {
        title: 'API scope (project-level)',
        bullets: [
          'Segments do not have their own environment key — they belong to the current project in the URL.',
          'Integrations still need Project ID (UUID) from the Projects page for any flag or evaluate API under this project.',
          'Segment keys are for dashboard organization only unless you reference them in flag rules via attributes.',
        ],
      },
      {
        title: 'How to: create a segment',
        bullets: [
          '1. Choose a clear Key (e.g. beta_users, internal_staff).',
          '2. Set Name for the dashboard.',
          '3. Enter Segment rules as a JSON array (same shape as flag targeting rules).',
          '4. Click Create segment.',
          '5. When editing flags, align attribute names with segment rules for consistency.',
        ],
      },
      {
        title: 'Segment rules JSON',
        bullets: [
          'Must be a JSON array. Use [] for an empty segment placeholder.',
          'Each rule: attribute, operator (equals, not_equals, in, not_in), value, serve, priority.',
          'Example: [{"attribute":"plan","operator":"in","value":["beta","internal"],"serve":true,"priority":0}]',
          'For in/not_in, value must be a JSON array.',
        ],
      },
      {
        title: 'How segments relate to flags',
        bullets: [
          'Flag evaluation uses targeting rules on the flag itself, evaluated against evaluate context attributes.',
          'Segments help teams agree on audience definitions; copy rule patterns into flags as needed.',
          'Use the same attribute names your apps send in POST .../flags/evaluate (e.g. country, plan, tier).',
        ],
      },
      {
        title: 'Operational tips',
        bullets: [
          'Edit instead of duplicating similar segments.',
          'Search the segment list before creating a new key.',
          'Before delete: check flags that might reference the same attribute logic.',
        ],
      },
      {
        title: 'Dokumentasi Wiki',
        bullets: ['Wiki › Konsep Flag Gen — kapan memakai segment vs flag rules.'],
      },
    ],
  },
  flags: {
    id: 'flags',
    eyebrow: 'Feature control',
    title: 'Flags — defaults, rollout, targeting, evaluate',
    summary:
      'Flags control feature behavior per environment. Use the form for definitions; use Evaluate to test what a specific user would receive at runtime.',
    sections: [
      {
        title: 'What this page does',
        bullets: [
          'Lists flags for one project + environment (scope is in the page eyebrow / URL).',
          'Create / Edit flag: enabled, client-side, value type, default, rollout %, targeting rules.',
          'Evaluate — dry-run POST .../flags/evaluate with userId and attributes JSON.',
          'History — version timeline for a flag.',
          'Live changes propagate to backend SDKs via WebSocket when outbox + broadcast are enabled.',
        ],
      },
      {
        title: 'API & SDK identifiers for this page',
        bullets: [
          'Project ID — UUID from Projects list (not the project key). Required in all /v1/projects/:projectId/... paths.',
          'Environment key — from the URL and page context (e.g. development). Same key as on Environments → Manage flags.',
          'Flag key — the code identifier per row (e.g. checkout_v2); used in evaluate and broadcast payloads.',
          'Example evaluate: POST /v1/projects/{uuid}/environments/development/flags/evaluate with body context.userId + context.attributes.',
          'Backend cache/WebSocket uses the same projectId + environmentKey pair.',
        ],
      },
      {
        title: 'How to: create a flag (safe rollout)',
        bullets: [
          '1. Key and Name — stable key (checkout_v2), readable name.',
          '2. Leave Enabled on only when ready to serve; off returns default with reason disabled.',
          '3. Client-side — on only if the value is safe in browser/mobile SDKs.',
          '4. Value type Boolean or JSON; set default to current production behavior first.',
          '5. Rollout % — start at 0, then 10 → 50 → 100; stickiness usually userId.',
          '6. Targeting rules [] until you need attribute-based overrides.',
          '7. Save, then use Evaluate to verify for test users before widening rollout.',
        ],
      },
      {
        title: 'Form fields explained',
        bullets: [
          'Enabled — master switch; false forces default value (flag is off).',
          'Client-side — allows /client and /public reads; never put secrets in values.',
          'Default value — boolean toggle or JSON textarea; this is served when no rule matches and rollout misses.',
          'Rollout % — stable percentage bucket; requires stickiness attribute (default userId).',
          'Targeting rules — ordered JSON rules; evaluated before rollout. [[popup:View example]]\n[\n  {"attribute":"country","operator":"equals","value":"ID","serve":true,"priority":0},\n  {"attribute":"plan","operator":"in","value":["test","beta"],"serve":true,"priority":1}\n]',
        ],
      },
      {
        title: 'How to: use Evaluate',
        bullets: [
          '1. Click Evaluate on a flag row.',
          '2. Enter User ID — any string; used for rollout bucketing when stickiness is userId.',
          '3. Attributes — JSON object, e.g. {"country":"ID","plan":"beta"}. Must match rule attribute names.',
          '4. Run evaluation — inspect value, reason (disabled, rule_match, rollout, default), source, version.',
          '5. reason rule_match shows matchedRuleId when a targeting rule hit.',
        ],
      },
      {
        title: 'Evaluation order (runtime)',
        bullets: [
          '1. If disabled → default, reason disabled.',
          '2. Targeting rules by priority (lowest number first).',
          '3. Rollout percentage bucket.',
          '4. Else default, reason default.',
          'Evaluate tests the real server path — not the same as reading raw flag JSON from cache.',
        ],
      },
      {
        title: 'Common mistakes',
        bullets: [
          'Using Variation()/local cache only in backend — use Evaluate API for per-user rules and rollout.',
          'JSON default left as [object Object] — use valid JSON in the textarea after edit.',
          'Client-side on for flags containing internal config.',
          'Using project key instead of Project ID in SDK config — copy UUID from Projects page (wiki ? on Projects).',
        ],
      },
      {
        title: 'Dokumentasi Wiki',
        bullets: [
          'Wiki › Konsep Flag Gen — definitions vs evaluate.',
          'Wiki › Developer — WebSocket dan evaluate API.',
        ],
      },
    ],
  },
  'flag-history': {
    id: 'flag-history',
    eyebrow: 'Audit trail',
    title: 'Flag history — version timeline',
    summary:
      'Every create, update, and delete on a flag records a version row. Use this page to see what changed and when — especially during incidents.',
    sections: [
      {
        title: 'What this page does',
        bullets: [
          'Lists flag_versions for one flag (version number, change type, timestamp).',
          'Opened from Flags → History on a specific flag.',
          'Rollback — use Rollback to this version on an older row (confirm with flag key); creates a new version from that snapshot.',
        ],
      },
      {
        title: 'How to: investigate a bad deploy',
        bullets: [
          '1. Note the incident time and affected flag key.',
          '2. Open History for that flag.',
          '3. Find the version row just before symptoms started — check change type (update, rollback, etc.).',
          '4. Open Edit on Flags and restore intended enabled, default, rules, or rollout.',
          '5. Re-run Evaluate for affected userIds to confirm.',
        ],
      },
      {
        title: 'Reading the table',
        bullets: [
          'Version — monotonic integer; higher is newer.',
          'Change — create, update, delete, or rollback semantics from the server.',
          'Created — UTC timestamp of the recorded version.',
        ],
      },
      {
        title: 'Rollback thinking',
        bullets: [
          'History is evidence, not a one-click undo button in MVP.',
          'Document manual fixes in your incident channel.',
          'Pair with Audit page (filter entity type flag) for who changed what.',
        ],
      },
    ],
  },
  audit: {
    id: 'audit',
    eyebrow: 'Read model',
    title: 'Audit — who changed what',
    summary:
      'Search audit log entries by project (required), optional entity type, and optional entity id. Correlation IDs link to API request logs.',
    sections: [
      {
        title: 'What this page does',
        bullets: [
          'Read-only view of audit_log rows for a selected project.',
          'Shows entity, action, correlation id, and timestamp.',
          'Copy correlation ID to trace requests in server logs.',
        ],
      },
      {
        title: 'How to: run a search',
        bullets: [
          '1. Open Project picker — search by name or key; selection stores the Project ID (UUID).',
          '2. Optionally set Entity type (flag, environment, segment, project, ...).',
          '3. Optionally use the toolbar search (min 3 characters) — flag key, flag name, or entity UUID.',
          '4. Click Search (toolbar refresh also re-runs).',
          '5. Use Copy on correlation ID when escalating to engineering.',
        ],
      },
      {
        title: 'Filters explained',
        bullets: [
          'Project — required; must be the UUID from the picker (shown in picker meta).',
          'Entity type — narrows to one resource kind.',
          'Entity search — matches flag key, flag name, or exact entity UUID (debounced, server-side).',
        ],
      },
      {
        title: 'Common issues',
        bullets: [
          'No results — confirm project is selected and Search was clicked.',
          'Wrong project — audit is per project; switch picker and search again.',
          'Empty correlation — older rows may lack IDs; use timestamp + entity instead.',
        ],
      },
      {
        title: 'Dokumentasi Wiki',
        bullets: ['Wiki › Panduan — posisi Audit di sidebar (System).'],
      },
    ],
  },
  users: {
    id: 'users',
    eyebrow: 'Access control',
    title: 'Users — dashboard accounts',
    summary:
      'Manage who can sign in to the flag-gen dashboard. Each user has one role; permissions come from that role.',
    sections: [
      {
        title: 'What this page does',
        bullets: [
          'Lists dashboard users (email, name, role, status).',
          'Creates users with email, name, password, and role.',
          'Updates name, role, and status; email is fixed after create.',
        ],
      },
      {
        title: 'How to: add a user',
        bullets: [
          '1. Ensure a suitable Role exists on the Roles page.',
          '2. Enter Email (login id) and Name.',
          '3. Set Password (min 8 characters) — share through your team secret channel.',
          '4. Select Role — defines permissions (flags:write, admin:*, etc.).',
          '5. Status Active — set Inactive to block login without deleting history.',
          '6. Submit Create user.',
        ],
      },
      {
        title: 'How to: change access',
        bullets: [
          '1. Click Edit on the user row.',
          '2. Change Role or Status; update Name if needed.',
          '3. Save — email cannot be changed (create a new user if email was wrong).',
          '4. For password reset, use your operational process (MVP may require admin recreate).',
        ],
      },
      {
        title: 'Field guide',
        bullets: [
          'Email — unique login; disabled on edit.',
          'Role — links to Roles page permission sets.',
          'Status — inactive users cannot authenticate.',
        ],
      },
      {
        title: 'Security notes',
        bullets: [
          'Dashboard users are separate from API keys (server/SDK auth).',
          'Do not use dashboard accounts in application code.',
          'Prefer least-privilege roles (e.g. flags:read only for viewers).',
        ],
      },
      {
        title: 'Dokumentasi Wiki',
        bullets: ['Wiki › Panduan — menu Access (Users, Roles).'],
      },
    ],
  },
  roles: {
    id: 'roles',
    eyebrow: 'Access control',
    title: 'Roles — permission bundles',
    summary:
      'Roles group permissions assigned to users. Use presets via permission chips or edit the JSON list directly.',
    sections: [
      {
        title: 'What this page does',
        bullets: [
          'CRUD roles: key, name, description, permissions list.',
          'Permission chips add common scopes quickly.',
          'Users page assigns roles to people.',
        ],
      },
      {
        title: 'How to: create a role',
        bullets: [
          '1. Key — stable id (e.g. flag_editor, viewer); cannot change after create.',
          '2. Name — display label.',
          '3. Description — optional purpose.',
          '4. Permissions — JSON array of strings, or click chips: flags:read, flags:write, flags:evaluate, segments:*, admin:*, *.',
          '5. Save and assign on Users page.',
        ],
      },
      {
        title: 'Permission reference',
        bullets: [
          'flags:read — list/get flags and environments.',
          'flags:write — create/update/delete flags.',
          'flags:evaluate — call evaluate endpoints.',
          'segments:* — manage segments.',
          'admin:* or * — broad admin (use sparingly).',
          'broadcast:subscribe — for API keys on WebSocket (server keys, not dashboard roles).',
        ],
      },
      {
        title: 'Suggested presets',
        bullets: [
          'Viewer — flags:read, segments:* read-only if split later.',
          'Editor — flags:read, flags:write, segments:*.',
          'Operator — add flags:evaluate for test runs.',
          'Admin — admin:* for user/role and system tasks.',
        ],
      },
      {
        title: 'Before deleting a role',
        bullets: [
          'Reassign all users on the Users page.',
          'Deleting a role in use may be blocked or leave users without access.',
        ],
      },
      {
        title: 'Dokumentasi Wiki',
        bullets: ['Wiki › Panduan — menu Access (Users, Roles).'],
      },
    ],
  },
  'wiki-guide': {
    id: 'wiki-guide',
    eyebrow: 'Wiki · Panduan',
    title: 'Panduan dashboard',
    summary:
      'Ringkasan owl: bantuan singkat per halaman. Untuk tutorial lengkap dan penjelasan tiap menu sidebar, buka halaman Wiki › Panduan.',
    sections: [
      {
        title: 'Buka dokumentasi lengkap',
        bullets: [
          'Sidebar Wiki › Panduan — about, tutorial, kartu per menu, pengenalan owl.',
          'Wiki › Konsep Flag Gen — apa/mengapa/menurut 5W1H produk.',
          'Wiki › Developer — SDK, endpoint, env vars.',
        ],
      },
      {
        title: 'Owl di halaman lain',
        bullets: [
          'FAB owl atau tombol ? membuka drawer seperti ini — konten mengikuti halaman aktif (Projects, Flags, …).',
          'Gunakan drawer saat mengisi form; gunakan Wiki sidebar untuk baca mendalam.',
        ],
      },
    ],
  },
  'wiki-flag-gen': {
    id: 'wiki-flag-gen',
    eyebrow: 'Wiki · Konsep',
    title: 'Konsep Flag Gen',
    summary:
      'Ringkasan: feature flag, dua alur definitions vs evaluate. Buka Wiki › Konsep Flag Gen untuk penjelasan Apa/Di mana/Siapa/Kapan/Mengapa/Bagaimana.',
    sections: [
      {
        title: 'Inti',
        bullets: [
          'Definisi flag — sync lewat WebSocket atau GET; satu set per project/environment.',
          'Evaluate — per userId + attributes; wajib untuk rules dan rollout.',
          'Halaman penuh: /dashboard/wiki/flag-gen',
        ],
      },
    ],
  },
  'developer-wiki': {
    id: 'developer-wiki',
    eyebrow: 'Wiki · Developer',
    title: 'Integrasi SDK',
    summary:
      'Ringkasan integrasi. Halaman penuh: Wiki › Developer (/dashboard/wiki/developer) — contoh Go/Angular, env vars, checklist WebSocket.',
    sections: [
      {
        title: 'Project key vs project ID',
        bullets: [
          'Project key — human label (e.g. development); shown on Projects list.',
          'Project ID — UUID; required in /v1/projects/:projectId/... and WebSocket projectId=.',
          'Environment key — used in paths (e.g. production); same as dashboard.',
          'Find IDs on Projects, Environments, or Flags via API & SDK identifiers panel.',
        ],
      },
      {
        title: 'Two integration flows',
        bullets: [
          'Flag definitions — list/sync enabled, value, rules, version (WebSocket snapshot + flag.updated, or GET .../flags).',
          'Evaluate — per request with userId + attributes; returns value, reason, source (POST .../flags/evaluate).',
          'Do not use local cache Variation() alone for targeting — it ignores rules and rollout.',
        ],
      },
      {
        title: 'Frontend (browser / Angular)',
        bullets: [
          'GET /v1/client/projects/{projectId}/environments/{env}/flags — clientSide flags only.',
          'POST .../flags/evaluate with context.attributes JSON.',
          'Header X-Public-Api-Key when PUBLIC_READ_API_KEY is set on the server.',
          'Never embed secrets in flag payloads.',
        ],
      },
      {
        title: 'Backend (Go / services)',
        bullets: [
          'WebSocket GET /v1/broadcast/flags?projectId=&environmentKey=&token= (dev token or API key).',
          'On connect: flags.snapshot; then flag.updated / flag.deleted.',
          'BROADCAST_DEV_TOKEN must match server .env in local dev.',
          'Evaluate with server API key scope flags:evaluate.',
          'Halaman penuh: /dashboard/wiki/developer · docs/guide/sdk/sdk-flagclient_id.md · examples/golang-service.',
        ],
      },
      {
        title: 'Live updates checklist',
        bullets: [
          'Server .env: OUTBOX_WORKER_ENABLED=true, BROADCAST_CONSUMER_ENABLED=true, RabbitMQ up.',
          'SDK: BROADCAST_DEV_TOKEN set; logs should show websocket connected.',
          'After dashboard edit, backend cache updates in seconds without redeploy.',
        ],
      },
    ],
  },
};
