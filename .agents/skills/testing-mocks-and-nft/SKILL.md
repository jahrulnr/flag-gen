---
name: testing-mocks-and-nft
description: Establishes test/mock and test/nft as centralized fixtures for unit tests. Use when writing Go unit tests, creating repository mocks with gomock, or when the user mentions test/mock, test/nft, fixtures, mocks, gomock, or unit test usecase.
---

# Testing: `test/mock` and `test/nft` as centralized fixtures

## Verbatim rules

di  @test/nft dan  @test/mock  , harus sebagai pusat mock. misal untuk unit test usecase, maka butuh mock repository. ini akan menjadi tempat terpusat untuk mock.

khusus nft, berbeda dari mock, ini biasanya dipakai untuk static output success case.

mock kita buat pakai gomock, jadi interface repository -> gomock -> arahkan ke test/mock/repository

## Directory convention (project)

- `test/mock/`
  - `test/mock/repository/` — generated gomock mocks for repository interfaces
  - (optional) `test/mock/service/` — generated gomock mocks for service interfaces
- `test/nft/`
  - static fixtures for success cases (golden outputs / canonical objects)

## When to use what

### Use `test/mock` (gomock) when

- Unit testing a **usecase** that depends on repositories/services.
- You need to assert calls, arguments, ordering, and error paths.
- You need controllable behavior per test case.

### Use `test/nft` when

- You want **static output success case** fixtures (canonical objects, expected responses).
- You want reuse across tests without expectation logic.

## Gomock flow (required)

1. Define repository/service interfaces in non-domain layer boundaries as agreed (interfaces only).
2. Generate gomock mocks from those interfaces.
3. Store generated mocks under:
   - `test/mock/repository` (for repository interfaces)
4. Use the centralized mock package in tests; do not duplicate mocks in feature folders.

## Test writing checklist

- Prefer table-driven tests.
- For usecase tests:
  - Arrange fixtures from `test/nft` (success shapes)
  - Arrange dependencies using `test/mock/...` (gomock)
  - Assert returned error uses `apperror` (see `app-error-handling` skill) when relevant

