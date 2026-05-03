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
| API | `xresponse.FailByError` (business), `xresponse.Fail` (validation) |
| Service | Sentinel errors via `pkg/xerror/` codes. `errors.Is` comparison. Wrap infra errors: `fmt.Errorf("%w", err)` |
| DAO | Return directly — `errors.New` for validation, raw GORM for DB |
| Global | Never `panic()` in API/Service/DAO. Only `xlogger.Panic` for fatal init |

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
