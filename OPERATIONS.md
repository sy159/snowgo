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

---

## 2. Testing

### 2.1 Test Types

| Type | Scope | Framework | Location |
|------|-------|-----------|----------|
| Unit | pkg/ utilities | testify/assert | `pkg/*_test.go` |
| Service | Business logic, mock DAO | testify/assert + require | `internal/service/{module}/*_test.go`（待创建） |
| DAO | Integration, real DB | testify/suite | `internal/dao/{module}/*_test.go`（待创建） |

### 2.2 Service Test Pattern

Use **interface mocking**. Mock DAO by embedding the DAO interface and overriding methods via struct fields. Mock cache with in-memory map implementing `xcache.Cache`. Cross-service mocks: `mockLogWriter` for operation log, `mockRolePerms` for permissions.

**Test Context**: `testCtx()` with auth fields (`xauth.XUserId`, `XUserName`, `XTraceId`, `XIp`, `XSessionId`).

**Assertions**: `require` for fatal checks, `assert` for non-fatal, `errors.Is` for sentinel errors.

Coverage: pkg/ >= 80%. New business modules require Service tests. All code should have test cases where practical — aim for coverage of happy path, boundary conditions, error branches, and permission checks.

### 2.3 Verification Scope

- Small, localized changes: run the affected package tests, for example `go test ./pkg/xauth/...`.
- Broad/shared behavior changes: run `go test ./...`.
- Integration changes: run integration tests with `make test-integration` or `go test -tags=integration ./pkg/...` when Redis/RabbitMQ/MySQL dependencies are available. Files that depend on external services use the `//go:build integration` build tag and are excluded from `go test ./...`.
- Always run `make lint` before declaring complete.
- Coverage commands are useful for coverage work, but are not the default completion gate for every change.

---

## 3. Commit Convention

Conventional commits: `<type>(<scope>): <desc>`. Types: feat, fix, docs, refactor, perf, test, chore, security.

---

## 4. Prohibited Patterns

- DO NOT manually edit `internal/dal/model/` or `internal/dal/query/`
- DO NOT use `fmt.Printf` / `log.Println` in business code; CLI tools, startup banners, package-level fallback loggers, and non-production console access logs are allowed when intentional
- DO NOT call `container.SomeService.Method()` inside a transaction
- DO NOT start transactions in DAO; Service owns transaction boundaries and passes `*query.Query`
- DO NOT expose internal error details in API responses
- Add `is_deleted = 0` filter only for tables that implement soft delete
- DO NOT commit secrets or `.env` files. `.env.example` and local/container example configs may include documented demo credentials for first-run testing only; production configs must use injected secrets without defaults.
- DO NOT skip tests or fabricate results

---

## 5. Code Review Checklist

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
- [ ] Performance targets met (P99 read < 200ms, write < 500ms)
- [ ] Indexes follow left-prefix rule
- [ ] Interface: cache-first reads, idempotent writes, graceful degradation
- [ ] Complex/important code has WHY comments
- [ ] README / AGENTS docs updated
