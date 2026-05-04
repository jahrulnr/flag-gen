---
name: project-layout-clean-architecture
description: Defines the Go project layout and dependency direction for clean architecture in this repo. Use when adding new packages, moving code out of cmd/app/main.go, introducing delivery/application/domain/infrastructure layers, or when the user mentions clean architecture, folder structure, package boundaries, or layering.
---

# Project Layout (Clean Architecture, Go)

## Goal

Keep codebase growth consistent and avoid “everything in main.go”.

## Target package layout (recommended)

```
cmd/app/                      # composition root (wiring)
delivery/http/                # Gin handlers, middleware, routing, DTOs
application/                  # usecases (commands/queries), orchestration
domain/                       # business rules only (no IO, no JSON tags)
infrastructure/               # DB/Redis/file/http-client implementations
infrastructure/di/            # Google Wire provider sets + injectors (wire_gen.go)
pkg/                          # shared libs (e.g., apperror)
test/mock/                    # gomock outputs (generated)
test/nft/                     # static success fixtures
```

## Dependency direction (must)

- `domain` depends on nothing outside `domain`.
- `application` depends on `domain` + domain ports (interfaces), not infra.
- `delivery` depends on `application` (calls usecases) and `pkg` (error/log helpers).
- `infrastructure` depends on external libs (db/redis/http), implements ports used by `application`.
- `cmd/app` wires everything together.

## Domain ports (interfaces)

- Put **interfaces only** under:
  - `domain/repository`
  - `domain/service`

No structs/implementations in domain ports.

## DTO vs Domain separation

- Delivery DTOs may have JSON tags.
- Domain structs must not be aliased as JSON (struct-only).
- Mapping happens in delivery/application boundary.

## Composition/wiring rules

- `cmd/app/main.go` should only:
  - init config/logging
  - build dependencies via Wire injector (preferred)
  - register routes
  - start server + graceful shutdown

## Wire (dependency injection)

- Use Google Wire for compile-time DI.
- Put provider sets and injectors in `infrastructure/di/*`.
- Generate `infrastructure/di/wire_gen.go` via `make wire`.

## Testing alignment

- Usecases are unit-tested with `test/mock/...` and fixtures in `test/nft/...`.
- Coverage standard is handled by `unit-testing-standards` skill.

