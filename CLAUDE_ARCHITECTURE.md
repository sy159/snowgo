# CLAUDE_ARCHITECTURE.md

> Architecture, database design, transactions, caching, interface availability & performance for snowgo.
>
> [Back to CLAUDE.md](./CLAUDE.md)

---

## Table of Contents

1. [Layered Architecture](#1-layered-architecture)
2. [Dependency Injection](#2-dependency-injection)
3. [DAL Code Generation](#3-dal-code-generation)
4. [Database Design](#4-database-design)
5. [Transactions](#5-transactions)
6. [Caching Strategy](#6-caching-strategy)
7. [Interface Availability & Performance](#7-interface-availability--performance)
8. [Query Optimization](#8-query-optimization)
9. [Feature Development Workflow](#9-feature-development-workflow)
10. [Code Comments](#10-code-comments-mandatory)

---

## 1. Layered Architecture

```
Router -> API -> Service -> DAO -> DAL (GORM Gen) -> MySQL / Redis
```

| Layer | Responsibility | Rules |
|-------|---------------|-------|
| Router | HTTP routing, middleware mounting | No business logic, no direct DB calls |
| API | Request validation, response formatting | Input binding, DTO conversion. No transactions |
| Service | Business orchestration, caching, transactions | ALL business rules. Coordinates DAO within transactions |
| DAO | Data access abstraction | Wraps GORM Gen. Direct + transaction methods. No business logic |
| DAL | Auto-generated model + query | Never hand-edit. Generated via make gen |

Communication rules: Each layer calls only the layer below. API calls Service (never DAO). Service calls DAO (never GORM Gen directly). DAO wraps GORM Gen, returns errors directly.

---

## 2. Dependency Injection

All infrastructure in internal/di/container.go. Option pattern for configuration. CloseManager shuts down in LIFO order. Access in handlers: di.GetContainer(c) or di.GetSystemContainer(c).

---

## 3. DAL Code Generation

internal/dal/model/ and internal/dal/query/ are machine-generated. Workflow: design schema -> make gen add / make gen update -> make gen query. Manual edits will be overwritten and break production.

---

## 4. Database Design

### 4.1 Schema

| Item | Rule |
|------|------|
| Table name | t_<module>_<entity> or <module>_<entity>. Consistent within project |
| Association table | <entity_a>_<entity_b> (e.g., user_role) |
| Foreign keys | None. Application-level integrity only |
| created_at | Mandatory. TIMESTAMP or DATETIME with DEFAULT CURRENT_TIMESTAMP |
| updated_at | Only for tables with UPDATE operations. Omit for read-only config tables |
| Primary key | INT for small/config tables (< 100M rows); BIGINT for high-growth tables. New tables default to BIGINT |
| NOT NULL | Default. Avoid nullable columns unless truly optional |
| Boolean | TINYINT(1) DEFAULT 0. 0 = false, 1 = true |
| Status/enum | VARCHAR with allowed values in column comments (e.g., 'Allowed: Active,Inactive'). Use strings, not numbers. Constants in internal/constant/constant.go |
| Strings | VARCHAR(n) - pick n carefully, not all 255 |
| Money | DECIMAL. Semi-structured data: JSON |

### 4.2 Soft Delete

USE when: audit/compliance required, referential integrity matters, users expect undo.
DO NOT USE when: high-volume log tables, junction tables, storage cost sensitive.

If used: column is_deleted TINYINT(1) NOT NULL DEFAULT 0. Index only if querying both deleted and undeleted rows. Use UpdateSimple; do not rely on GORM Delete hook. Document rationale.

### 4.3 Index Design

Design indexes based on actual query patterns, not theoretical possibilities.

When to create: WHERE filter columns (frequent), ORDER BY, JOIN ON, GROUP BY. High-cardinality columns. Low-cardinality (boolean, gender) should use composite indexes, not standalone.

| Type | Usage |
|------|-------|
| Single-column | High-cardinality single-field filtering (username, tel) |
| Composite | Multiple columns. Equality first, then range, then sort |
| Unique | Business uniqueness (uk_code, uk_username) |
| Covering | All SELECT columns included - avoids table lookup |
| Association | Composite unique (a_id, b_id) |

Left-prefix rule (最左前缀原则): (a, b, c) serves (a), (a, b), (a, b, c). Cannot serve (b), (c), (b, c), (a, c). Range queries (> < LIKE prefix%) break the chain.

Index invalidation: LIKE '%term%' (leading wildcard), function on index (YEAR()), implicit type conversion, OR with unindexed branch, NOT/!=, skip leftmost column.

---

## 5. Transactions

All multi-table writes must use transactions. Never call container.SomeService.Method() inside a transaction. Service MUST NOT directly use GORM Gen query APIs.

Operation logs: current phase uses synchronous writes within the transaction to guarantee data consistency. Only important operations are logged (create, update, delete of core business entities). When throughput becomes a bottleneck, migrate to async via MQ.

```go
err := db.WriteQuery().Transaction(func(tx *query.Query) error {
    // Use Transaction* DAO variants
    // Write operation log within same tx
    return nil
})
```

Read/write separation:

| Method | Usage |
|--------|-------|
| repo.WriteQuery() | All mutations (INSERT, UPDATE, DELETE) |
| repo.ReadQuery() | Read-only queries that tolerate replication lag |
| repo.Query() | Default (resolves to write node) |

DAO method pattern: each DAO provides CreateXxx(ctx, model) for direct use and TransactionCreateXxx(ctx, tx, model) for transaction use. Always use the Transaction* variant inside service transactions.

---

## 6. Caching Strategy

Cache at Service layer, never in DAO. Keys: constant.CacheXXXPrefix + entity ID. Format: module:entity:<id> (colon-separated). Invalidation MUST happen after successful DB commit. Never inside a transaction. Cache set is non-blocking - failure is logged but never propagated to client.

### Key Convention

Cache key constants are defined in `internal/constant/cache_key.go`. Use `constant.CacheXxxPrefix + value` pattern — never construct key strings inline.

| Constant | Key Pattern | TTL | Scope |
|----------|-------------|-----|-------|
| CacheMenuTree | `account:menu_data` | 15 days | Menu tree |
| CacheUserRolePrefix | `account:user_role:<userId>` | 15 days | User-role mapping |
| CacheRolePermsPrefix | `account:role_perms:<roleId>` | 15 days | Role-permission mapping |
| CacheRoleMenuPrefix | `account:role_menu:<roleId>` | 15 days | Role-menu mapping |
| SystemDictPrefix | `system:dict:<code>` | 30 days (1h if empty) | Dict items |

Non-cache Redis keys (same file):

| Constant | Key Pattern | TTL | Scope |
|----------|-------------|-----|-------|
| CacheLoginFailPrefix | `login:fail:<username>` | 3 min | Login failure window |
| CacheRefreshJtiPrefix | `jwt:refresh:jti:<sessionId>` | session | Refresh token JTI tracking |

### Pattern: Read-Through + Write-Behind

READ: try cache first, miss → DB → fill cache (non-blocking). WRITE: transaction → commit → invalidate cache AFTER (never inside transaction). Failure to set cache is logged but never propagated to client.

---

## 7. Interface Availability & Performance

### 7.1 Availability Patterns

Read-heavy, infrequently-changing data (menus, permissions, dicts) — see §6 Caching Strategy for the read-through pattern.

Graceful degradation: Cache down → still query DB. DB down → return cached stale data. Never let a single dependency failure bring down the entire interface.

### 7.2 Idempotency

| Scenario | Strategy |
|----------|----------|
| Create by unique key | Unique index - duplicate returns specific error |
| Duplicate request | Check request_id in Redis before processing |
| Status transitions | Check current state: WHERE status = 'Active' |
| Payment/financial | Distributed lock + unique transaction number |

### 7.3 Distributed Lock

Use xlock.RedisLock for read-modify-write concurrent operations. Callback-based API — unlock is managed internally. Not a substitute for unique indexes.

### 7.4 Async Processing (MQ)

Long-running or non-critical operations decoupled via RabbitMQ: email/SMS notifications, data sync, batch processing.

### 7.5 Retry & Timeout

External calls: wrap with context.WithTimeout. Retry transient errors only (network timeout, connection refused) - max 3 times, exponential backoff. No retry on 4xx, validation errors, unique constraint violations. RabbitMQ has built-in reconnection - no additional retry wrapper.

### 7.6 Interface Checklist

- Read: cache tried first, miss handled, fill non-blocking
- Write: idempotent for duplicates
- Concurrent: distributed lock if read-modify-write
- Timeout: context with timeout for external calls
- Degradation: graceful fallback when dependency is down
- Error isolation: cache/lock failures not propagated as 500

---

## 8. Query Optimization

Avoid N+1 queries. Use JOINs or Preload. Use GORM Gen scopes for reusable dynamic filters. Paginate all list endpoints. Default limit: constant.DefaultLimit (10). SELECT specific columns over SELECT *. EXPLAIN before committing complex queries.

| Concern | Guideline |
|---------|-----------|
| List queries | Always paginated, never unbounded |
| Tree data | Load all rows, build tree in memory. No recursive DB queries |
| Aggregation | COUNT with narrow filters + index; large tables use approximate counts |
| Batch ops | Validate size limits, use bulk insert/update |
| Fuzzy search | Use ES/Meilisearch. No LIKE '%term%' on large tables |
| Export | Stream results, never load full dataset into memory |

### Performance Targets

| Metric | Target |
|--------|--------|
| API P99 read latency | < 200ms |
| API P99 write latency | < 500ms |
| Slow SQL threshold | 2 seconds |
| Cache hit rate (user-role / permission) | > 90% |

---

## 9. Feature Development Workflow

1. Database Design: list query patterns -> design schema (types, comments, indexes) -> decide soft-delete per table -> safety check for existing table changes -> created_at mandatory, updated_at if needed
2. Generate: make gen add -> make gen query
3. DAO: implement internal/dao/{module}/ with direct + transaction methods
4. Service: implement internal/service/{module}/ with business logic, caching, operation logging
5. API: implement internal/api/{module}/ with binding, validation, DTO conversion
6. Routes: register in internal/router/{module}_router.go. Apply JWT + PermissionAuth
7. Permissions: add constants to internal/constant/permission.go
8. DI: wire service into internal/di/container.go
9. Config: add config structs to config/config.go if needed
10. Tests: Service unit tests (mock DAO) + DAO integration tests
11. Docs: update README.md if user-facing behavior changes
12. CI: update .github/workflows/ if infrastructure changes

---

## 10. Code Comments (MANDATORY)

Simple code needs no comments. The following must have comments:

| What | Why |
|------|-----|
| Transaction boundaries | Where Transaction() begins/ends, which tables are involved |
| Cache behavior | What is cached, when invalidated, TTL, why this strategy |
| Complex business logic | Non-obvious rules, hidden constraints, workarounds for specific bugs |
| Index rationale | Why this index was added or not added |
| Soft delete decisions | Why a table uses or does not use soft delete |
| Error handling branches | Why a specific error is handled differently |
| Interface design choices | Why cache-first, why distributed lock, why async |

Comment style: one short line explaining WHY, not WHAT. The code already tells WHAT.
