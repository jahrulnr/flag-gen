---
name: feature-flag-wiki-best-practices
description: Applies feature flag best practices from docs/guide/concepts/feature-flags*.md. Use when the user asks what a feature flag is, when to use feature flags vs configuration, naming/ownership/cleanup guidance, rollout/rollback strategy, or when writing/triaging docs about feature flags.
---

# Feature Flag Wiki Best Practices

## Source of truth

- `docs/guide/concepts/feature-flags.md`
- `docs/guide/concepts/feature-flags_id.md`

## Core guidance to apply

### When to use feature flags

Use feature flags for:

- Gradual rollout / canary / beta testing
- Emergency rollback (kill switch)
- Segment-based enablement (tenant/user segment)
- Coordinating changes across services

### When NOT to use feature flags

- Do not use feature flags to store **credential/secret/sensitive config**.
- Anything client-side should be assumed readable by users; treat payloads as public.

### Feature flag vs configuration

Use feature flags when:

- lifecycle is temporary (should be removed after rollout stabilizes)
- changes during rollout are frequent

Use config when:

- it’s permanent operational settings (timeouts, URLs, DB configs)

## Decision checklist (quick)

Before recommending a flag, answer:

- Is this a **new feature** that needs staged enablement or fast rollback? → flag
- Is this a **permanent setting** (timeout, endpoint URL)? → config
- Will the value ever reach client-side? If yes, ensure it’s not sensitive.
- Is there a plan to **clean up/remove** the flag after stable?

## Rollout / rollback practices

- Prefer starting small (e.g., 1–10%) then increase.
- Monitor usage/errors during rollout.
- Be able to disable immediately if issues detected.
- Once stable and fully rolled out, remove the flag to reduce complexity.

## Naming & docs hygiene

- Use clear, consistent flag names (`enable_*`, `*_v2`, etc.).
- Document purpose, owner, and removal plan.
- Test both states (enabled/disabled) during development.

