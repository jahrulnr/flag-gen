---
name: domain-modeling
description: Defines domain-layer modeling rules and DDD classification for Aggregate vs Entity vs Value Object in Go. Use when working on domain packages, introducing new domain types, refactoring domain models, or when the user mentions domain, domain layer, domain model, DDD, aggregate, entity, or value object.
---

# Domain Modeling (Aggregate, Entity, Value Object)

## Non-negotiable rules (verbatim)

domain hanyalah bisnis rule. bukan mapper, struct tidak boleh meng-aliaskan sebagai json (struct only). boleh pakai builder pattern jika kena sonarqube complexity

## Domain artifacts (what can exist in domain)

Domain is not only aggregates/entities/value objects. Domain may contain:

- **Aggregates**: consistency boundary + mutation entrypoint
- **Entities**: identity-based objects (often inside an aggregate)
- **Value Objects**: immutable-by-value types with validation
- **Domain Events**: plain structs that represent “something happened” in the business (no IO)
- **Domain Policies / Specifications**: reusable business-rule checks (pure functions/structs)
- **Domain Primitives**: IDs/Keys/Names/Percentages as small validated types (often VOs)
- **Factories (pure)**: constructors that build valid aggregates/Vos when initialization is non-trivial (no IO)

## Quick classification guide

### Use an Aggregate when

- There is a **consistency boundary**: invariants must hold across multiple objects.
- There is a **single point of change**: all mutations go through one root.
- It needs **versioning / optimistic locking** at the boundary.
- External code should not mutate internal parts directly.

**Rule**: Only the aggregate root exposes mutation methods. Everything inside is modified via aggregate methods.

### Use an Entity when

- It has a **stable identity** (ID) that survives changes.
- Two instances with the same identity represent the same conceptual thing.
- It changes over time, but equality is based on identity, not all fields.

**Rule**: Entities may live inside an aggregate; if so, they are not modified directly from outside the aggregate.

### Use a Value Object when

- It is defined by its **value** (structural equality), not identity.
- It is **immutable by default** (create new instance for changes).
- It encapsulates validation/normalization (e.g., `FlagKey`, `ProjectKey`, `Percentage`, `EnvironmentKey`).
- It is safe to copy and compare.

**Rule**: Value objects should validate on construction; never allow an invalid VO instance.

## Domain layer boundaries (what belongs vs what doesn’t)

### Domain layer should contain

- Business rules, invariants, and deterministic computations
- Domain types: aggregates, entities, value objects, domain errors
- Domain events as plain structs (no IO)
- Domain policies/specifications for reusable rule logic
- Interface-only ports under domain packages (see below)

### Domain structs may have behavior

Domain structs are allowed, and often expected, to have pure business methods:

- `value.HasValid()`
- `snapshot.HasValid()`
- `flag.Can(doSomething)`
- `NewSnapshot(a, b, c)` / `NewFlag(...)` constructors or factories

Use methods when they protect invariants, answer domain questions, or express business capabilities. Keep them pure: no database, Redis, HTTP, file, logging, metrics, or JSON/DTO concerns.

### Domain layer must not contain

- JSON mapping tags/DTO concerns (`json:"..."`) or transport schema concerns
- Database models/ORM mapping
- Redis/file/network calls
- Logging/metrics/tracing calls
- Any repository/service implementations (structs). Keep infra-complex logic out of domain.

### Domain ports: `domain/repository` and `domain/service` are interface-only

- `domain/repository`: **interfaces only**, no structs/implementations.
- `domain/service`: **interfaces only**, no structs/implementations.

Rationale: repo/service implementations usually contain complex logic tied to infra (HTTP clients, DB, Redis, files, retries, timeouts). Domain stays pure business rules; infrastructure/application layers provide implementations.

## Domain events

Use domain events when an aggregate mutation needs to notify “the rest of the system” *after commit*.

- Keep events as **plain structs**
- Only business fields (IDs, versions, timestamps), no transport/infra fields
- No publishing logic in domain; application layer handles outbox/publish

## Domain errors

Prefer typed, stable errors so application/presentation can map them consistently.

- Validation/invariant violations should return domain errors
- Avoid embedding HTTP status codes or transport messages in domain

## Policies / Specifications

Use policies/specifications when:

- the same rule is applied in multiple aggregate methods, or
- rule logic grows and would inflate method complexity.

Keep them pure and deterministic (no time, randomness, or IO unless explicitly injected as an input value).

## Invariants and mutation rules

- Enforce invariants at the aggregate boundary. If an operation can violate invariants, it must be impossible to represent it in domain code.
- Prefer methods like `Enable()`, `Disable()`, `UpdateValue(v)`, `AddRule(r)` over exposing fields.
- Prefer domain question/capability methods like `HasValid()`, `CanActivate()`, `CanDelete()`, or `Can(doSomething)` when the answer is a business rule.
- Return typed domain errors for validation failures (`ErrInvalidFlagKey`, etc.).

## Complexity and builder pattern (SonarQube)

If a constructor or mutation method becomes too complex (branchy validation, many optional fields):

- Introduce a **builder** that accumulates inputs and validates once on `Build()`.
- Keep builder as a domain helper (still no JSON, no IO).
- Ensure `Build()` returns a fully valid aggregate/value object or a domain error.

## Examples (Go)

### Value Object: `FlagKey`

```go
type FlagKey struct {
	value string
}

func NewFlagKey(raw string) (FlagKey, error) {
	// validate + normalize
	// return domain error on invalid
	return FlagKey{value: raw}, nil
}

func (k FlagKey) String() string { return k.value }
```

### Aggregate: `Flag`

```go
type Flag struct {
	id      string
	key     FlagKey
	enabled bool
	version int
	rules   []Rule // entity or VO depending on identity needs
}

func (f *Flag) Enable() error {
	f.enabled = true
	f.version++
	return nil
}
```

### Entity vs Value Object decision example: `Rule`

- Make `Rule` an **Entity** if it has its own stable `RuleID` and is referenced/updated independently.
- Make `Rule` a **Value Object** if it’s purely “by value” and replaced wholesale as part of aggregate updates.

## Checklist before adding a new domain type

- Does it have a stable identity? → **Entity**
- Is it immutable and equality-by-value? → **Value Object**
- Does it guard invariants and own mutations? → **Aggregate root**
- Any JSON/DB/IO concerns? → move them out of domain (presentation/infrastructure)

