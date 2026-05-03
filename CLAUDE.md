# CLAUDE.md

> **Go Version**: 1.25+
> **Status**: Active — single source of truth for engineering decisions.

## Document Index

| Document | Scope |
|----------|-------|
| [CLAUDE_CODING.md](./CLAUDE_CODING.md) | Naming, error handling, logging, context, validation, concurrency |
| [CLAUDE_ARCHITECTURE.md](./CLAUDE_ARCHITECTURE.md) | Database design, transactions, caching, query optimization |
| [CLAUDE_OPERATIONS.md](./CLAUDE_OPERATIONS.md) | Security, testing, review checklist |

---

## Core Rules

1. **Never** manually edit `internal/dal/model/` or `internal/dal/query/`.
2. All mutations use `WriteQuery().Transaction()`. Operation logs written synchronously within the same transaction.
3. Cache invalidation happens **after** DB commit, never inside a transaction.
4. Admin endpoints require `JWTAuth()` + `PermissionAuth(constant.PermXXXX)`. Middleware in `internal/router/middleware/auth.go`.
5. Use `xlogger.InfofCtx` / `xlogger.ErrorfCtx` for all logging.
6. Define sentinel errors in Service using `pkg/xerror/` codes; compare with `errors.Is`.
7. API layer validates all input before reaching Service.
8. No `panic()` in API/Service/DAO for business errors.
9. `created_at` mandatory for all tables; soft delete via `is_deleted tinyint(1) DEFAULT 0` + `deleted_at DATETIME(6) DEFAULT NULL` 实现。
10. Run `go test ./... -cover` and `make lint` before declaring complete.
11. Core and complex code must have Chinese comments. Update README / CLAUDE docs alongside code changes.

---

## AI Workflow

Non-trivial changes: output plan before coding. Steps: understand task → identify files → plan → note risks → verify.
