# CLAUDE_ARCHITECTURE.md

> Database, transactions, caching, query rules for snowgo.
>
> [Back to CLAUDE.md](./CLAUDE.md)

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
| Status/enum | VARCHAR with allowed values in column comments. Strings, not numbers |
| Money | DECIMAL. Semi-structured: JSON |
| Strings | VARCHAR(n) — set n based on actual business limits, avoid blanket VARCHAR(255) |

### 2.2 Soft Delete

USE: audit/compliance, referential integrity, user undo.
SKIP: high-volume logs, junction tables.

Column: `is_deleted TINYINT(1) NOT NULL DEFAULT 0`. Index only if querying both states. Use `UpdateSimple`.

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

All multi-table writes must use transactions. Never call `container.SomeService.Method()` inside a transaction. Service MUST NOT directly use GORM Gen query APIs.

```go
err := db.WriteQuery().Transaction(func(tx *query.Query) error {
    // Use Transaction* DAO variants
    // Operation log within same tx
    return nil
})
```

Read/write separation: `repo.WriteQuery()` for mutations, `repo.ReadQuery()` for reads, `repo.Query()` default (write node).

DAO pattern: `CreateXxx(ctx, model)` direct use, `TransactionCreateXxx(ctx, tx, model)` for transactions. Always use Transaction* variant inside transactions.

Operation logs: synchronous within transaction for consistency.

**Read-write in transactions**: All reads and writes inside a transaction go to the write node. Outside transactions, reads default to `ReadQuery()`; use `WriteQuery()` explicitly when strong consistency is required.

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

Core and complex code must have Chinese comments. Simple code needs none. The following must have comments:

| What | Why |
|------|-----|
| Transaction boundaries | Where Transaction() begins/ends, tables involved |
| Cache behavior | What cached, when invalidated, TTL, why |
| Complex business logic | Non-obvious rules, hidden constraints |
| Index rationale | Why this index (or not) |
| Error handling branches | Why handled differently |

Style: one line. WHY, not WHAT.
