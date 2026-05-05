# CLAUDE_CODING.md

> Coding standards for snowgo.
>
> [Back to CLAUDE.md](./CLAUDE.md)

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
| Struct tags | json + form + binding | - |

---

## 2. Constants

All in `internal/constant/`. Never inline.

| File | Scope |
|------|-------|
| `constant.go` | General (status values, default limits) |
| `cache_key.go` | Redis cache key prefixes |
| `permission.go` | RBAC permission strings |
| `mq.go` | RabbitMQ exchange/queue/routing key names |

Error codes: `pkg/xerror/` (5-digit scheme, separate registry).

---

## 3. Error Handling

| Layer | Rule |
|-------|------|
| API | `errors.As` 提取 `e.BizError`，用 `FailByError(c, bizErr.Code)` 响应。仅 binding 错误用 `Fail`。禁止 `Fail(c, code, err.Error())` 用于 service 错误 |
| Service | `e.NewBizError(e.Code)` 定义 sentinel。禁止 `errors.New` 用于业务逻辑错误。基础设施错误用 `fmt.Errorf("%w", err)` |
| DAO | Return directly — `errors.New` for validation, raw GORM for DB |
| Global | Never `panic()` in API/Service/DAO. Only `xlogger.Panic` for fatal init |

### BizError Pattern

Service 层用 `xerror.BizError` 携带 `xerror.Code`，API 层统一提取：

```go
// Service: 定义 sentinel
var ErrUserNotFound = e.NewBizError(e.UserNotFound)

// API: 提取并响应
var bizErr *e.BizError
if errors.As(err, &bizErr) {
    xresponse.FailByError(c, bizErr.Code)
    return
}
xlogger.ErrorfCtx(ctx, "...: %v", err)
xresponse.FailByError(c, e.FallbackCode)
```

新增业务错误只需：
1. 在 `pkg/xerror/error.go` 添加 Code
2. 在 Service 添加 BizError sentinel

（API handler 无需修改，`errors.As` 自动处理）

### Error Code Scheme

5-digit integers via `xerror.NewCode(category, code, msg)`. Duplicate codes panic at init.

| Range | Meaning |
|-------|---------|
| 0-504 | HTTP status codes |
| 1xxxx | Business errors |
| 2xxxx | System/infra errors |

Structure: `[level][module][specific]` — first digit = level, digits 2-3 = module, digits 4-5 = specific.

### HTTP Status Mapping

| Status | Trigger |
|--------|---------|
| 400 | Validation / bad input |
| 401 | Auth failure |
| 403 | Permission denied |
| 404 | Not found |
| 429 | Rate limit |
| 500 | Server error |

---

## 4. Logging

- Always use `xlogger.InfofCtx` / `xlogger.ErrorfCtx` (injects trace_id).
- Never `fmt.Printf` / `log.Println`.
- `Warn`: reserved for access logs via `xlogger.Access()`.
- `Info`: business events. `Error`: anomalies. `Debug`: disabled in production.
- Sensitive fields auto-masked: password, token, secret, access_token, refresh_token, phone, id_card, email.

---

## 5. Context Propagation

- All functions accepting `context.Context` must propagate to downstream calls.
- Extract auth data via `xauth` constants: `XUserId`, `XUserName`, `XIp`, `XSessionId`, `XTraceId`.
- Use `context.WithTimeout` / `WithCancel` only when adding a deadline scope.

---

## 6. Input Validation

- API layer is the gate. Validate before reaching Service.
- Gin binding tags: `binding:"required,max=64"`. Add explicit validation for business rules.
- Common tags: `required`, `max=N`, `min=N`, `email`, `oneof=A B`.

---

## 7. Concurrency

- No goroutines in Service/DAO unless justified.
- Propagate `context.Context` and handle cancellation/timeout.
- Distributed lock: `xlock.RedisLock` (`pkg/xlock/`). Callback-based — `TryLock()`, `Lock()`, `LockWithTries()`, `LockWithTriesTime()`. Unlock managed internally.
