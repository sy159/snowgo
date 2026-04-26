# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`snowgo` is a production-grade Go backoffice/admin scaffold built on **Gin + GORM Gen**. It targets Go 1.25+ and enforces strict layered architecture, RBAC authorization, JWT authentication, read-write database separation, Redis caching, distributed locking, RabbitMQ messaging, OpenTelemetry tracing, and Prometheus metrics.

**Goal**: All code must be observable, secure, testable, and performant. Do not cut corners for "simplicity."

---

## 1. Commands

```bash
# Local development
go run ./cmd/http               # HTTP API server
go run ./cmd/consumer           # MQ consumer (deploy separately)
go run ./cmd/mq-declarer        # RabbitMQ topology setup (one-shot)

# Testing
make test                       # Full suite with coverage
go test ./pkg/xerror -run TestName -v   # Single package/test

# DAL code generation (MANDATORY workflow — see Section 3)
make gen init                   # Generate models for ALL tables
make gen add                    # Generate models for NEW tables (interactive)
make gen update                 # Regenerate models for EXISTING tables
make gen query                  # Regenerate GORM Gen query APIs

# Docker
make api-build                  # Build API Docker image
make consumer-build             # Build consumer Docker image
make up / make down             # Docker Compose start/stop

# Database
make mysql-init                 # Run DB initialization scripts
make mq-init                    # Declare RabbitMQ exchanges/queues
```

---

## 2. Architecture Principles

### 2.1 Layered Architecture (Strict Boundaries)

```
Router  →  API  →  Service  →  DAO  →  DAL (GORM Gen)  →  MySQL / Redis
```

| Layer | Responsibility | Rules |
|-------|---------------|-------|
| **Router** | HTTP routing, middleware mounting, grouping | No business logic. No direct DB calls. |
| **API** | Request validation, response formatting, calling Service | Input binding (`ShouldBindJSON`/`ShouldBindQuery`), DTO conversion. No transactions. |
| **Service** | Business orchestration, caching, transaction coordination | Contains ALL business rules. Coordinates DAO calls within transactions. |
| **DAO** | Data access abstraction | Wraps GORM Gen queries. Provides both direct and transaction-aware methods. No business logic. |
| **DAL** | Auto-generated model + query code | **Never hand-edit.** Generated via `make gen`. |

### 2.2 Dependency Injection

- All infrastructure (DB, Redis, JWT, Cache, Lock, MQ) lives in `internal/di/container.go`.
- Services are constructed in `NewContainer(...)` using the **Option pattern**.
- Resources registered with `CloseManager` are shut down in **LIFO** order.
- Access container in handlers: `di.GetContainer(c)` or `di.GetSystemContainer(c)`.

---

## 3. DAL Code Generation (Non-Negotiable)

**`internal/dal/model/` and `internal/dal/query/` are machine-generated. Never edit manually.**

Workflow:
1. Design table schema in MySQL.
2. Run `make gen add` (new tables) or `make gen update` (schema changes).
3. Run `make gen query` to regenerate query APIs.
4. `internal/dal/query_model.go` is auto-updated by the generator.

**If you manually edit generated files, the next `make gen` will overwrite your changes and break production.**

---

## 4. Coding Standards

### 4.1 Naming

- **Packages**: lowercase, no underscores (`account`, `system`, `xlogger`).
- **Files**: snake_case (`user_service.go`, `account_router.go`).
- **Interfaces**: describe behavior (`UserRepo`, not `IUser` or `UserInterface`).
- **Structs**: noun-based (`UserService`, `DictParam`).
- **Methods**: verb-based (`CreateUser`, `GetUserList`, `ValidatePassword`).
- **Constants**: CamelCase for exported, camelCase for unexported.
- **DTOs**: API-layer DTOs named `{Entity}Info`, `{Entity}List`, `{Entity}Param`. Service-layer DTOs avoid `json` tags unless crossing boundaries.

### 4.2 Error Handling

**Rules:**
1. **Never panic in API/Service/DAO layers.** Only `xlogger.Panic` for fatal init failures (e.g., DB connection impossible).
2. **API layer** uses `xresponse.FailByError(c, e.SomeError)` for client errors, `xresponse.Fail(c, code, msg)` for edge cases.
3. **Service layer** defines sentinel errors:
   ```go
   var ErrUserNotFound = errors.New(e.UserNotFound.GetErrMsg())
   ```
   Compare with `errors.Is(err, ErrUserNotFound)` — **never compare error strings.**
4. **Error wrapping**: use `errors.WithMessage(err, "contextual description")` when crossing layer boundaries. Log at the origin; wrap when propagating.
5. **DAO layer** returns `errors.WithStack(err)` for raw GORM errors. Never swallow errors.
6. **HTTP status codes**: 400 for validation, 401 for auth, 403 for permission, 404 for missing resource, 429 for rate limit, 500 for unexpected server errors.

### 4.3 Logging

