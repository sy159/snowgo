# ARCHITECTURE.md

> Database, transactions, caching, query rules for snowgo.
>
> [Back to AGENTS.md](./AGENTS.md)

---

## 1. Layered Architecture

```
Router → API → Service → DAO → DAL (GORM Gen) → MySQL / Redis
```

- Each layer calls only the layer below. API → Service (never DAO). Service → DAO (never GORM Gen directly).
- `repo` (`dal/repo/repo.go`) provides `WriteQuery()` / `ReadQuery()` / `Query()` / `ChangeDB()`.
- DI: `di.GetContainer(c)` or `di.GetSystemContainer(c)`. Option pattern, LIFO close.

---

## 2. Database Design

### 2.1 Schema

Design principle: drive schema by query patterns, avoid over-engineering, choose types reasonably, cover high-frequency queries with indexes.

| Item | Rule |
|------|------|
| Table name | `<module>_<entity>` (e.g., `user`, `role_menu`) |
| Association table | `<entity_a>_<entity_b>` (e.g., `user_role`) |
| Foreign keys | None. Application-level integrity |
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

Column: `is_deleted TINYINT(1) NOT NULL DEFAULT 0` + `deleted_at DATETIME(6) DEFAULT NULL`. Index only if querying both states. Use `UpdateSimple`.

### 2.3 Index Design

| Type | Usage |
|------|-------|
| Single-column | High-cardinality field filtering |
| Composite | Multiple columns. Equality first, then range, then sort |
| Unique | Business uniqueness (uk_code, uk_username) |
| Covering | All SELECT columns included |
| Association | Composite unique (a_id, b_id) |

Left-prefix rule: `(a, b, c)` serves `(a)`, `(a, b)`, `(a, b, c)`. Range queries break the chain.

**Performance checklist**: avoid SELECT * on large tables; use EXPLAIN before committing complex queries; avoid implicit type conversion in WHERE; prefer covering indexes for hot paths; composite index for low-cardinality + high-cardinality columns; no leading wildcard LIKE in production queries.

---

## 3. Transactions

The Service layer owns transaction boundaries. DAO methods accept a caller-provided `*query.Query`, so the same DAO method can run inside a transaction (`tx`) or outside a transaction (`repo.Query()` / `repo.WriteQuery()`). DAO must not start, commit, or rollback transactions.

Use `WriteQuery().Transaction()` for:

- Multi-table writes.
- Business mutations that must persist operation logs atomically.
- Read-after-write logic that must be isolated from replica lag.

Independent single-table writes may run without an explicit transaction when there is no cross-table or operation-log atomicity requirement, for example login logs. Prefer `repo.WriteQuery()` for non-transactional writes when the write node must be explicit; `repo.Query()` may rely on dbresolver auto-routing.

Never call `container.SomeService.Method()` inside a transaction. Service MUST NOT directly use GORM Gen query APIs.

```go
err := db.WriteQuery().Transaction(func(tx *query.Query) error {
    // Pass tx to DAO methods
    obj, err := dao.CreateUser(ctx, tx, &model.SysUser{...})
    // Operation log within same tx
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
// DAO: unified signature, q source determined by caller
func (u *UserDao) CreateUser(ctx context.Context, q *query.Query, user *model.SysUser) (*model.SysUser, error) {
    return q.SysUser.WithContext(ctx).Omit(userDefaultSkipColumns...).Create(user)
}

// Service: inside transaction → pass tx
err := s.db.WriteQuery().Transaction(func(tx *query.Query) error {
    userObj, err = s.userDao.CreateUser(ctx, tx, &model.SysUser{...})
    return nil
})

// Service: outside transaction → pass repo query explicitly
userObj, err := s.userDao.CreateUser(ctx, s.db.WriteQuery(), &model.SysUser{...})
```

There is only one method per DAO operation — no separate `Transaction*Xxx` variants.

Operation logs for audited business mutations are synchronous within the same transaction for consistency.

**Read-write in transactions**: All reads and writes inside a transaction go to the write node. Outside transactions, `repo.Query()` auto-detects via dbresolver; use `WriteQuery()` or `ReadQuery()` when you need to override automatic routing, for example read-after-write to avoid replication lag.

**Reusing business logic in transactions**: When business logic from another service is needed inside a transaction, extract it as a DAO method or a stateless utility function in `pkg/`. Never call another service to avoid implicit transaction nesting or circular dependencies.

---

## 4. Caching Strategy

Service layer only. Keys: `constant.CacheXXXPrefix + value`. Invalidation **after** DB commit — never inside transaction. Cache set is non-blocking; failure logged, never propagated.

| Constant | Key Pattern | TTL | Scope |
|----------|-------------|-----|-------|
| CacheMenuTree | `account:menu_data` | 15 days | Menu tree |
| CacheUserRolePrefix | `account:user_role:<userId>` | 15 days | User-role mapping |
| CacheRolePermsPrefix | `account:role_perms:<roleId>` | 15 days | Role-permission |
| CacheRoleMenuPrefix | `account:role_menu:<roleId>` | 15 days | Role-menu |
| SystemDictPrefix | `system:dict:<code>` | 30 days (1h if empty) | Dict items |

Non-cache keys: `CacheLoginFailPrefix` (login failure, 3 min), `CacheRefreshJtiPrefix` (JWT refresh JTI).

Pattern: Read-through (cache → miss → DB → fill non-blocking). Write-behind (tx → commit → invalidate).

---

## 5. Interface Availability

Graceful degradation: Cache down → query DB. DB down → return stale cache.

External calls: `context.WithTimeout`. Retry transient errors only — max 3 times, exponential backoff. No retry on 4xx, validation errors, unique constraint violations.

Idempotency: unique index for create-by-key, request_id in Redis for duplicate detection, WHERE status for transitions, distributed lock + unique tx number for financial.

---

## 6. Query Optimization

Paginate all lists. Default limit: `constant.DefaultLimit` (10). Avoid N+1 (JOINs/Preload). GORM Gen scopes for dynamic filters.

| Concern | Guideline |
|---------|-----------|
| Trees | Load all rows, build in memory |
| Batch ops | Validate size limits, bulk insert/update |
| Fuzzy search | ES/Meilisearch. No LIKE '%term%' on large tables |
| Export | Stream results |

Performance targets: P99 read < 200ms, write < 500ms, slow SQL > 2s, cache hit rate > 90%.

---

## 7. Code Comments

Core and complex code must have comments. Simple code needs none. The following must have comments:

| What | Why |
|------|-----|
| Transaction boundaries | Where Transaction() begins/ends, tables involved |
| Cache behavior | What cached, when invalidated, TTL, why |
| Complex business logic | Non-obvious rules, hidden constraints |
| Index rationale | Why this index (or not) |
| Error handling branches | Why handled differently |

Style: one line. WHY, not WHAT.
