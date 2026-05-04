---
name: evaluator-semantics
description: Implements deterministic flag evaluation rules and metadata per docs/assessment/evaluation-semantics.md. Use when building evaluator functions, rule matching, stable bucketing/rollout, reason/source enums, and not-found/degraded semantics.
---

# Evaluator Semantics (MVP)

## Source of truth

- `docs/assessment/evaluation-semantics.md`

## Deterministic precedence

1. Disabled short-circuit
2. Rule match (priority asc)
3. Rollout gate (if configured)
4. Default

## Rule matching

- Attribute lookup from `context.attributes`
- Operators: `equals`, `not_equals`, `in`, `not_in`
- Strict JSON equality (no coercion)

## Stable bucketing

- Stickiness: `userId` default, or attribute name
- If stickiness value missing → rollout miss
- Hash: sha256 over (scope + flagKey + stable id) then map to 0..100

## Result metadata

Always return:

- `reason`: `disabled|rule_match|rollout_match|rollout_miss|default|not_found|degraded`
- `source`: `cache_l1|cache_l2|database|snapshot|default`
- `version` and optional `matchedRuleId`

## No silent default

Remote reads must not return successful defaults when DB + snapshot cannot serve; return `503 service_degraded` unless endpoint explicitly supports degraded defaults.

