# CODING.md

> Coding standards for snowgo.
>
> [Back to AGENTS.md](./AGENTS.md)

---

## 1. Naming

| Category | Rule | Example |
|----------|------|---------|
| Packages | lowercase, no underscores | account, system, xlogger |
| Files | snake_case | user_service.go |
| Interfaces | describe behavior | UserRepo |
| Structs | noun-based | UserService, DictParam |
| Methods | verb-based | CreateUser, GetUserList |
| Constants | CamelCase exported, camelCase unexported | CacheMenuTree, defaultLimit |
| DTOs (API) | {Entity}Info, {Entity}List, {Entity}Param | UserInfo, UserList, UserParam |
| DTOs (Service) | no json tags unless crossing boundaries | UserCondition |
| API request tags | Add `json` or `form` for the accepted transport and `binding` for transport validation; do not add unused tags | `json:"username" binding:"required,max=64"` |

---

## 2. Constants

Shared business constants live in `internal/constant/`. Do not inline values that are reused, persisted, exposed through APIs, used for permissions/cache/MQ, or compared across packages. Local one-off strings in tests, logs, and small private helpers may stay inline when extracting them would reduce readability.

| File | Scope |
|------|-------|
| `constant.go` | General (status values, default limits) |
| `cache_key.go` | Redis cache key prefixes |
| `permission.go` | RBAC permission strings |
| `mq.go` | RabbitMQ exchange/queue/routing key names |

Error codes: `pkg/xerror/` (5-digit scheme, separate registry).

Configuration keys and environment variable names should be documented in `.env.example` and README when they affect deployment.

---

## 3. Error Handling

| Layer | Rule |
|-------|------|
| API | Use `errors.As` to extract `e.BizError`, respond with `FailByError(c, bizErr.Code)`. Use `Fail` only for transport/request validation or protocol-specific messages. Never `Fail(c, code, err.Error())` for Service errors |
| Service | Define sentinels with `e.NewBizError(e.Code)`. Never `errors.New` for business errors. Use `fmt.Errorf("%w", err)` for infrastructure errors |
| DAO | Return directly — raw GORM for DB. DAO validation should be minimal; business validation belongs in Service |
| Global | Never `panic()` in API/Service/DAO. Only `xlogger.Panic` for fatal init |

### BizError Pattern

Service layer uses `xerror.BizError` carrying `xerror.Code`, API layer extracts uniformly:

```go
// Service: define sentinel
var ErrUserNotFound = e.NewBizError(e.UserNotFound)

// Service: use WrapBizError to preserve underlying cause
if err := validatePassword(pw); err != nil {
    return e.WrapBizError(e.PwdError, err)
}

// API: extract and respond
var bizErr *e.BizError
if errors.As(err, &bizErr) {
    xresponse.FailByError(c, bizErr.Code)
    return
}
xlogger.ErrorfCtx(ctx, "...: %v", err)
xresponse.FailByError(c, e.HttpInternalServerError)
```

To add a new business error:
1. Add Code in `pkg/xerror/error.go`
2. Add BizError sentinel in Service

(API handler unchanged, `errors.As` handles it automatically)

Prefer returning an existing sentinel with `errors.Is` compatibility when callers need branching. Use `WrapBizError` when the underlying cause is useful for logs but should not be exposed to clients.

Never silently ignore an error or turn an unexpected error into a successful result. An explicitly best-effort operation must document why failure is acceptable and log or emit a metric for the failure. Do not log an error and continue when doing so would violate correctness or consistency.

### Error Code Scheme

5-digit integers via `xerror.NewCode(category, code, msg)`. Duplicate codes panic at init.

| Range | Meaning |
|-------|---------|
| 0 and selected 2xx-5xx values | Success and protocol-shaped application codes |
| 1xxxx | Business errors |
| 2xxxx | System/infra errors |

Structure: `[level][module][specific]` — first digit = level, digits 2-3 = module, digits 4-5 = specific.

### Response Code Semantics