- **Always** use `xlogger.InfofCtx(ctx, ...)` / `xlogger.ErrorfCtx(ctx, ...)` so `trace_id` is automatically injected.
- **Never** use `fmt.Printf` in production code. Console output is allowed only in middleware access logs for dev readability.
- Log levels:
  - `Info`: business events (user created, login success).
  - `Error`: failures requiring investigation. Include `zap.Error(err)` field.
  - `Debug`: verbose diagnostics (disabled in production).
- Access logs auto-mask sensitive fields (`password`, `token`, `secret`, `access_token`, `refresh_token`). If you add new sensitive fields, update `middleware.sensitiveRoots`.

### 4.4 Input Validation

- **API layer is the gate.** All user input must be validated before reaching Service.
- Use Gin binding tags: `binding:"required,max=64"`.
- Add **explicit validation** for business rules (e.g., password complexity, ID > 0, enum values).
- **Never trust frontend.** Re-validate permissions server-side.

### 4.5 Concurrency

- **No goroutines in Service/DAO** unless justified (e.g., background cache warm-up).
- If spawning goroutines, propagate `context.Context` and handle cancellation/timeout.
- Use `xlock.RedisLock` for distributed critical sections. Always `defer unlock()`.

---

## 5. Database & Transaction Rules

### 5.1 Read/Write Separation

- `repo.Query()` — default (respects resolver hint).
- `repo.WriteQuery()` — **mandatory for all mutations** (INSERT, UPDATE, DELETE).
- `repo.ReadQuery()` — for read-only queries that tolerate replication lag.
- **Multi-DB**: use `repo.ChangeDB(dbName)` to switch connections.

### 5.2 Transactions

- **All write operations involving multiple tables must be wrapped in a transaction.**
- Pattern:
  ```go
  err := db.WriteQuery().Transaction(func(tx *query.Query) error {
      // Call DAO transaction methods, passing tx
      // Call operation log within same tx
      return nil
  })
  ```
- **Never** call `container.SomeService.Method()` inside a transaction — services manage transactions, they don't nest.
- Operation logs (`system.OperationLogService.CreateOperationLog`) must be written **within** the same transaction as the business mutation.
- **Service MUST NOT directly use GORM Gen query APIs.** All DB operations MUST go through DAO methods. This is the final enforcement of the DAO layer.

### 5.3 Soft Delete

- All entity tables must have `is_deleted` boolean/tinyint.
- Queries must filter `is_deleted = false` **unless explicitly querying deleted records.**
- Use GORM Gen `UpdateSimple` for soft deletes:
  ```go
  tx.User.Where(tx.User.ID.Eq(id)).UpdateSimple(tx.User.IsDeleted.Value(true))
  ```

### 5.4 Query Optimization

- **Avoid N+1 queries.** Use JOINs or `Preload` when fetching associations.
- **Use GORM Gen scopes** for reusable dynamic filters (e.g., `UserNameScope`, `StatusScope`).
- **Paginate all list endpoints.** Default limit is `constant.DefaultLimit` (10). Enforce `MaxLimit` if applicable.
- Add database indexes for query filters. Document in migration comments.

---

## 6. Security Requirements

### 6.1 Authentication

- JWT access tokens expire short (`access_expiration_time`).
- Refresh tokens are **single-use** with JTI tracking in Redis (`CacheRefreshJtiPrefix`).
- Login failure rate limiting: 5 failures / 3 minutes per username (`CacheLoginFailPrefix`).

### 6.2 Authorization

- Every admin endpoint must have `middleware.JWTAuth()` + `middleware.PermissionAuth(constant.PermXXXX)` unless explicitly public.
- Permission strings are constants in `internal/constant/permission.go`.
- RBAC resolves permissions via menu tree (`MenuTypeBtn` carries `perms`).

### 6.3 Sensitive Data

- **Never log raw passwords, tokens, or secrets.**
- Passwords must be hashed with `xcryption.HashPassword()` (bcrypt) before storage.
- API responses must not leak internal error details to clients. Log the detail; return a generic code.

### 6.4 Rate Limiting

- Use `middleware.AccessLimiter` for route-level rate limiting (token bucket).
- Use `middleware.KeyLimiter` for IP/user-level rate limiting.
- Apply rate limits to auth endpoints and expensive APIs.

---

## 7. Caching Strategy

- Cache at **Service layer**, never in DAO.
- Cache keys: use `constant.CacheXXXPrefix` + entity ID.
- **Cache invalidation**: always invalidate on mutation (update/delete).
- **Cache invalidation MUST happen immediately after successful DB commit.** Do NOT update or invalidate cache inside a transaction — if the transaction rolls back, the cache remains inconsistent.
- Prefer caching read-heavy, infrequently-changing data (user-role mappings, permission trees).
- Set explicit TTLs. Default user-role cache: `CacheUserRoleExpirationDay` days.

---

## 8. Feature Development Workflow

