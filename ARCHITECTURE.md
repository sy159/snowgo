# ARCHITECTURE.md

> Database, transactions, caching, query rules for snowgo.
>
> [Back to AGENTS.md](./AGENTS.md)

---

## 1. Layered Architecture

```
Router → API → Service → DAO → DAL (GORM Gen) → MySQL
                       ├→ Cache → Redis
                       ├→ MQ
                       └→ External services
```

- Keep business dependencies directed through the declared boundaries. API calls Service, never DAO; Service calls DAO for database access, never GORM Gen directly. Service may call approved infrastructure components and stable contracts.
- `repo` (`dal/repo/repo.go`) provides `WriteQuery()` / `ReadQuery()` / `Query()` / `ChangeDB()`.
- DI: `di.GetContainer(c)` or `di.GetSystemContainer(c)`. Option pattern, LIFO close.
- Cross-package Service dependencies use stable contracts under `internal/service/admin/contract` for shared capabilities. Business packages depend on contracts, not concrete Services from other business packages.

---

## 2. Database Design

### 2.1 Schema

Design principle: drive schema by query patterns, avoid over-engineering, choose types reasonably, cover high-frequency queries with indexes.

| Item | Rule |
|------|------|
| Table name | System/admin tables use `sys_<entity>`; business tables use `<module>_<entity>` when a module prefix is needed |
| Association table | `<entity_a>_<entity_b>` with the same prefix strategy, e.g. `sys_user_role` |
| Foreign keys | None. Enforce application-level integrity with validation, transactions, unique constraints, and reconciliation or cleanup where needed |
| created_at | Mandatory. TIMESTAMP/DATETIME with DEFAULT CURRENT_TIMESTAMP |
| updated_at | Only for tables with UPDATE |
| Primary key | BIGINT default. INT for small/config tables |
| NOT NULL | Default. Avoid nullable unless truly optional |
| Boolean | TINYINT(1) DEFAULT 0. 0 = false, 1 = true |
| Status/enum | TINYINT for large tables (efficiency), VARCHAR for small/config tables (readability). Document values in column comments |
| Money | DECIMAL. Semi-structured: JSON |
| Strings | VARCHAR(n) — set n based on actual business limits, avoid blanket VARCHAR(255) |

### 2.2 Soft Delete

Non-mandatory. Decide per business need:

- **USE**: audit/compliance, referential integrity, user undo (e.g., orders, payments)
- **SKIP**: high-volume logs, junction tables, simple config tables

Column: `is_deleted TINYINT(1) NOT NULL DEFAULT 0` + `deleted_at DATETIME(6) DEFAULT NULL`. Do not index `is_deleted` alone by default; include it in a query-driven composite index only when EXPLAIN and selectivity justify it. Use `UpdateSimple`.

### 2.3 Index Design

| Type | Usage |
|------|-------|
| Single-column | High-cardinality field filtering |
| Composite | Multiple columns. Equality first, then range, then sort |
| Unique | Business uniqueness (uk_code, uk_username) |
| Covering | All SELECT columns included |
| Association | Composite unique (a_id, b_id) |

Left-prefix rule: `(a, b, c)` serves predicates starting with `(a)`, `(a, b)`, or `(a, b, c)`. A range predicate usually prevents later columns from further narrowing the index scan; verify the actual plan with EXPLAIN.

**Performance checklist**: avoid SELECT * on large tables; use EXPLAIN before committing complex queries; avoid implicit type conversion in WHERE; prefer covering indexes for hot paths; composite index for low-cardinality + high-cardinality columns; no leading wildcard LIKE in production queries.

---

## 3. Transactions

The Service layer owns transaction boundaries. DAO methods accept a caller-provided `*query.Query`, so the same DAO method can run inside a transaction (`tx`) or outside a transaction (`repo.Query()` / `repo.WriteQuery()`). DAO must not start, commit, or rollback transactions.

Use `WriteQuery().Transaction()` for:

- Multi-table writes.
- Business mutations that must persist operation logs atomically.

Do not open a transaction solely because a read follows a write. Independent single-table writes may run without an explicit transaction when there is no cross-table or operation-log atomicity requirement, for example login logs. When read-after-write consistency only requires avoiding replica lag, use `repo.WriteQuery()` for the read; use a transaction only when atomicity or transactional consistency is actually required. `repo.Query()` may rely on dbresolver auto-routing.

Never call business Service methods through `container.SomeService.Method()` inside a transaction. Service MUST NOT directly use GORM Gen query APIs.

```go
err := db.WriteQuery().Transaction(func(tx *query.Query) error {
    // 在同一事务内传递 tx，保证多表写入和审计日志原子一致。
    if _, err := dao.CreateUser(ctx, tx, &model.SysUser{...}); err != nil {
        return fmt.Errorf("create user: %w", err)
    }
    // 审计日志必须复用该 tx，避免业务数据与审计记录不一致。
    if err := logWriter.CreateOperationLog(ctx, tx, &contract.OperationLogInput{...}); err != nil {
        return fmt.Errorf("create operation log: %w", err)
    }
    return nil
})
```

Read/write separation: `repo.WriteQuery()` forces write node, `repo.ReadQuery()` forces read replicas, `repo.Query()` relies on dbresolver auto-detection (SELECT→replica, INSERT/UPDATE/DELETE→source).

**DAO `*query.Query` parameter convention**: DAO methods that may participate in a transaction accept `*query.Query` as a parameter. The DAO does not care whether it is in a transaction — the Service layer decides what to pass.

