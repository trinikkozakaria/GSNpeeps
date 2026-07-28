---
name: hris-backend
description: Use this skill whenever working on the GSNpeeps Go backend. Trigger it for HTTP endpoints, handlers, services, repositories, PostgreSQL migrations, middleware, JWT and Redis sessions, RBAC and row-level authorization, DTOs, validation, Nextcloud WebDAV, exports, audit logs, scheduled workers, notifications, Docker backend services, or any Go file under backend/. Also use it for requests such as add an endpoint, build a service, fix a handler, create a repository, implement attendance or approval rules, or review backend security. GSNpeeps has strict OpenAPI, role, workflow, audit, storage, and idempotency requirements; consult this skill before editing backend code.
---

# GSNpeeps Backend Skill

Follow this workflow for all backend changes. Do not improvise contracts or introduce an unapproved stack.

## Quick rules

- **Language**: Go using the version approved in the repository.
- **Architecture**: `router -> middleware -> handler -> service/use-case -> repository/integration`.
- **Data**: PostgreSQL 16 with 26 contract tables.
- **Session/rate limit**: Redis 7.
- **Files**: Nextcloud via backend WebDAV; PostgreSQL stores only URL/path.
- **API**: OpenAPI 3.1, 42 operations, JSON `snake_case`.
- **Auth**: JWT 8 hours plus Redis `session:<user_id>`; no refresh endpoint.
- **Roles**: `karyawan`, `atasan`, `hr`, `top_management`.
- **Tool choices**: use only router, data-access, migration, validation, logger, and testing libraries approved in architecture documents.

## Before starting

1. Read `CLAUDE.md`, the active task, and its prompt.
2. Read `docs/openapi.yaml` when available.
3. Read the relevant source document through `../../specs/document-index.md`.
4. Inspect existing patterns and dependencies; never add a parallel implementation.
5. Read every reference listed below that applies to the current change.

## Vertical-slice workflow

When adding or changing a resource:

1. Update OpenAPI first if the approved contract changes.
2. Add a reversible migration.
3. Add or update domain types.
4. Define request/response DTOs and validation.
5. Define repository interfaces from the service's needs and implement them.
6. Implement service rules, transactions, authorization scope, audit, and events.
7. Implement thin handlers and response mapping.
8. Register routes with authentication and authorization.
9. Add unit, integration, concurrency, and negative authorization tests as relevant.
10. Run all applicable quality gates.

Complete one coherent vertical slice before starting another.

## Required reference routing

- `references/architecture.md` — packages, dependency direction, and composition roots.
- `references/handler-pattern.md` — parsing, validation, service calls, and streaming.
- `references/service-pattern.md` — business rules, errors, transactions, and events.
- `references/repository-pattern.md` — query, filters, scopes, and concurrency.
- `references/middleware-pattern.md` — recovery, request ID, auth, RBAC, and rate limit.
- `references/router-pattern.md` — public/protected route registration and 42-operation review.
- `references/response-helper.md` — exact GSNpeeps response/error shape.
- `references/dto-validation.md` — create/update DTOs and field validation.
- `references/migration-pattern.md` — PostgreSQL migration rules.
- `references/audit-log.md` — append-only audit records and redaction.
- `references/nextcloud-worker.md` — WebDAV storage and scheduled jobs.
- `references/testing.md` — unit, integration, concurrency, worker, and security tests.

## Standard response

Success:

```json
{
  "success": true,
  "data": {},
  "message": "OK"
}
```

Paginated response adds:

```json
{
  "meta": {
    "page": 1,
    "limit": 20,
    "total_data": 134,
    "total_page": 7
  }
}
```

Error:

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Data belum valid",
    "fields": {
      "email": "Format email tidak valid"
    }
  }
}
```

Use SCREAMING_SNAKE_CASE codes and Bahasa Indonesia messages. Never expose raw infrastructure errors.

## Domain errors

Use typed/sentinel application errors and map them centrally. Examples:

```go
var (
    ErrInvalidCredentials       = errors.New("invalid credentials")
    ErrAccountLocked            = errors.New("account locked")
    ErrEmployeeNotFound         = errors.New("employee not found")
    ErrConflict                 = errors.New("conflict")
    ErrForbidden                = errors.New("forbidden")
    ErrOutOfRadius              = errors.New("out of radius")
    ErrDuplicateCheckIn         = errors.New("duplicate check-in")
    ErrAlreadyDecided           = errors.New("already decided")
    ErrInsufficientLeaveBalance = errors.New("insufficient leave balance")
)
```

Keep HTTP status knowledge out of services.

## Never

- Never add endpoints, request fields, statuses, or roles missing from approved OpenAPI.
- Never add refresh-token tables, cookies, or endpoints.
- Never return password hashes, token values, session fingerprints, or technical credentials.
- Never use `SELECT *`, unbounded list queries, or concatenated user input in SQL/sort.
- Never authorize only in the frontend or only by role when ownership/team scope is required.
- Never use AutoMigrate in production.
- Never update/delete Audit Log.
- Never hard-delete employee/history/notification records that require soft-delete.
- Never log passwords, JWTs, Authorization headers, files, or unnecessary PII.
- Never use `context.Background()` inside a request service flow.
- Never swallow errors or panic for ordinary control flow.
- Never implement attendance reminders, overtime compensation, or Benefit budget approval.

## Naming

- Package: concise lowercase.
- Exported types/functions: PascalCase.
- Private identifiers: camelCase.
- Sentinel errors: `ErrXxxYyy`.
- Files: `snake_case.go`.
- Database and JSON fields: `snake_case`.
- End-user text: Bahasa Indonesia.
- Code identifiers and commit subjects: English.

## Completion gate

Run the project equivalents of:

```text
format
go mod tidy
go vet ./...
lint
unit tests
integration/concurrency tests
build API and worker
migration clean/up/down-one/re-up
OpenAPI lint
docker compose config
```

Report any check not run. Do not claim completion while required checks fail.

