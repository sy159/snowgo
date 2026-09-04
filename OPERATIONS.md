# OPERATIONS.md

> Security, testing, review checklist for snowgo.
>
> [Back to AGENTS.md](./AGENTS.md)

---

## 1. Security

- JWT access tokens short-lived. Refresh tokens single-use with JTI tracking in Redis.
- Admin endpoints require `JWTAuth()` after login. Add `PermissionAuth(constant.PermXXXX)` for privileged management operations or scoped business data. Login-only endpoints, such as current user permissions, server info, and allowed dictionary lookups, should be explicit in route comments.
- Never log passwords, tokens, secrets, or PII, and do not rely solely on access-log masking. Passwords use `xcryption.HashPassword()` (bcrypt). New logged fields require a sensitivity review and masking tests where applicable.
- API responses: no internal error details to clients.
- Production secrets must come from injected environment variables or a secret manager. Do not rely on YAML defaults outside local/demo environments.
- Rotate JWT secrets, database passwords, Redis passwords, RabbitMQ credentials, and deployment SSH/GHCR tokens through an approved release window.
- New endpoints must define their auth level explicitly: public, login-only, or permission-protected. Non-obvious public and login-only endpoints require a route comment or configuration rationale explaining why `PermissionAuth` is not used; self-evident cases such as login and health checks do not need boilerplate comments.

---

## 2. Testing

### 2.1 Test Types

| Type | Scope | Framework | Location |
|------|-------|-----------|----------|
| Unit | pkg/ utilities | `testing`; testify when useful | `pkg/*_test.go` |
| Service Unit | Business logic, fake or mock DAO/contract/cache | `testing`; testify when useful | `internal/service/{domain}/{module}/*_test.go` |
| Service Integration | Critical write paths, real DB | `testing`; testify when useful | `internal/service/{domain}/{module}/*_integration_test.go` |
| DAO | Complex queries and mappings with a real DB, when needed | `testing`; testify when useful | `internal/dao/{domain}/{module}/*_test.go` |

### 2.2 Service Testing Strategy (Two-Tier)

**Tier 1 — Service Unit Tests (mock dependencies):** Primary test suite for Service business logic.

Mock external Service dependencies such as DAO interfaces, `contract.OperationLogWriter`, and `xcache.Cache`. Verify business branches, state-dependent authorization, parameter validation, error propagation, and cache invalidation. Test router middleware classification separately from Service authorization rules.

**Test Context**: `testCtx()` with auth fields (`xauth.XUserId`, `XUserName`, `XTraceId`, `XIp`, `XSessionId`).

**Assertions**: `require` for fatal checks, `assert` for non-fatal, `errors.Is` for sentinel errors.

**Tier 2 — Service Integration Tests (real MySQL):** Critical write-path subset only.

Cover core write operations (create/update/delete) that involve transactions, multi-table mutations, and ORM mappings. Not a full re-run of unit tests — only the paths where database-level behavior matters (unique constraint fallback, transaction rollback, soft delete filters, audit log in same transaction). Read-only endpoints and simple pass-through CRUD do not need integration tests.

Files use `//go:build integration` build tag and are excluded from `go test ./...`. Run with `go test -tags=integration ./internal/service/...` when service integration tests exist for the touched module. Integration tests use environment variables such as `MYSQL_DSN`, `REDIS_ADDR`, `REDIS_DB`, `REDIS_PASSWORD`, and `RABBITMQ_URL`; Service integration tests must connect only to test databases.

**Coverage policy**: `pkg/ >= 80%` is currently an advisory target because CI does not enforce a threshold. Do not claim the target is met without measuring it. New business modules require Service unit tests for non-trivial methods and Service integration tests for critical write paths. Existing modules should be backfilled when they are materially changed.

### 2.3 Verification Scope

- Small, localized changes: run the affected package tests, for example `go test ./pkg/xauth/...`.
- Broad/shared behavior changes: run `go test ./...`.
- Integration changes: run integration tests with `make test-integration`, `go test -tags=integration ./internal/service/...`, or a narrower package command when Redis/RabbitMQ/MySQL dependencies are available. Files that depend on external services use the `//go:build integration` build tag and are excluded from `go test ./...`.
- For Go code changes, format only changed files with `gofmt`, run `make lint`, and run scoped tests. CI pins `golangci-lint v2.12.2`; if the local executable is unavailable, report lint as skipped and do not claim it passed. For documentation-only or non-Go changes, run relevant link, format, and diff checks.
- For security-sensitive Go changes, run the relevant tests and local security scans when available. CodeQL and gosec in CI are the canonical security gates; report any scan not actually run.
- Coverage commands are useful for coverage work, but are not the default completion gate for every change.
- Do not change production behavior solely to satisfy a test when it conflicts with the intended requirement. Surface test/specification conflicts instead of optimizing blindly for green tests.

---

## 3. Release & Configuration

- Use immutable image tags for UAT/prod releases. `latest` is allowed only as an additional production convenience tag, not as the release identifier.
- Production deploys must record image tag, commit SHA, target environment, config version, schema-change artifacts, and rollback plan.
- Database changes must be backward compatible within one deployment window: add columns before code reads them and deploy compatible code before removing old columns. The repository currently has no versioned migration runner; keep `docs/sql/init.sql` as the installation baseline. For each deployable schema change, create a date directory such as `docs/sql/20260903/` and place one or more ordered, descriptive SQL files inside, for example `001_add_order_table.sql`. Document execution order and rollback instructions in the SQL or change hand-off, and do not claim a migration version that does not exist. Introducing migration tooling is separate approved work.
- Changes to routes, request or response fields, application error codes, and authentication behavior require an explicit compatibility assessment and versioning or rollout plan when breaking compatibility cannot be avoided.
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
- [ ] Admin endpoints: JWTAuth; PermissionAuth added for privileged/scoped endpoints, with non-obvious public/login-only exceptions documented
- [ ] API transport validation and Service business validation are both present where applicable
- [ ] Errors use `e.NewBizError(e.Code)` sentinels in Service; API uses `errors.As` + `FailByError`; no `Fail(c, code, err.Error())` leaking internals
- [ ] Logs use `*Ctx` variants
- [ ] Cache invalidation after DB commit, not inside transaction
- [ ] Logs contain no credentials, tokens, secrets, or PII; configured access-log masking has tests where applicable
- [ ] Tests appropriate to the change scope pass
- [ ] Changed Go files are formatted; scoped tests pass; `make lint` passes or an unavailable local tool is reported without claiming success
- [ ] New dependencies are justified and reviewed for maintenance, license, security, and operational impact
- [ ] Performance impact reviewed; hot paths have indexes, pagination, and bounded batch sizes
- [ ] Indexes follow left-prefix rule
- [ ] Interface behavior: cache-first reads when applicable, idempotent writes for retryable operations, clear degradation behavior
- [ ] Complex or core code has concise Chinese WHY comments; simple code is not over-commented
- [ ] README / AGENTS docs updated when code behavior, configuration, APIs, or operational workflows change
- [ ] Compatibility and deployment impact documented for API, config, database, queue, auth, or observability changes
