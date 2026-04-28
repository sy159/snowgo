# CLAUDE.md

> **Go Version**: 1.25+
> **Status**: Active — single source of truth for engineering decisions.

Entry point for all engineers and AI assistants working in this repository.

---

## Document Index

| Document | Scope |
|----------|-------|
| [CLAUDE_CODING.md](./CLAUDE_CODING.md) | Naming, error handling, logging, validation, concurrency |
| [CLAUDE_ARCHITECTURE.md](./CLAUDE_ARCHITECTURE.md) | Layered architecture, database design, transactions, caching, interface availability & performance |
| [CLAUDE_OPERATIONS.md](./CLAUDE_OPERATIONS.md) | Security, testing, observability, Git workflow, deployment, review checklist |

---

## Project Overview

`snowgo` is a production-grade Go admin scaffold built on **Gin + GORM Gen**. It targets Go 1.25+ and enforces:

- Strict layered architecture (Router → API → Service → DAO → DAL)
- RBAC authorization with JWT dual-token authentication
- Read-write database separation, Redis caching, distributed locking
- RabbitMQ messaging, OpenTelemetry tracing, Prometheus metrics

**Goal**: All code must be observable, secure, testable, and performant. No shortcuts.

---

## AI Execution Workflow (MANDATORY)

Follow: **Analyze → Design → Implement → Validate → Document**. Non-trivial changes: output plan before coding.

1. Task understanding
2. Files to modify and why
3. Implementation plan
4. Risk points and compatibility impact
5. Verification approach

---

## Top 10 Rules (Memorize These)

1. **Never** manually edit `internal/dal/model/` or `internal/dal/query/`.
2. All mutations use `WriteQuery().Transaction()`. Operation logs written synchronously within the same transaction.
3. Cache invalidation happens **after** DB commit, never inside a transaction.
4. Admin endpoints require `JWTAuth()` + `PermissionAuth(constant.PermXXXX)`. Permission strings in `internal/constant/permission.go`.
5. Use `xlogger.InfofCtx(ctx, ...)` / `xlogger.ErrorfCtx(ctx, ...)` for all logging.
6. Define sentinel errors in Service using registered error codes from `pkg/xerror/`; compare with `errors.Is`, never string comparison.
7. API layer validates all input before reaching Service.
8. No `panic()` in API/Service/DAO for business errors.
9. `created_at` is mandatory for all tables; `updated_at` only for tables with update operations. Soft delete is a **business decision** — use `is_deleted tinyint(1) DEFAULT 0` with an index only if querying deleted rows is needed.
10. Run `go test ./... -cover` and ensure CI passes before declaring complete. Complex/important code must have WHY comments (transaction boundaries, cache behavior, business rules).

---

## Document Maintenance

- These documents are the single source of truth.
- Update the relevant sub-document **before** landing a convention change.
- Keep all sub-documents in sync with this entry document.