| Context | DAO `q` parameter | Source |
|---------|-------------------|--------|
| Inside transaction | `tx` | From `Transaction(func(tx *query.Query) error)` |
| Outside transaction | `repo.Query()` / `repo.WriteQuery()` | From the Service's repository |

```go
// DAO 使用统一签名，使调用方能够决定是否参与事务。
func (u *UserDao) CreateUser(ctx context.Context, q *query.Query, user *model.SysUser) (*model.SysUser, error) {
    return q.SysUser.WithContext(ctx).Omit(userDefaultSkipColumns...).Create(user)
}

// 事务内传入 tx，确保后续写操作处于同一原子边界。
err := s.db.WriteQuery().Transaction(func(tx *query.Query) error {
    userObj, err = s.userDao.CreateUser(ctx, tx, &model.SysUser{...})
    return err
})

// 非事务写入显式使用写库，避免路由语义不清晰。
userObj, err := s.userDao.CreateUser(ctx, s.db.WriteQuery(), &model.SysUser{...})
```

There is only one method per DAO operation — no separate `Transaction*Xxx` variants.

Operation logs for audited business mutations are synchronous within the same transaction for consistency.

**Read-write in transactions**: All reads and writes inside a transaction go to the write node. Outside transactions, `repo.Query()` auto-detects via dbresolver; use `WriteQuery()` or `ReadQuery()` when you need to override automatic routing, for example read-after-write to avoid replication lag.

**Reusing business logic in transactions**: When business logic from another service is needed inside a transaction, extract it as a DAO method or a stateless utility function in `pkg/`. Transaction-safe infrastructure contracts may be used only when they are explicitly designed to receive the caller's `*query.Query` and do not start nested transactions, such as synchronous operation-log writing.

---

## 4. Caching Strategy

Service layer only. Keys: `constant.CacheXXXPrefix + value`. Invalidation happens **after** DB commit, never inside a transaction. Cache reads and fills are synchronous calls but may be best-effort when falling back to the database preserves correctness. Every ignored cache failure must be logged or measured; invalidation failures require enough context to diagnose the stale-key risk and a retry, versioning, short-TTL, or reconciliation strategy when stale data is business-critical. Existing silent `_ = cache...` calls are technical debt, not a pattern to copy.

| Constant | Key Pattern | TTL | Scope |
|----------|-------------|-----|-------|
| CacheMenuTree | `account:menu_data` | 15 days | Menu tree |
| CacheUserRolePrefix | `account:user_role:<userId>` | 15 days | User-role mapping |
| CacheRolePermsPrefix | `account:role_perms:<roleId>` | 15 days | Role-permission |
| CacheRoleMenuPrefix | `account:role_menu:<roleId>` | 15 days | Role-menu |
| SystemDictPrefix | `system:dict:<code>` | 30 days (1h if empty) | Dict items |

Non-cache keys: `CacheLoginFailPrefix` (login failure, 3 min), `CacheRefreshJtiPrefix` (JWT refresh JTI).

Pattern: cache-aside read (`cache → miss → DB → best-effort fill`) and post-commit invalidation (`DB transaction → commit → invalidate`).

---

## 5. Availability & Resilience

Graceful degradation: cache down → query DB and log the cache error. DB down → return stale cache only for read paths that explicitly implement stale-cache semantics; otherwise return a controlled system error.

External calls must have a caller-bounded timeout. Retry only transient failures for idempotent or idempotency-protected operations, within the total deadline and component retry budget. Use exponential backoff with jitter and honor `Retry-After` when applicable. Do not retry validation, authentication, authorization, unique-constraint, or other permanent failures; 408 and 429 require an explicit policy. Default to at most three total attempts unless the component has a stricter documented policy.

Idempotency: unique index for create-by-key, request_id in Redis for duplicate detection, WHERE status for transitions, distributed lock + unique tx number for financial.

Queue consumers must be idempotent. Acknowledge messages only after durable side effects succeed; on retryable failures, reject/requeue according to queue policy; on poison messages, route to a dead-letter queue or persist a failure record for manual handling.

---

## 6. Query Optimization

Paginate all lists. Default limit: `constant.DefaultLimit` (10). Avoid N+1 (JOINs/Preload). GORM Gen scopes for dynamic filters.

| Concern | Guideline |
|---------|-----------|
| Trees | For bounded admin trees, load the required rows and build in memory; paginate or redesign unbounded hierarchies |
| Batch ops | Validate size limits, bulk insert/update |
| Fuzzy search | Avoid leading-wildcard LIKE on large tables; evaluate a search engine only when query volume and search requirements justify it |
| Export | Stream results |

Define latency, error-rate, throughput, and cache-hit objectives per endpoint or module from production telemetry. The slow-SQL threshold is configuration-driven. Example targets are not release commitments unless they are documented, measurable, and owned by the affected module.

---

## 7. Code Comments

Core and complex code must have concise Chinese comments. Simple code needs none. Comments explain **why** an invariant, boundary, or decision exists; they must not narrate obvious code. The following must have comments:

| What | Why |
|------|-----|
| Transaction boundaries | Where Transaction() begins/ends, tables involved |
| Cache behavior | What cached, when invalidated, TTL, why |
| Complex business logic | Non-obvious rules, hidden constraints |
| Index rationale | Why this index (or not) |
| Error handling branches | Why handled differently |

Style: prefer one concise line. Write WHY, not WHAT.
