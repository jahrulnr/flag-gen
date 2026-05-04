---
name: wire-dependency-injection
description: Uses Google Wire for compile-time dependency injection in this repo. Use when wiring new components, creating provider sets/injectors under infrastructure/di, generating wire_gen.go, or when the user mentions wire, DI, provider set, or injector.
---

# Google Wire (Dependency Injection)

## Repo convention

- Wire lives in: `infrastructure/di/*`
- Generated file: `infrastructure/di/wire_gen.go`
- Generation command: `make wire`

## What goes into `infrastructure/di`

- Provider functions (constructors) for:
  - repositories (infra)
  - caches
  - clients (DB/Redis/HTTP)
  - usecases (application)
  - HTTP handlers/router setup (delivery)
- Wire sets (`wire.NewSet(...)`) grouped by concern (db, cache, http, app)
- One or more injectors that return fully wired root objects used by `cmd/app/main.go`

## What does NOT go into Wire

- Business logic (domain)
- Request handlers themselves (logic) — only constructors/providers

## Practical workflow

1. Add a constructor (provider) for the new component.
2. Add it to the appropriate Wire set.
3. Run `make wire`.
4. `cmd/app/main.go` should call the injector and then start the server.

