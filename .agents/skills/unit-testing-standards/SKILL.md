---
name: unit-testing-standards
description: Enforces unit test standards for this repo: minimum 80% coverage for all changes, centralized mocks/NFT fixtures, and linting pragmas for test-related files. Use when adding or changing Go code, writing unit tests, generating mocks/fixtures, or discussing coverage, golangci-lint, gosec, or CI quality gates.
---

# Unit Testing Standards

## Verbatim rules

unit test. semua changes wajib >= 80%. untuk semua file *_mock.go, *_nft.go, *_test.go, wajib punya header // nolint untuk menurunkan effort fix linter. fix linter dan gosec hanya wajib di real code.

## Coverage policy

- Any code change must keep **coverage >= 80%** for the changed scope/workstream.
- Prefer unit tests at usecase/application layer with mocks for dependencies.

## Linting policy for tests/mocks/fixtures

- For files matching:
  - `*_mock.go`
  - `*_nft.go`
  - `*_test.go`
- Put a file header:
  - `// nolint`

This reduces effort in fixing linter warnings in non-production code.

## Security/lint rules: “real code” vs non-real code

- **Real code** (delivery/application/domain/infrastructure runtime code): linter + gosec findings must be fixed.
- **Non-real code** (tests, mocks, nft fixtures): linter/gosec fixes are not required; rely on `// nolint` header convention.

## Practical workflow (recommended)

- When implementing a feature:
  - Add tests first (or alongside) to keep coverage target.
  - Use centralized mocks and fixtures (see `testing-mocks-and-nft` skill).
  - Keep test assertions deterministic and table-driven.

## Test file pairing convention (required)

Unit tests must be colocated and paired with the file under test:

- `foo.go` → `foo_test.go`
- `flag_repository.go` → `flag_repository_test.go`

Avoid “mega test files” that test many unrelated files at once, unless the tests are intentionally integration-style.

