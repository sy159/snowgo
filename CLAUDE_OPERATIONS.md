# CLAUDE_OPERATIONS.md

> Security, testing, observability, Git workflow, deployment, and review checklist for snowgo.
>
> [Back to CLAUDE.md](./CLAUDE.md)

---

## Table of Contents

1. [Security](#1-security)
2. [Testing](#2-testing)
3. [Observability](#3-observability)
4. [Git Workflow](#4-git-workflow)
5. [Prohibited Patterns](#5-prohibited-patterns)
6. [Code Review Checklist](#6-code-review-checklist)
7. [Deployment](#7-deployment)

---

## 1. Security

### 1.1 Authentication

JWT access tokens expire short. Refresh tokens are single-use with JTI tracking in Redis. Login failure protection: 5 failures / 3 minutes per username.

### 1.2 Authorization

Every admin endpoint: middleware.JWTAuth() + middleware.PermissionAuth(constant.PermXXXX). Permission strings in internal/constant/permission.go. RBAC resolves via menu tree.

### 1.3 Sensitive Data

Never log raw passwords, tokens, secrets, PII. Passwords hashed with xcryption.HashPassword() (bcrypt). API responses: no internal error details to clients. Access logs auto-mask sensitive fields.

---

## 2. Testing

### 2.1 Test Types

Unit (pkg/ utilities, testify/assert), Service (business logic, mock DAO), DAO (integration, testify/suite), API (request validation, httptest/e2e).

### 2.2 Test Requirements

Run tests directly related to the change. Expand scope if public modules affected. Cover applicable scenarios: happy path, boundaries, null/default, invalid input, exception branches, permission checks, idempotency, regression. Test names express intent. One test = one behavior. Explicit assertions required - no pseudo-tests. Test failure: identify root cause, distinguish new vs historical, provide fix. Do not mark task complete.

Coverage targets: pkg/ >= 80%. New business modules require Service layer tests. Run go test ./... -cover before committing.

Definition of Done: code implemented + tests added/updated + all pass + docs updated + verification method provided.

---

## 3. Observability

### 3.1 Tracing

Optional OpenTelemetry/Tempo via cfg.Application.EnableTrace. trace_id propagated via X-Trace-Id header, injected into all *Ctx logs.

### 3.2 Metrics

Prometheus metrics exposed. Critical paths instrumented (auth, DB, cache). Alert on: P99 > 500ms, error rate > 1%, cache hit rate < 80%.

### 3.3 Health Checks

/healthz for liveness. /readyz for readiness (MySQL + Redis check). Pprof routes available via config.

---

## 4. Git Workflow

### 4.1 Branch Strategy

| Branch | Purpose |
|--------|---------|
| main | Production-ready. Protected. All merges via PR |
| dev | Integration branch. Features merge here first |
| feature/* | All non-hotfix work. From dev, PR to dev |
| hotfix/* | Urgent production fixes. From main, PR to main + dev |
| release/* | Release stabilization. From dev, PR to main |

Naming: <type>/<kebab-case-desc> (2-4 words). All non-hotfix work uses feature/.

### 4.2 Commit Messages

Conventional commits: <type>(<scope>): <desc>. Types: feat, fix, docs, style, refactor, perf, test, chore, security.

### 4.3 PR Requirements

CI passes (lint, test, security scan). Description references issue/ticket + summary of changes. internal/dal/ changes: confirm make gen was run, no manual edits.

---

## 5. Prohibited Patterns

See CLAUDE_CODING.md and CLAUDE_ARCHITECTURE.md for layer-specific rules. These are cross-cutting prohibitions:

- DO NOT manually edit internal/dal/model/ or internal/dal/query/
- DO NOT use fmt.Printf / log.Println in production code
- DO NOT nest service calls inside transactions
- DO NOT expose internal error details in API responses
- Always add is_deleted = 0 filter for soft-delete tables
- DO NOT commit secrets or .env files
- DO NOT skip tests or claim completion without verification
- DO NOT modify unrelated modules
- DO NOT fabricate test results
- DO NOT modify core infrastructure without approval
- DO NOT perform large-scale refactoring unless explicitly requested

---

## 6. Code Review Checklist

- created_at mandatory, updated_at only if table has updates
- Soft delete per business need; PK type (INT vs BIGINT) matches volume
- DAL generated, not hand-written
- Multi-table mutations use WriteQuery().Transaction()
- Operation log within transaction (sync, consistency guaranteed)
- Admin endpoints: JWTAuth + PermissionAuth
- Input validation at API layer
- Errors use xerror constants; sentinel errors in Service; errors.Is for comparison
- Logs use *Ctx variants
- Cache invalidation after DB commit, not inside transaction
- Sensitive data masked
- Tests pass: make test
- Performance targets met (P99 read < 200ms, write < 500ms)
- Indexes follow left-prefix rule; no invalidation patterns
- Interface: cache-first for reads, idempotent for writes, graceful degradation
- Complex/important code has WHY comments
- README / CLAUDE docs updated

---

## 7. Deployment

### 7.1 Docker

Pinned base image versions. Multi-stage builds mandatory. Never run as root in production.

### 7.2 Configuration

Environment-specific configs (config.dev.yaml, config.uat.yaml, config.prod.yaml). Secrets via environment variables or secret management - never hardcoded. .env in .gitignore.

### 7.3 Monitoring

Production deployments require: health checks in load balancer, log aggregation (ELK or equivalent), metrics (Prometheus + Grafana), alerting on error rate, latency, availability.
