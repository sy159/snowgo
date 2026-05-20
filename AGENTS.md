# AGENTS.md

> **Go Version**: 1.25+
> **Status**: Active — single source of truth for engineering decisions.

## Document Index

| Document | Scope |
|----------|-------|
| [CODING.md](./CODING.md) | Naming, error handling, logging, context, validation, concurrency |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Database design, transactions, caching, query optimization |
| [OPERATIONS.md](./OPERATIONS.md) | Security, testing, review checklist |

---

## Core Rules

1. **Never** manually edit `internal/dal/model/` or `internal/dal/query/`.
2. Service owns transaction boundaries. DAO methods accept caller-provided `*query.Query` and never open transactions themselves. Multi-table mutations, and audited business mutations that require operation logs, use `WriteQuery().Transaction()` and pass `tx *query.Query` into DAO methods. Independent single-table writes may use a non-transactional `*query.Query` when atomic cross-table consistency is not required. Operation logs for audited business mutations are written synchronously within the same transaction.
3. Cache invalidation happens **after** DB commit, never inside a transaction.
4. Admin endpoints require `JWTAuth()` after login. Add `PermissionAuth(constant.PermXXXX)` only for endpoints that perform privileged management operations or expose scoped business data. Login-only endpoints, such as current user permissions, server info, and allowed dictionary lookups, must be documented in route comments. Middleware in `internal/router/middleware/auth.go`.
5. Use `xlogger.InfofCtx` / `xlogger.ErrorfCtx` for all logging.
6. Define sentinel errors in Service using `pkg/xerror/` codes; compare with `errors.Is`.
7. API layer validates all input before reaching Service.
8. No `panic()` in API/Service/DAO for business errors.
9. `created_at` mandatory for all tables; soft delete (`is_deleted tinyint(1) DEFAULT 0` + `deleted_at DATETIME(6) DEFAULT NULL`) is optional, decide per business need. Use for tables requiring audit/compliance/user undo (e.g., orders). Skip for high-volume logs and junction tables.
10. Run tests according to change scope before declaring complete: small, localized changes run affected package tests; broad/shared behavior changes run `go test ./...`. Always run `make lint`. Use coverage commands only when coverage is the explicit goal.
11. Core and complex code must have comments. Update README / Codex docs alongside code changes.

---

## AI Workflow

Non-trivial changes: output plan before coding. Steps: understand task → identify files → plan → note risks → verify.
