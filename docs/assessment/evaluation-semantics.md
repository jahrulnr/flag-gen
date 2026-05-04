# Evaluation Semantics (Deterministic MVP Spec)

## Purpose

This document defines **deterministic** evaluation rules for the MVP so implementations across:

- in-service evaluation (HTTP evaluate endpoints),
- server-side SDK evaluation (local read state),
- snapshot fallback evaluation,

produce consistent outcomes and metadata (`reason`, `source`, versions).

## Inputs

### EvaluationContext

- `userId`: optional string
- `attributes`: optional map of string → JSON value

### Flag Model (MVP)

- `enabled`: boolean
- `value.default`: JSON value (boolean or JSON)
- `rules[]`: ordered rules with `priority` (lower first)
- `rollout.percentage`: 0–100 (optional)
- `rollout.stickiness`: string (default `"userId"`)
- `version`: aggregate version integer

### Rule Model (MVP)

Rule fields:

- `id`
- `attribute`
- `operator`: `equals|not_equals|in|not_in`
- `value`: JSON scalar/array/object as required by operator
- `serve`: JSON value returned when the rule matches
- `priority`: integer (lower is evaluated first)

## Deterministic Evaluation Algorithm

Given `(flag, context)`:

1. **Disabled short-circuit**
   - If `flag.enabled == false`, return:
     - `value = flag.value.default`
     - `reason = "disabled"`
     - `matchedRuleId = null`
2. **Rule evaluation (priority order)**
   - Sort rules by `priority` ascending (stable by rule id if ties).
   - For each rule in order:
     - Compute `match = ruleMatches(rule, context)`.
     - If `match == true`, return:
       - `value = rule.serve`
       - `reason = "rule_match"`
       - `matchedRuleId = rule.id`
3. **Rollout evaluation (if configured)**
   - If `rollout.percentage` is present:
     - Compute `bucket = stableBucket(flag.key, context, rollout.stickiness)`
     - If `bucket < rollout.percentage`, return:
       - `value = true` if the flag value type is boolean and `default` is boolean
       - otherwise return `flag.value.default` (MVP: no per-variant rollout; rollout only gates “enabled default”)
       - `reason = "rollout_match"`
     - Else:
       - return default with `reason = "rollout_miss"`
4. **Default**
   - Return:
     - `value = flag.value.default`
     - `reason = "default"`

Notes:

- MVP rollout semantics above intentionally avoid multi-variant experiments. If the flag type is JSON, rollout controls whether the flag returns default vs default (no-op) unless extended later; implementations may choose to treat rollout as a gate for `enabled` only in MVP.

## Rule Matching Semantics

### Attribute Lookup

- A rule reads `context.attributes[rule.attribute]`.
- If the attribute is missing:
  - the rule does **not** match.

### Operators

- `equals`: attribute must be JSON-equal to `rule.value`
- `not_equals`: attribute must be present and JSON-not-equal to `rule.value`
- `in`: attribute must be present and found inside `rule.value` (which must be an array)
- `not_in`: attribute must be present and not found inside `rule.value` (array)

Equality is strict JSON equality (type-aware), not string coercion.

## Stable Bucketing (Rollout)

### Bucket Key

Compute a stable identifier string:

- If `rollout.stickiness == "userId"`:
  - If `context.userId` is empty/missing, rollout cannot be computed → treat as **rollout miss**.
  - `id = context.userId`
- Else if `rollout.stickiness` matches an attribute name:
  - If that attribute is missing or not a string/number, treat as **rollout miss**.
  - `id = string(attributeValue)`
- Else:
  - treat as **rollout miss** (MVP strictness)

### Hashing

Compute:

- `h = sha256(projectScopeSalt + ":" + environmentKey + ":" + flag.key + ":" + id)`

Where `projectScopeSalt` is a stable string per project (MVP: can be `projectId`).

Convert the first 8 bytes of `h` to an unsigned integer \(0..2^64-1\), then map to \(0..100\):

- `bucket = (uint64 / 2^64) * 100`

Use a consistent implementation across service/SDKs.

## Evaluation Result Metadata

Every evaluation result includes:

- `value`: JSON value
- `version`: flag aggregate version (from the evaluated flag payload)
- `reason`: one of:
  - `disabled`
  - `rule_match`
  - `rollout_match`
  - `rollout_miss`
  - `default`
  - `not_found`
  - `degraded` (only when endpoint explicitly supports degraded defaults)
- `matchedRuleId`: nullable
- `source`: one of:
  - `cache_l1`
  - `cache_l2`
  - `database`
  - `snapshot`
  - `default`

## Not Found Semantics

For HTTP single-flag reads:

- If DB confirms missing flag key, return `404 flag_not_found`.
- Cache an **L1-only** not-found marker for 5 minutes (per existing docs).

For bulk evaluate:

- Either return `404` if any requested flag is missing, or return a partial result map with missing flags marked as `not_found`.
- MVP recommendation: **partial map** with per-flag `reason="not_found"` while HTTP status stays `200` (more ergonomic for clients).

## Degraded Reads and “No Silent Default”

Remote HTTP reads must **not** silently return successful default values when:

- L1 miss + L2 miss/unavailable + DB unavailable + snapshot missing/invalid

In that case:

- Return `503 service_degraded` (as per API contract/sequence diagram)

Only endpoints explicitly documented as “degraded defaults” may return `200` with `reason="degraded"` and `source="default"`.

## Precedence Summary (Highest to Lowest)

1. Disabled short-circuit
2. Rule match (lowest priority number first)
3. Rollout gate (if configured)
4. Default

