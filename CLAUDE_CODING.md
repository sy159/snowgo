# CLAUDE_CODING.md

> **Version**: 1.3.0
> **Last Updated**: 2026-04-26
> Coding standards, naming conventions, error handling, logging, input validation, and concurrency rules for `snowgo`.
>
> [Back to CLAUDE.md](./CLAUDE.md)

---

## Table of Contents

1. [Naming](#1-naming)
2. [Error Handling](#2-error-handling)
3. [Logging](#3-logging)
4. [Input Validation](#4-input-validation)
5. [Concurrency](#5-concurrency)

---

## 1. Naming

| Category | Rule | Example |
|----------|------|---------|
| Packages | lowercase, no underscores | `account`, `system`, `xlogger` |
| Files | snake_case | `user_service.go`, `account_router.go` |
| Interfaces | describe behavior | `UserRepo` (not `IUser`) |
| Structs | noun-based | `UserService`, `DictParam` |
| Methods | verb-based | `CreateUser`, `GetUserList`, `ValidatePassword` |
| Constants | CamelCase exported, camelCase unexported | `CacheUserRolePrefix`, `defaultLimit` |
| DTOs (API) | `{Entity}Info`, `{Entity}List`, `{Entity}Param` | `UserInfo`, `UserList`, `UserParam` |
| DTOs (Service) | Avoid `json` tags unless crossing boundaries | — |

---

## 2. Error Handling (MANDATORY)

### Layer Rules

| Layer | Rule |
|-------|------|
| API | `xresponse.FailByError(c, e.SomeError)` for client errors; `xresponse.Fail(c, code, msg)` for edge cases. |
| Service | Define sentinel errors: `var ErrUserNotFound = errors.New(e.UserNotFound.GetErrMsg())`. Compare with `errors.Is(err, ErrUserNotFound)` — **never compare error strings.** |
| Cross-layer | Wrap with `errors.WithMessage(err, "context")` when propagating. Log at origin; wrap when crossing. |
| DAO | Return `errors.WithStack(err)` for raw GORM errors. Never swallow errors. |
| Global | **Never panic in API/Service/DAO.** Only `xlogger.Panic` for fatal init failures (e.g., DB connection impossible). |

### HTTP Status Codes

| Code | Usage |
|------|-------|
| 400 | Validation errors |
| 401 | Authentication failure |
| 403 | Permission denied |
| 404 | Missing resource |
| 429 | Rate limited |
| 500 | Unexpected server errors |

---

## 3. Logging (MANDATORY)

- **Always** use `xlogger.InfofCtx(ctx, ...)` / `xlogger.ErrorfCtx(ctx, ...)` so `trace_id` is automatically injected.
- **Never** use `fmt.Printf` / `log.Println` in production code. Console output allowed only in middleware access logs for dev readability.

### Log Levels

| Level | When to use |
|-------|-------------|
| `Info` | Business events (user created, login success) |
| `Error` | Failures requiring investigation. Include `zap.Error(err)` field. |
| `Debug` | Verbose diagnostics (disabled in production) |

### Sensitive Data Masking

Access logs auto-mask: `password`, `token`, `secret`, `access_token`, `refresh_token`, `phone`, `id_card`, `email`.

- To add new sensitive fields, update `middleware.sensitiveRoots` in `internal/router/middleware/`.

---

## 4. Input Validation (MANDATORY)

- **API layer is the gate.** All user input must be validated before reaching Service.
- Use Gin binding tags: `binding:"required,max=64"`.
- Add **explicit validation** for business rules (password complexity, ID > 0, enum values).
- **Never trust frontend.** Re-validate permissions server-side.

---

## 5. Concurrency

- **No goroutines in Service/DAO** unless justified (e.g., background cache warm-up).
- If spawning goroutines, propagate `context.Context` and handle cancellation/timeout.
- Use `xlock.RedisLock` for distributed critical sections. Always `defer unlock()`.
