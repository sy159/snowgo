# CLAUDE.md

> **Version**: 1.3.0
> **Last Updated**: 2026-04-26
> **Go Version**: 1.25+
> **Status**: Active — single source of truth for engineering decisions.

Entry point for all engineers and AI assistants working in this repository.

---

## Document Index

| Document | Scope |
|----------|-------|
| [CLAUDE_CODING.md](./CLAUDE_CODING.md) | Naming, error handling, logging, validation, concurrency |
| [CLAUDE_ARCHITECTURE.md](./CLAUDE_ARCHITECTURE.md) | Layered architecture, database design, transactions, caching, performance |
| [CLAUDE_OPERATIONS.md](./CLAUDE_OPERATIONS.md) | Security, testing standards, Git workflow, CI/CD, deployment, review checklist |

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

When handling any task, follow this 5-phase workflow. Do not skip steps.

| Phase | Action | Key Rules |
|-------|--------|-----------|
| **1. Analyze** | Read related code, call chains, dependencies, and existing conventions. | Reuse existing patterns before inventing new ones. |
| **2. Design** | Identify which layer(s) need changes. Output a plan before coding. | Do not introduce new abstractions unless necessary. |
| **3. Implement** | Make minimal, focused changes. Match existing naming, error handling, and transaction style. | One task at a time. No unrelated refactoring. |
| **4. Validate** | Run relevant tests. Expand scope if public modules are affected. | If tests fail, the task is **not** complete. |
| **5. Document** | Update `README.md`, API docs, config docs, or CLAUDE sub-documents as needed. | Convention changes must update docs **before** code lands. |

### Design-before-code Requirement

For any non-trivial change, output the following **before** modifying code:

1. Task understanding — what and why.
2. Files to modify — and why these files.
3. Implementation plan.
4. Risk points and compatibility impact.
5. Verification approach.

### Task Output Format

Structure your response as follows:

1. **Task Understanding**
2. **Implementation Plan**
3. **Code Changes**
4. **Testing** — new/updated tests, commands run, results.
5. **Documentation Updates**
6. **Risks & Follow-ups**

---

## Top 10 Rules (Memorize These)

1. **Never** manually edit `internal/dal/model/` or `internal/dal/query/`.
2. All mutations use `WriteQuery().Transaction()`.
3. Cache invalidation happens **after** DB commit, never inside a transaction.
4. Admin endpoints require `JWTAuth()` + `PermissionAuth(constant.PermXXXX)`.
5. Use `xlogger.InfofCtx(ctx, ...)` / `xlogger.ErrorfCtx(ctx, ...)` for all logging.
6. Define sentinel errors in Service; compare with `errors.Is`, never string comparison.
7. API layer validates all input before reaching Service.
8. No `panic()` in API/Service/DAO for business errors.
9. Soft delete is a **business decision** per table; all tables have `created_at`/`updated_at`.
10. Run `go test ./... -cover` and ensure CI passes before declaring complete.

---

## Document Maintenance

- These documents are the single source of truth.
- Update the relevant sub-document **before** landing a convention change.
- Update `Last Updated` on every modification; bump `Version` for breaking changes.
- Keep all sub-document versions in sync with this entry document.
