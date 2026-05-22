# OPERATIONS.md

> Security, testing, review checklist for snowgo.
>
> [Back to AGENTS.md](./AGENTS.md)

---

## 1. Security

- JWT access tokens short-lived. Refresh tokens single-use with JTI tracking in Redis.
- Admin endpoints require `JWTAuth()` after login. Add `PermissionAuth(constant.PermXXXX)` for privileged management operations or scoped business data. Login-only endpoints, such as current user permissions, server info, and allowed dictionary lookups, should be explicit in route comments.
- Never log passwords, tokens, secrets, PII. Passwords via `xcryption.HashPassword()` (bcrypt).
- API responses: no internal error details to clients.
- Production secrets must come from injected environment variables or a secret manager. Do not rely on YAML defaults outside local/demo environments.
- Rotate JWT secrets, database passwords, Redis passwords, RabbitMQ credentials, and deployment SSH/GHCR tokens through an approved release window.
- New endpoints must define their auth level explicitly: public, login-only, or permission-protected. Public and login-only endpoints require a route comment explaining why `PermissionAuth` is not used.

---

## 2. Testing

### 2.1 Test Types

| Type | Scope | Framework | Location |
|------|-------|-----------|----------|
| Unit | pkg/ utilities | testify/assert | `pkg/*_test.go` |
| Service Unit | Business logic, mock DAO/contract/cache | testify/assert + require | `internal/service/{module}/*_test.go` |
| Service Integration | Critical write paths, real DB | testify/suite | `internal/service/{module}/*_integration_test.go` |
| DAO | Integration, real DB | testify/suite | `internal/dao/{module}/*_test.go` |

### 2.2 Service Testing Strategy (Two-Tier)

**Tier 1 — Service Unit Tests (mock dependencies):** Primary test suite for Service business logic.

Mock all external interfaces: DAO interfaces, `contract.OperationLogWriter`, `xcache.Cache`, cross-service mocks. Verify business logic branches, permission checks, parameter validation, error propagation, and cache invalidation.

**Test Context**: `testCtx()` with auth fields (`xauth.XUserId`, `XUserName`, `XTraceId`, `XIp`, `XSessionId`).

**Assertions**: `require` for fatal checks, `assert` for non-fatal, `errors.Is` for sentinel errors.

**Tier 2 — Service Integration Tests (real MySQL):** Critical write-path subset only.

Cover core write operations (create/update/delete) that involve transactions, multi-table mutations, and ORM mappings. Not a full re-run of unit tests — only the paths where database-level behavior matters (unique constraint fallback, transaction rollback, soft delete filters, audit log in same transaction). Read-only endpoints and simple pass-through CRUD do not need integration tests.

Files use `//go:build integration` build tag and are excluded from `go test ./...`. Run with `go test -tags=integration ./internal/service/...` when service integration tests exist for the touched module. Integration tests use environment variables such as `MYSQL_DSN`, `REDIS_ADDR`, `REDIS_DB`, `REDIS_PASSWORD`, and `RABBITMQ_URL`; Service integration tests must connect only to test databases.

**Coverage targets**: pkg/ >= 80%. New business modules require Service unit tests for non-trivial methods and Service integration tests for critical write paths. Existing modules should be backfilled when they are materially changed.

### 2.3 Verification Scope

- Small, localized changes: run the affected package tests, for example `go test ./pkg/xauth/...`.
- Broad/shared behavior changes: run `go test ./...`.
- Integration changes: run integration tests with `make test-integration`, `go test -tags=integration ./internal/service/...`, or a narrower package command when Redis/RabbitMQ/MySQL dependencies are available. Files that depend on external services use the `//go:build integration` build tag and are excluded from `go test ./...`.
- Always run `make lint` before declaring complete.
- Coverage commands are useful for coverage work, but are not the default completion gate for every change.

---

## 3. Release & Configuration

- Use immutable image tags for UAT/prod releases. `latest` is allowed only as an additional production convenience tag, not as the release identifier.
- Production deploys must record image tag, commit SHA, target environment, config version, database migration version, and rollback plan.
- Database changes must be backward compatible within one deployment window: add columns before code reads them, deploy code before removing old columns, and keep rollback scripts for destructive changes.
- Config changes that affect security, persistence, queues, or rate limits require review. Document default values and production overrides in `.env.example` or deployment docs.
- RabbitMQ topology changes should be applied by `cmd/mq-declarer` before deploying code that depends on new exchanges, queues, or bindings.
- Observability changes should include what to check after deployment: health endpoints, key logs, trace availability, queue depth, slow SQL, and error rate.

---

## 4. Commit Convention

Conventional commits: `<type>(<scope>): <desc>`. Types: feat, fix, docs, refactor, perf, test, chore, security.

---

## 5. Prohibited Patterns

- DO NOT manually edit `internal/dal/model/` or `internal/dal/query/`
- DO NOT place implementations, DAOs, business logic, or `init()` in `internal/service/admin/contract`; the contract package contains only interfaces and DTOs
- DO NOT use `fmt.Printf` / `log.Println` in business code; CLI tools, startup banners, package-level fallback loggers, and non-production console access logs are allowed when intentional
- DO NOT call business Service methods through `container.SomeService.Method()` inside a transaction. Transaction-safe infrastructure contracts, such as synchronous operation log writers that receive `*query.Query`, are allowed.
- DO NOT start transactions in DAO; Service owns transaction boundaries and passes `*query.Query`
- DO NOT expose internal error details in API responses
- Add `is_deleted = 0` filter only for tables that implement soft delete
- DO NOT commit secrets or `.env` files. `.env.example` and local/container example configs may include documented demo credentials for first-run testing only; production configs must use injected secrets without defaults.
- DO NOT skip tests or fabricate results

---

## 6. Code Review Checklist

- [ ] created_at mandatory, updated_at only if table has updates
- [ ] Soft delete per business need; PK type (INT vs BIGINT) matches volume
- [ ] DAL generated, not hand-written
- [ ] Transaction boundary is owned by Service; DAO receives caller-provided `*query.Query`
- [ ] Multi-table mutations and audited business mutations use `WriteQuery().Transaction()`
- [ ] Operation log for audited business mutations is written within the same transaction
- [ ] Admin endpoints: JWTAuth; PermissionAuth added for privileged/scoped endpoints, with login-only exceptions documented
- [ ] Input validation at API layer
- [ ] Errors use `e.NewBizError(e.Code)` sentinels in Service; API uses `errors.As` + `FailByError`; no `Fail(c, code, err.Error())` leaking internals
- [ ] Logs use `*Ctx` variants
- [ ] Cache invalidation after DB commit, not inside transaction
- [ ] Sensitive data masked
- [ ] Tests appropriate to the change scope pass
- [ ] `make lint` passes
- [ ] Performance impact reviewed; hot paths have indexes, pagination, and bounded batch sizes
- [ ] Indexes follow left-prefix rule
- [ ] Interface behavior: cache-first reads when applicable, idempotent writes for retryable operations, clear degradation behavior
- [ ] Complex/important code has WHY comments
- [ ] README / AGENTS docs updated
- [ ] Deployment impact documented for config, database, queue, auth, or observability changes