When adding a new module (e.g., `order`, `inventory`):

1. **Database**: Design schema, create migration, ensure `is_deleted`, `created_at`, `updated_at`.
2. **Generate**: `make gen add` → `make gen query`.
3. **DAO**: Implement `internal/dao/{module}/` with direct + transaction methods.
4. **Service**: Implement `internal/service/{module}/` with business logic, caching, operation logging.
5. **API**: Implement `internal/api/{module}/` with binding, validation, DTO conversion.
6. **Routes**: Register in `internal/router/{module}_router.go`. Apply JWT + PermissionAuth.
7. **Permissions**: Add constants to `internal/constant/permission.go`.
8. **DI**: Wire service into `internal/di/container.go`.
9. **Config**: Add config structs to `config/config.go` if new infrastructure is needed.
10. **Tests**: Write unit tests for Service (mock DAO) and DAO (integration with test DB).

---

## 9. Testing Standards

- **Unit tests**: Mandatory for `pkg/` utilities. Use `testify/assert`.
- **Service tests**: Mock DAO dependencies. Test business logic, edge cases, error paths.
- **DAO tests**: Integration tests against a real test database (not mocks). Use `testify/suite` for setup/teardown.
- **Coverage target**: `pkg/` packages ≥ 70%. New business modules must have Service layer tests.
- Run: `go test ./... -cover` before committing.

---

## 10. Observability

### 10.1 Tracing

- Optional OpenTelemetry/Tempo integration via `cfg.Application.EnableTrace`.
- `trace_id` is propagated via `X-Trace-Id` header and injected into all context-aware logs.
- All middleware spans include `http.client_ip`, `http.user_agent`, `http.method`, `http.route`.

### 10.2 Metrics

- Prometheus metrics are exposed for service monitoring.
- Critical paths (auth, DB queries, cache hits/misses) should have metric instrumentation.

### 10.3 Health Checks

- `/healthz` — liveness probe.
- `/readyz` — readiness probe.
- Pprof routes (`/debug/pprof/*`) are enabled via config and restricted to private IPs (`127.0.0.1/32`, `192.168.0.0/16`).

---

## 11. Prohibited Patterns (DO NOT)

- **DO NOT** manually edit `internal/dal/model/` or `internal/dal/query/`.
- **DO NOT** call `panic()` in API/Service/DAO for business errors.
- **DO NOT** compare errors with `err.Error() == "..."`. Use `errors.Is` or sentinel errors.
- **DO NOT** use `fmt.Printf` / `log.Println` in production code.
- **DO NOT** construct SQL with string concatenation. Use GORM Gen type-safe APIs.
- **DO NOT** nest service calls inside transactions. DAO methods only inside `Transaction()`.
- **DO NOT** expose internal error details in API responses.
- **DO NOT** skip permission checks on admin endpoints.
- **DO NOT** store plaintext passwords.
- **DO NOT** forget `is_deleted = false` in list/detail queries.
- **DO NOT** use raw `*gorm.DB` in Service/DAO when `*repo.Repository` or `*query.Query` is available.
- **DO NOT** let Service call GORM Gen query APIs directly. All DB access goes through DAO methods.

---

## 12. Code Review Checklist

Before marking a task complete, verify:

- [ ] New tables follow soft-delete convention (`is_deleted`).
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

---

## 13. AI Execution Workflow (MANDATORY)

When handling any task, Claude MUST follow:

1. **Understand existing code**
   - Locate related Service/DAO implementations.
   - Check if similar logic already exists. Reuse before inventing.

2. **Design before coding**
   - Identify which layer(s) need changes.
   - Do NOT introduce new abstractions unless necessary.

3. **Implement with minimal changes**
   - Reuse existing patterns (naming, error handling, transaction style).
   - Avoid large refactors. Scope changes to the requested task.

4. **Validate correctness**
   - Layer boundaries respected (no DB calls in API, no business logic in DAO).
   - Transactions used correctly (`WriteQuery().Transaction()`).
   - Cache invalidation handled **after** successful DB commit, never inside a transaction.
   - Permission checks applied to all admin endpoints.

5. **Self-review using checklist (Section 12)**
   - Verify every item before declaring the task complete.

---

## 14. Code Generation Requirements

- All generated code must include clear comments explaining:
  - Business logic intent.
  - Transaction boundaries (where `Transaction()` begins and ends).
  - Cache behavior (what is cached, when it is invalidated, TTL if applicable).
- Do not generate placeholder, TODO, or incomplete code.
- Follow existing naming and file structure strictly.
- Prefer modifying existing files over creating new ones unless a new module is required.

---

## 15. Complexity Control

- Do NOT introduce new layers, abstractions, interfaces, generics, or design patterns unless explicitly required by the task.
- Prefer simple, maintainable solutions over generic or extensible designs.
- Avoid over-engineering. If the existing codebase solves the problem without an interface, do not add one.
- When in doubt, match the simplest existing pattern in the same module.
