# CLAUDE_OPERATIONS.md

> Security, testing, review checklist for snowgo.
>
> [Back to CLAUDE.md](./CLAUDE.md)

---

## 1. Security

- JWT access tokens short-lived. Refresh tokens single-use with JTI tracking in Redis.
- Every admin endpoint: `JWTAuth()` + `PermissionAuth(constant.PermXXXX)`.
- Never log passwords, tokens, secrets, PII. Passwords via `xcryption.HashPassword()` (bcrypt).
- API responses: no internal error details to clients.

---

## 2. Testing

### 2.1 Test Types

| Type | Scope | Framework | Location |
|------|-------|-----------|----------|
| Unit | pkg/ utilities | testify/assert | `pkg/*_test.go` |
| Service | Business logic, mock DAO | testify/assert + require | `internal/service/{module}/*_test.go` |
| DAO | Integration, real DB | testify/suite | `internal/dao/{module}/*_test.go` |

### 2.2 Service Test Pattern

Use **interface mocking**. Reference: `internal/service/account/user_test.go`.

- **Mock DAO**: Embed DAO interface, override methods via struct fields.
- **Mock Cache**: In-memory `map[string]string` implementing `xcache.Cache`.
- **Mock cross-service**: `mockLogWriter` for operation log, `mockRolePerms` for permissions.
- **Transaction methods**: Real database transactions (rollback, isolation) should be covered by DAO integration tests. Business logic within transaction methods can still be tested via mocks.

**Test Context**: `testCtx()` with auth fields (`xauth.XUserId`, `XUserName`, `XTraceId`, `XIp`, `XSessionId`).

**Assertions**: `require` for fatal checks, `assert` for non-fatal, `errors.Is` for sentinel errors.

Coverage: pkg/ >= 80%. New business modules require Service tests. All code should have test cases where practical — aim for coverage of happy path, boundary conditions, error branches, and permission checks.

---

## 3. Commit Convention

Conventional commits: `<type>(<scope>): <desc>`. Types: feat, fix, docs, refactor, perf, test, chore, security.

---

## 4. Prohibited Patterns

- DO NOT manually edit `internal/dal/model/` or `internal/dal/query/`
- DO NOT use `fmt.Printf` / `log.Println` in production code
- DO NOT call `container.SomeService.Method()` inside a transaction
- DO NOT expose internal error details in API responses
- Always add `is_deleted = 0` filter for soft-delete tables
- DO NOT commit secrets or .env files
- DO NOT skip tests or fabricate results

---

## 5. Code Review Checklist

- [ ] created_at mandatory, updated_at only if table has updates
- [ ] Soft delete per business need; PK type (INT vs BIGINT) matches volume
- [ ] DAL generated, not hand-written
- [ ] Multi-table mutations use `WriteQuery().Transaction()`
- [ ] Operation log within transaction (sync, consistency guaranteed)
- [ ] Admin endpoints: JWTAuth + PermissionAuth
- [ ] Input validation at API layer
- [ ] Errors use xerror constants; sentinel errors in Service; `errors.Is` comparison
- [ ] Logs use `*Ctx` variants
- [ ] Cache invalidation after DB commit, not inside transaction
- [ ] Sensitive data masked
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] Performance targets met (P99 read < 200ms, write < 500ms)
- [ ] Indexes follow left-prefix rule
- [ ] Interface: cache-first reads, idempotent writes, graceful degradation
- [ ] Complex/important code has WHY comments
- [ ] README / CLAUDE docs updated
