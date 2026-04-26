# CLAUDE_OPERATIONS.md

> **Version**: 1.3.0
> **Last Updated**: 2026-04-26
> Security requirements, testing standards, observability, Git workflow, CI/CD, deployment, prohibited patterns, and review checklist for `snowgo`.
>
> [Back to CLAUDE.md](./CLAUDE.md)

---

## Table of Contents

1. [Security Requirements](#1-security-requirements)
2. [Testing Standards](#2-testing-standards)
3. [Observability](#3-observability)
4. [Git Workflow & Repository Maintenance](#4-git-workflow--repository-maintenance)
5. [Prohibited Patterns (DO NOT)](#5-prohibited-patterns-do-not)
6. [Code Review Checklist](#6-code-review-checklist)
7. [Infrastructure & Deployment](#7-infrastructure--deployment)

---

## 1. Security Requirements (MANDATORY)

### 1.1 Authentication

- JWT access tokens expire short (`access_expiration_time`).
- Refresh tokens are **single-use** with JTI tracking in Redis (`CacheRefreshJtiPrefix`).
- Login failure rate limiting: 5 failures / 3 minutes per username (`CacheLoginFailPrefix`).

### 1.2 Authorization

- Every admin endpoint must have `middleware.JWTAuth()` + `middleware.PermissionAuth(constant.PermXXXX)` unless explicitly public.
- Permission strings are constants in `internal/constant/permission.go`.
- RBAC resolves permissions via menu tree (`MenuTypeBtn` carries `perms`).

### 1.3 Sensitive Data

- **Never log raw passwords, tokens, secrets, phone numbers, or ID card numbers.**
- Passwords must be hashed with `xcryption.HashPassword()` (bcrypt) before storage.
- API responses must not leak internal error details to clients. Log the detail; return a generic code.
- All PII must be masked in logs and non-admin API responses.

### 1.4 Rate Limiting

- Use `middleware.AccessLimiter` for route-level rate limiting (token bucket).
- Use `middleware.KeyLimiter` for IP/user-level rate limiting.
- Apply rate limits to auth endpoints and expensive APIs.

---

## 2. Testing Standards (MANDATORY)

### 2.1 Test Types

| Type | Scope | Tool |
|------|-------|------|
| **Unit tests** | `pkg/` utilities | `testify/assert` |
| **Service tests** | Business logic, edge cases, error paths | Mock DAO dependencies |
| **DAO tests** | Integration against real test database | `testify/suite` (not mocks) |
| **API tests** | Request validation, response structure | httptest or e2e |

### 2.2 Test Execution Rules

After every code change, execute the minimum sufficient verification:

1. Run tests directly related to the change.
2. Run tests for affected modules.
3. If public infrastructure is affected, expand to broader tests.
4. Prefer the project's standard test command.

Your output must explicitly state:
- New or updated test files / cases.
- Commands executed.
- Which tests passed.
- Which tests were skipped due to environment limits (if any).
- If tests cannot be run, state the reason and provide a manual verification approach.

### 2.3 Test Case Requirements

New tests **must** cover all applicable scenarios:

| Scenario | Description |
|----------|-------------|
| Happy path | Expected behavior under standard input. |
| Parameter boundaries | Max, min, boundary values. |
| Null / default values | nil, empty string, zero value. |
| Invalid input | Format errors, type errors, out-of-range values. |
| Exception branches | Dependency failure, timeout, resource exhaustion. |
| Permission / state checks | Unauthorized, disabled, state mismatch. |
| Idempotency | Repeated calls yield the same result (if applicable). |
| Regression | Bug fix must cover the original trigger condition. |

Test design requirements:
- Test names clearly express intent (e.g., `TestCreateUser_Success`, `TestCreateUser_DuplicateUsername`).
- One test verifies one core behavior (or a tightly related group).
- Avoid brittle tests (do not over-rely on internal implementation details).
- Prefer verifying behavior and results over implementation details.
- Tests must be repeatable.
- Minimize dependency on unstable external environments.

### 2.4 Pseudo-Tests Are NOT Allowed

The following do **not** count as valid tests:
- Print logs without asserting results.
- Test files that do not cover real logic.
- Functions called without behavior verification.
- Manual descriptions without automated tests.
- Meaningless mocks that hide real problems.

**Tests must contain explicit assertions.**

### 2.5 Test Failure Handling

If a test fails, you must:
1. Clearly identify the failing test and its output.
2. Analyze the root cause.
3. Distinguish whether the failure is introduced by this change or is historical.
4. Provide a fix recommendation.
5. **Do not mark the task as complete.**

### 2.6 Environment Limitations

If full testing is impossible due to environment limits (missing DB / Redis / MQ / external services):
1. Explicitly state the reason.
2. Run the minimum unit tests / static checks (`go vet`, `go build`).
3. Provide the full local test command.
4. Provide manual verification steps.
5. Clearly state: "Full verification was not completed due to environment limitations."

### 2.7 Definition of Done

A feature is only complete when:

1. Feature code is implemented.
2. Related tests are added or updated.
3. All related tests pass.
4. No existing functionality is broken.
5. Necessary documentation is updated.
6. A verification method is provided.
7. Risk notes are provided (if applicable).

### Coverage Targets

| Scope | Target |
|-------|--------|
| `pkg/` packages | >= 70% |
| New business modules | Service layer tests mandatory |

- Run: `go test ./... -cover` before committing.
- All tests must pass in CI before merge.

---

## 3. Observability

### 3.1 Tracing

- Optional OpenTelemetry/Tempo integration via `cfg.Application.EnableTrace`.
- `trace_id` is propagated via `X-Trace-Id` header and injected into all context-aware logs.
- All middleware spans include `http.client_ip`, `http.user_agent`, `http.method`, `http.route`.

### 3.2 Metrics

- Prometheus metrics exposed for service monitoring.
- Critical paths (auth, DB queries, cache hits/misses) should have metric instrumentation.
- **Alert on**: P99 latency > 500ms, error rate > 1%, cache hit rate < 80%.

### 3.3 Health Checks

- `/healthz` — liveness probe.
- `/readyz` — readiness probe with MySQL + Redis dependency checks.
- Pprof routes (`/debug/pprof/*`) enabled via config, restricted to private IPs (`127.0.0.1/32`, `192.168.0.0/16`).

---

## 4. Git Workflow & Repository Maintenance (MANDATORY)

### 4.1 Branch Strategy

| Branch | Purpose |
|--------|---------|
| `main` | Production-ready. Protected. All merges via PR. |
| `dev` | Integration branch. Features merge here first. |
| `feature/*` | Individual features. Branch from `dev`, PR back to `dev`. |
| `hotfix/*` | Urgent production fixes. Branch from `main`, PR to both `main` and `dev`. |

### 4.2 Commit Message Convention

Follow conventional commits: `<type>(<scope>): <description>`

| Type | Usage |
|------|-------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `style` | Formatting, no logic change |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `perf` | Performance improvement |
| `test` | Adding or correcting tests |
| `chore` | Build process or auxiliary tool changes |
| `security` | Security fix |

Examples:
- `feat(account): add user role assignment endpoint`
- `fix(auth): resolve JWT refresh token race condition`
- `perf(dal): add index on user.created_at for list query`
- `security(pkg): fix WeakRandInt63n data race`

### 4.3 Pull Request Requirements

- All PRs must pass CI (lint, test, security scan) before merge.
- PR description must reference the issue/ticket and include a summary of changes.
- PRs touching `internal/dal/` must confirm `make gen` was run and no manual edits exist.

---

## 5. Prohibited Patterns (DO NOT)

### Code Quality
- **DO NOT** manually edit `internal/dal/model/` or `internal/dal/query/`.
- **DO NOT** call `panic()` in API/Service/DAO for business errors.
- **DO NOT** compare errors with `err.Error() == "..."`. Use `errors.Is` or sentinel errors.
- **DO NOT** use `fmt.Printf` / `log.Println` in production code.
- **DO NOT** construct SQL with string concatenation. Use GORM Gen type-safe APIs.
- **DO NOT** nest service calls inside transactions. DAO methods only inside `Transaction()`.
- **DO NOT** expose internal error details in API responses.
- **DO NOT** skip permission checks on admin endpoints.
- **DO NOT** store plaintext passwords.
- **DO NOT** forget `is_deleted = false` in list/detail queries **when the table uses soft delete**.
- **DO NOT** use raw `*gorm.DB` in Service/DAO when `*repo.Repository` or `*query.Query` is available.
- **DO NOT** let Service call GORM Gen query APIs directly. All DB access goes through DAO methods.
- **DO NOT** commit secrets, credentials, or `.env` files to version control.
- **DO NOT** disable SSL/TLS in production environments.

### Process & Verification
- **DO NOT** skip tests or claim completion without verification.
- **DO NOT** modify unrelated modules.
- **DO NOT** delete core logic without explicit approval.
- **DO NOT** fabricate test results.
- **DO NOT** lower functional correctness to make tests pass.
- **DO NOT** modify core infrastructure (DB schema, CI/CD, deployment config) without explicit approval.
- **DO NOT** assume external services are available without verification.
- **DO NOT** perform large-scale refactoring unless explicitly requested.

---

## 6. Code Review Checklist (MANDATORY)

Before marking a task complete, verify:

- [ ] New table schema includes `created_at`/`updated_at`; soft delete decided per business need with rationale documented.
- [ ] DAL code is generated, not hand-written.
- [ ] All mutations use `WriteQuery().Transaction()`.
- [ ] Operation logs are written within the transaction.
- [ ] API endpoints have `JWTAuth()` + `PermissionAuth()` (if admin).
- [ ] Input validation happens at API layer.
- [ ] Errors use `xerror` constants; sentinel errors used in Service.
- [ ] Logs use `*Ctx` variants with context.
- [ ] Cache invalidation is implemented for mutations.
- [ ] Sensitive data is masked in logs/responses.
- [ ] Tests pass: `make test`.
- [ ] No `panic` for business logic.
- [ ] Performance targets met (P99 read < 200ms, write < 500ms).
- [ ] Database indexes added for query filters and `ORDER BY` on large/high-growth tables (small config tables may skip); composite indexes follow left-prefix rule.
- [ ] README.md updated if user-facing behavior changed.
- [ ] CI workflows updated if infrastructure changed.
- [ ] CLAUDE.md or sub-document updated if conventions changed.

---

## 7. Infrastructure & Deployment

### 7.1 Docker

- Base images must use pinned versions (e.g., `golang:1.25.0-alpine`, not `alpine:latest`).
- Multi-stage builds mandatory to minimize attack surface.
- Never run containers as root in production.

### 7.2 Environment Configuration

- Configuration must be environment-specific (`config.dev.yaml`, `config.container.yaml`, `config.uat.yaml`, `config.prod.yaml`).
- Secrets must be injected via environment variables or secret management (never hardcoded).
- `.env` files must be in `.gitignore`.

### 7.3 Monitoring & Alerting

All production deployments must have:
- Health check endpoints (`/healthz`, `/readyz`) configured in load balancer.
- Log aggregation (ELK or equivalent).
- Metrics collection (Prometheus + Grafana).
- Alerting on error rate, latency, and availability.