`xresponse.Json` currently returns HTTP 200 for JSON envelopes and places the application code in the response `code` field. Values such as 400, 401, 403, 404, and 500 are therefore protocol-shaped application codes, not transport status codes. Rate limiting currently uses system codes 20101 and 20102. Do not change this contract or assume HTTP status equals application code without an explicit compatibility decision and API tests.

---

## 4. Logging

- Business code uses the context-aware logger APIs, all of which inject `trace_id`: prefer `xlogger.InfoCtx` / `xlogger.ErrorCtx` for structured fields and use `xlogger.InfofCtx` / `xlogger.ErrorfCtx` when formatted logging is clearer.
- Avoid `fmt.Printf` / `log.Println` in business code. CLI tools, startup banners, package-level fallback loggers, and non-production console access logs may use standard output when intentional.
- `Warn`: reserved for access logs via `xlogger.Access()`.
- `Info`: business events. `Error`: anomalies. `Debug`: disabled in production.
- Never rely on automatic masking as the only control. Business logs must not include credentials, tokens, secrets, or PII. The access logger masks only its configured JSON paths; when a new sensitive field can enter request or response logs, update the masking configuration and tests.

---

## 5. Context Propagation

- All functions accepting `context.Context` must propagate to downstream calls.
- Extract auth data via `xauth` constants: `XUserId`, `XUserName`, `XIp`, `XSessionId`, `XTraceId`.
- Use `context.WithTimeout` / `WithCancel` only when adding a deadline scope.

---

## 6. Input Validation

- API validates transport-level input before reaching Service: binding, required fields, format, length, enums, and basic request limits.
- Service validates business rules and state-dependent preconditions, such as resource existence, allowed status transitions, inventory, and operation eligibility.
- Gin binding tags: `binding:"required,max=64"`. Add explicit validation for request-level constraints.
- Common tags: `required`, `max=N`, `min=N`, `email`, `oneof=A B`.

---

## 7. Concurrency

- No goroutines in Service/DAO unless justified.
- Propagate `context.Context` and handle cancellation/timeout.
- Distributed lock: `xlock.RedisLock` (`pkg/xlock/`). Callback-based — `TryLock()`, `ReTryLock()`, `LockWithTries()`, `LockWithTriesTime()`. LockContext provides `Unlock()` / `Extend()` methods; lock is auto-released after callback returns.

---

## 8. Service Dependencies

Use package boundaries to decide whether a Service dependency needs an interface.

Before introducing a dependency, error, logging, transaction, cache, response, authorization, or configuration pattern, search the repository for the established implementation and reuse it unless it is insufficient for the requirement.

| Scenario | Rule | Example |
|----------|------|---------|
| Same package | Inject the concrete `*XxxService`. Do not add one-method interfaces only for indirection. | `system.DictService` may depend on `*OperationLogService`. |
| Cross package, shared capability | Depend on a stable contract in `internal/service/admin/contract`. The contract package contains only interfaces and DTOs; no implementations, DAO, cache, logging side effects, or business orchestration. | `account` services depend on `contract.OperationLogWriter`; `system.OperationLogService` implements it. |
| Cross package, one-off domain dependency | Prefer a consumer-local interface when the dependency is specific to one caller or likely to create an import cycle. | A future workflow package can define the exact user lookup methods it needs locally. |
| Cross domain/module | Isolate behind `contract` or a consumer-local interface. Treat this as a future service boundary. | Admin-domain code calling another domain's Service must not import that domain's concrete Service. |

`internal/service/admin/contract` is for admin-level public service contracts that are stable and reused by multiple packages, such as audit logs, notifications, or file storage. Do not place DAO interfaces, per-feature repositories, or single-consumer test doubles there. Adapters belong in `internal/di` only when an implementation cannot directly satisfy the contract.

Contracts may include `*query.Query` only when the capability must participate synchronously in the caller's transaction, such as audited operation logs. Prefer context-only contracts for non-transactional or asynchronous capabilities.

Transaction-aware contracts must not start, commit, or rollback transactions internally. They must use the caller-provided `*query.Query` and keep side effects limited to the declared capability.
