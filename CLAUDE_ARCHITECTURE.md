# CLAUDE_ARCHITECTURE.md

> **Version**: 1.3.0
> **Last Updated**: 2026-04-26
> Architecture, database design, DAL generation, transactions, caching, and feature development workflow for `snowgo`.
>
> [Back to CLAUDE.md](./CLAUDE.md)

---

## Table of Contents

1. [Layered Architecture](#1-layered-architecture)
2. [Dependency Injection](#2-dependency-injection)
3. [DAL Code Generation (MANDATORY)](#3-dal-code-generation-mandatory)
4. [Database Design & Transaction Rules](#4-database-design--transaction-rules)
5. [Query Optimization & Performance](#5-query-optimization--performance)
6. [Caching Strategy](#6-caching-strategy)
7. [Feature Development Workflow](#7-feature-development-workflow)
8. [Code Quality Requirements](#8-code-quality-requirements)

---

## 1. Layered Architecture

```
Router -> API -> Service -> DAO -> DAL (GORM Gen) -> MySQL / Redis
```

| Layer | Responsibility | Rules |
|-------|---------------|-------|
| **Router** | HTTP routing, middleware mounting | No business logic. No direct DB calls. |
| **API** | Request validation, response formatting, calling Service | Input binding, DTO conversion. No transactions. |
| **Service** | Business orchestration, caching, transaction coordination | ALL business rules. Coordinates DAO calls within transactions. |
| **DAO** | Data access abstraction | Wraps GORM Gen queries. Direct + transaction-aware methods. No business logic. |
| **DAL** | Auto-generated model + query code | **Never hand-edit.** Generated via `make gen`. |

---

## 2. Dependency Injection

- All infrastructure (DB, Redis, JWT, Cache, Lock, MQ) lives in `internal/di/container.go`.
- Services are constructed in `NewContainer(...)` using the **Option pattern**.
- Resources registered with `CloseManager` shut down in **LIFO** order.
- Access container in handlers: `di.GetContainer(c)` or `di.GetSystemContainer(c)`.

---

## 3. DAL Code Generation (MANDATORY)

**`internal/dal/model/` and `internal/dal/query/` are machine-generated. Never edit manually.**

Workflow:
1. Design table schema in MySQL.
2. Run `make gen add` (new tables) or `make gen update` (schema changes).
3. Run `make gen query` to regenerate query APIs.
4. `internal/dal/query_model.go` is auto-updated.

> **WARNING**: Manual edits to generated files are overwritten by the next `make gen` and will break production.

---

## 4. Database Design & Transaction Rules (MANDATORY)

### 4.1 Schema Design

> **Rule of thumb**: Design tables for the query patterns, not the entity model.

#### Table Naming
- Business tables: `t_<module>_<entity>` or `<module>_<entity>` (be consistent).
- Association tables: `<entity_a>_<entity_b>` (e.g., `user_role`).
- All tables must have `created_at` and `updated_at`.

#### Field Design
- Use explicit `NOT NULL` with sensible defaults. Avoid nullable columns unless truly optional.
- Use appropriate types: `BIGINT` for IDs (recommended for new tables; current tables may use `INT`); `VARCHAR(n)` for strings (pick `n` carefully, not all `255`), `DECIMAL` for money, `JSON` for semi-structured data.
- Store enums as `VARCHAR` with application-level validation; document allowed values in column comments.
- **Foreign keys**: Optional. Use them for data integrity in core tables; skip them in high-write or sharded tables.

#### Index Design

> **Rule of thumb**: Small tables (predicted < 1,000 rows, e.g., config tables like `menu`, `dict`, `role`) usually do NOT need extra secondary indexes. The optimizer will full-scan anyway; extra indexes only increase write overhead and storage.

- **Large / high-growth tables**: every query must have an index path.
- **Single-column indexes**: high-cardinality filtering columns (e.g., `username`, `code`).
- **Composite indexes**: follow the left-prefix rule — equality filters first, then range filters, then ordering columns.
  - Example: `WHERE status = ? AND created_at > ? ORDER BY created_at` → index `(status, created_at)`.
- **Covering indexes**: for high-frequency lookups to avoid table lookups.
- **Unique indexes**: for business uniqueness constraints (e.g., `uk_code`, `uk_username`).
- **Association tables**: composite unique index on `(a_id, b_id)` to prevent duplicates.
- Document the rationale for each index (or the explicit decision to omit one) in migration comments.

#### Soft Delete Strategy

- **Soft delete is a business decision, not a mandate.** Use it when:
  - The data has audit/compliance requirements.
  - Deletion would break referential integrity in downstream systems.
  - Users expect "undo" or trash-can behavior.
- **Do NOT use soft delete** when:
  - The table is a high-volume log/audit table (use hard delete + archival).
  - The table is a many-to-many junction table with no standalone business meaning.
  - Storage cost is a concern and data loss is acceptable.
- If soft delete is used:
  - Add `is_deleted tinyint(1) DEFAULT 0`.
  - Add `INDEX idx_is_deleted` only if querying mixed deleted/undeleted rows.
  - All queries must filter `is_deleted = false` unless explicitly querying deleted records.
  - Use `UpdateSimple` for soft deletes; do not rely on GORM's `Delete` hook.

### 4.2 Read/Write Separation

| Method | Usage |
|--------|-------|
| `repo.Query()` | Default (respects resolver hint) |
| `repo.WriteQuery()` | **Mandatory for all mutations** (INSERT, UPDATE, DELETE) |
| `repo.ReadQuery()` | Read-only queries that tolerate replication lag |
| `repo.ChangeDB(dbName)` | Switch connections for multi-DB setups |

### 4.3 Transactions

- **All write operations involving multiple tables must be wrapped in a transaction.**
- Pattern:
  ```go
  err := db.WriteQuery().Transaction(func(tx *query.Query) error {
      // Call DAO transaction methods, passing tx
      // Call operation log within same tx
      return nil
  })
  ```
- **Never** call `container.SomeService.Method()` inside a transaction — services manage transactions, they don't nest.
- Operation logs (`system.OperationLogService.CreateOperationLog`) must be written **within** the same transaction as the business mutation.
- **Service MUST NOT directly use GORM Gen query APIs.** All DB operations MUST go through DAO methods.

---

## 5. Query Optimization & Performance (MANDATORY)

### 5.1 Query Design Rules

- **Avoid N+1 queries.** Use JOINs or `Preload` when fetching associations.
- **Use GORM Gen scopes** for reusable dynamic filters (e.g., `UserNameScope`, `StatusScope`).
- **Paginate all list endpoints.** Default limit is `constant.DefaultLimit` (10). Enforce `MaxLimit` if applicable.
- **Prefer `SELECT` specific columns** over `SELECT *` in list queries.
- **Use `EXPLAIN` before committing** complex queries or new indexes.
- **Denormalize cautiously.** Only when read performance is critical and consistency can be managed (e.g., cached counters).

### 5.2 API Performance Design

| Concern | Guideline |
|---------|-----------|
| List queries | Always paginated; never return unbounded result sets. |
| Tree / hierarchical data | Load all rows and build the tree in memory; avoid recursive DB queries. |
| Aggregation / counts | Use `COUNT(*)` with narrow filters + index; for large tables, use approximate counts or materialized views. |
| Batch operations | Accept batch input, validate size limits, use bulk insert/update where supported. |
| Search / full-text | Use dedicated search (Elasticsearch/Meilisearch) for fuzzy search; do not use `LIKE '%term%'` on large tables. |
| Export | Stream results; do not load entire dataset into memory. |

### 5.3 Database Index Checklist

Before a feature is considered complete, verify:
- [ ] **Table size assessed**: small tables (<1K rows) confirmed no extra indexes needed; large / high-growth tables have index coverage.
- [ ] All `WHERE` filters on large tables are indexed.
- [ ] All `ORDER BY` columns on large tables are indexed (or covered by a composite index).
- [ ] Association tables have composite unique indexes.
- [ ] Index selectivity is high enough (avoid indexing low-cardinality columns alone on large tables).
- [ ] Migration includes index rationale (or explicit decision to omit) in comments.

### Performance Targets (Production)

| Metric | Target |
|--------|--------|
| API P99 read latency | < 200ms |
| API P99 write latency | < 500ms |
| Slow SQL threshold | 2 seconds (`SlowSqlThresholdTime`) |
| Cache hit rate (user-role / permission) | > 90% |

> Any query exceeding the slow SQL threshold must be optimized and documented.

---

## 6. Caching Strategy

- Cache at **Service layer**, never in DAO.
- Cache keys: use `constant.CacheXXXPrefix` + entity ID.
- **Cache invalidation**: always invalidate on mutation (update/delete).
- **Cache invalidation MUST happen immediately after successful DB commit.** Do NOT invalidate inside a transaction.
- Prefer caching read-heavy, infrequently-changing data (user-role mappings, permission trees).
- Set explicit TTLs. Default user-role cache: `CacheUserRoleExpirationDay` days.

---

## 7. Feature Development Workflow

When adding a new module (e.g., `order`, `inventory`):

1. **Database Design**:
   - List expected query patterns (filter, sort, join).
   - Design schema with proper types, comments, and **indexes for large/high-growth tables** (small config tables may skip secondary indexes).
   - Decide soft-delete strategy per table with documented rationale.
   - **Database change safety check** (if modifying existing tables):
     - Evaluate whether the change is truly necessary.
     - Define migration strategy (additive-only preferred; avoid destructive changes).
     - Assess compatibility impact on existing data and running code.
     - Assess index impact (new indexes on large tables may lock tables).
     - Plan historical data compatibility (defaults, backfills, or data migration).
     - Provide minimal validation approach (rollback plan if applicable).
   - Write migration (or update schema and run `make gen add`).
   - Ensure `created_at`, `updated_at` on all tables.
2. **Generate**: `make gen add` -> `make gen query`.
3. **DAO**: Implement `internal/dao/{module}/` with direct + transaction methods.
4. **Service**: Implement `internal/service/{module}/` with business logic, caching, operation logging.
5. **API**: Implement `internal/api/{module}/` with binding, validation, DTO conversion.
6. **Routes**: Register in `internal/router/{module}_router.go`. Apply JWT + PermissionAuth.
7. **Permissions**: Add constants to `internal/constant/permission.go`.
8. **DI**: Wire service into `internal/di/container.go`.
9. **Config**: Add config structs to `config/config.go` if new infrastructure is needed.
10. **Tests**: Write unit tests for Service (mock DAO) and DAO (integration with test DB).
11. **Docs**: Update README.md and API documentation if user-facing behavior changes.
12. **CI**: Update `.github/workflows/` if infrastructure or build process changes.

---

## 8. Code Quality Requirements (MANDATORY)

All code written or modified with AI assistance must include clear comments explaining:
- Business logic intent.
- Transaction boundaries (where `Transaction()` begins and ends).
- Cache behavior (what is cached, when it is invalidated, TTL if applicable).
- Performance considerations (why this query pattern, index usage).

Do not generate placeholder, TODO, or incomplete code. Follow existing naming and file structure strictly. Prefer modifying existing files over creating new ones unless a new module is required.
