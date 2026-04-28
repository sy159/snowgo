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
| Constants | CamelCase exported, camelCase unexported | CacheUserRolePrefix, defaultLimit |
| DTOs (API) | {Entity}Info, {Entity}List, {Entity}Param | UserInfo, UserList, UserParam |
| DTOs (Service) | avoid json tags unless crossing boundaries | UserCondition |
| Struct tags | json + form + binding | - |

---

## 2. Constants Management

All constants live in `internal/constant/`. Unified management for:

| File | Scope |
|------|-------|
| `constant.go` | General constants (status values, default limits, etc.) |
| `cache_key.go` | Redis cache key prefixes (e.g., `CacheMenuTree`, `CacheUserRolePrefix`) |
| `permission.go` | RBAC permission strings (e.g., `PermUserList`) |
| `mq.go` | RabbitMQ exchange, queue, routing key names |

Never define constants inline or in service/API layers. Error codes live separately in `pkg/xerror/` with 5-digit scheme.

---

## 3. Error Handling (MANDATORY)

| Layer | Rule |
|-------|------|
| API | xresponse.FailByError for business errors; xresponse.Fail for validation errors |
| Service | Define sentinel errors using registered error codes from pkg/xerror/. Compare with errors.Is. Wrap infra errors with fmt.Errorf("%w"). |
| DAO | Return errors directly - errors.New for validation, raw GORM errors for DB failures |
| Global | Never panic in API/Service/DAO. Only xlogger.Panic for fatal init failures |

Error codes live in pkg/xerror/. Pattern: 5-digit integer (e.g., 10201, 20101) registered via xerror.NewCode(category, code, msg). Business errors start with 1, system errors start with 2. New codes must be registered in the global registry.

HTTP status: 400 validation, 401 auth, 403 permission, 404 not found, 429 rate limit, 500 server error.

---

## 4. Logging (MANDATORY)

- Always use xlogger.InfofCtx / xlogger.ErrorfCtx for trace_id injection.
- Never use fmt.Printf / log.Println in production code.
- Warn level is reserved for access logs via xlogger.Access(). Not used in business logic.
- Info: business events. Error: anomalies. Debug: disabled in production.

Sensitive fields auto-masked in access logs: password, token, secret, access_token, refresh_token, phone, id_card, email. Add new fields via middleware.sensitiveRoots.

---

## 5. Input Validation (MANDATORY)

- API layer is the gate. Validate before reaching Service.
- Use Gin binding tags: binding:"required,max=64". Add explicit validation for business rules. Never trust frontend.
- Common tags: required, max=N, min=N, email, oneof=A B.

---

## 6. Concurrency

- No goroutines in Service/DAO unless justified.
- Propagate context.Context and handle cancellation/timeout.
- Use xlock.RedisLock for distributed critical sections. Callback-based API — lock and unlock are managed internally via fn callback.
