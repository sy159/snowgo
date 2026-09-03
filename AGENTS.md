# AGENTS.md

> AI coding-agent execution manual for snowgo.
>
> Go: 1.26+ (toolchain: go1.26.7).
>
> Goal: deliver the smallest necessary, verifiably correct change without compromising correctness, security, or maintainability.

## 1. Instruction Precedence and Document Roles

Resolve conflicts in this order:

1. The user's current explicit request, constraints, and explicitly confirmed exceptions.
2. The hard rules in this `AGENTS.md`.
3. The relevant project-specific references.
4. Existing conventions in the affected module, unless they conflict with a higher-priority rule.

The user's request defines the outcome and scope, but do not infer permission to bypass generated-file, data-integrity, security, or authorization constraints. Surface the conflict and obtain explicit confirmation before treating it as a policy exception.

`CODING.md` defines code-level practices; `ARCHITECTURE.md` defines layers, data, transactions, caching, and queries; `OPERATIONS.md` defines security, testing, configuration, release, and review practices.

Before implementation, read only the references relevant to the affected change surface. Use the Reference Index to decide what is required. Do not read unrelated references for a small localized change; if relevance or a boundary is unclear, read the applicable reference or ask.

Material ambiguity affecting APIs, data, permissions, security, compatibility, or scope must be surfaced with viable options. Do not silently select a high-risk interpretation.

---

## 2. AI Coding Behavior Guidelines

### Think Before Coding

For non-trivial changes, understand the requested outcome, identify material assumptions and ambiguities, consider affected boundaries and existing architecture, and define how the result will be verified. Ask when missing information would materially change the solution.

### Simplicity and Existing Patterns First

- Write only the minimum code needed for the current requirement. Do not add speculative features, abstractions, configuration, compatibility layers, or extension points.
- Before introducing a pattern for dependency injection, errors, logging, transactions, caching, API responses, authorization, or configuration, search for and reuse the established project pattern unless it is insufficient.
- Add a third-party dependency only when the standard library and existing dependencies are insufficient. Assess maintenance, license, security, and operational impact; then run `go mod tidy` and relevant tests.

### Surgical, Real Changes

- Inspect and preserve existing working-tree changes. Every changed line must be traceable to the task; do not opportunistically refactor, format, or clean up adjacent code.
- Do not silently ignore errors or turn unexpected errors into successful results. An explicitly best-effort operation must document its reason and log or measure failure.
- Do not claim a capability is complete when it is a TODO, placeholder, production stub, hard-coded success, or mock that does not perform the required behavior.

### Evidence-Based Execution

Define verifiable acceptance criteria. Prefer regression tests for bugs and cover important success, failure, authorization, and boundary paths for changed behavior. Report only verification actually performed, including skipped checks, reasons, and residual risk.

---

## 3. Standard Workflow

1. **Understand:** read the task, relevant code, and applicable references; identify the affected boundaries.
2. **Inspect:** check existing working-tree changes and define success criteria, risks, and relevant verification.
3. **Plan:** for non-trivial work, state concise steps with verification and material trade-offs.
4. **Implement:** follow existing patterns and the hard constraints below; keep the diff minimal.
5. **Verify:** run checks appropriate to the actual change scope and inspect the diff.
6. **Hand off:** report changed behavior, verification evidence, skipped checks, compatibility or release impact, and known risks.

---

## 4. Project Architecture Hard Constraints

- The primary business call direction is fixed: `Router → API → Service → DAO → DAL (GORM Gen) → MySQL`. Service may also use approved cache, queue, and external-service components. API never calls DAO directly; Service never calls GORM Gen query APIs directly.
- **Never manually edit** `internal/dal/model/` or `internal/dal/query/`. After a schema change, run `make gen do=add` or `make gen do=update` and inspect the generated diff. `make new-module domain=business module=order` produces a skeleton only.
- Service owns transaction boundaries. DAO receives the caller-provided `*query.Query` and never begins, commits, or rolls back transactions. Do not call business Services inside an open transaction.
- Use `WriteQuery().Transaction()` only for atomic multi-table writes, audited mutations, or other true transactional-consistency needs. Do not open a transaction solely because a read follows a write; use `repo.WriteQuery()` when primary-read consistency is sufficient to avoid replica lag.
- Audited mutations write operation logs synchronously in the same transaction. Service owns cache behavior; invalidate cache only after a successful database commit.
- API performs transport validation: binding, required fields, format, length, enums, and basic request limits. Service validates business rules and state-dependent preconditions.
- Admin endpoints after login require `JWTAuth()`; privileged or scoped endpoints require `PermissionAuth(constant.PermXxx)`. Every endpoint is public, login-only, or permission-protected. Explain only non-obvious public or login-only exceptions in code or route configuration.
- Same-package Services inject concrete Services. Cross-package shared infrastructure uses stable contracts in `internal/service/admin/contract`; narrow one-off dependencies use consumer-local interfaces. Contracts contain interfaces and DTOs only.
- Every table requires `created_at`; choose soft delete based on business need. Core or complex logic needs concise **Chinese** comments explaining **why**, not obvious code.

---

## 5. Mandatory Checks by Change Type

| Change surface | Required focus |
|----------------|----------------|
| Go code, errors, logging, context, validation, concurrency, dependencies | Read `CODING.md`; preserve established patterns and error semantics. |
| API or authorization | Read `CODING.md` and `OPERATIONS.md`; classify endpoint, validate transport input, preserve contract and error-code compatibility. |
| Schema, DAO, transactions, cache, queries | Read `ARCHITECTURE.md`; for schema changes also read `OPERATIONS.md`. Generate DAL code and assess indexes, migration artifacts, transaction scope, cache timing, and compatibility. |
| Writes, retries, queues, external calls | Read `ARCHITECTURE.md` and `OPERATIONS.md`; define idempotency, timeout, retry, durable side effects, and failure handling. |
| Security, configuration, release, observability | Read `OPERATIONS.md`; assess secrets, authorization, rollout, rollback, and production checks. |

---

## 6. Verification and Hand-Off Requirements

- Run affected-package tests for localized code changes; run `go test ./...` for shared or broad behavior changes. Run integration tests for affected critical paths when dependencies are available.
- For Go changes, format only changed files with `gofmt`, run scope-appropriate tests, and run `make lint`. If local lint tooling is unavailable, report the skipped check and do not claim lint passed; CI uses `golangci-lint v2.12.2` as the canonical gate. For documentation-only or non-Go changes, run relevant link, format, and diff checks.
- Do not change production behavior solely to satisfy a test when it conflicts with the requirement. Surface test/specification conflicts instead of optimizing blindly for green tests.
- Before hand-off, review `git diff` or equivalent and confirm every modified file and meaningful change belongs to the task and no unrelated user change was overwritten.

---

## 7. Reference Index

| Document | Read when |
|----------|-----------|
| [`CODING.md`](./CODING.md) | Changing Go code, errors, logging, context, validation, concurrency, or Service dependencies. |
| [`ARCHITECTURE.md`](./ARCHITECTURE.md) | Changing layers, database, DAO, transactions, caching, external calls, or query performance. |
| [`OPERATIONS.md`](./OPERATIONS.md) | Changing authorization, security, tests, configuration, release behavior, queues, or delivery review. |

Update README or engineering references only when code behavior, configuration, APIs, or operational workflows actually change; avoid mechanical updates to unrelated documentation.
