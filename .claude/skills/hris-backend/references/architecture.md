# Backend architecture

Use this reference when creating a backend module, deciding package placement, reviewing
dependency direction, or changing PostgreSQL, Redis, Nextcloud, export, API, and worker
integration in GSNpeeps.

## Contents

- [Architecture target](#architecture-target)
- [System context](#system-context)
- [Deployment topology](#deployment-topology)
- [Canonical backend structure](#canonical-backend-structure)
- [Dependency rules](#dependency-rules)
- [HTTP request lifecycle](#http-request-lifecycle)
- [Layer responsibilities](#layer-responsibilities)
- [Composition roots](#composition-roots)
- [Transaction and consistency boundaries](#transaction-and-consistency-boundaries)
- [Data architecture](#data-architecture)
- [Redis architecture](#redis-architecture)
- [Nextcloud and file architecture](#nextcloud-and-file-architecture)
- [Background jobs](#background-jobs)
- [Security boundaries](#security-boundaries)
- [Failure handling](#failure-handling)
- [Configuration and secrets](#configuration-and-secrets)
- [Observability](#observability)
- [Testing seams](#testing-seams)
- [Architecture decision gate](#architecture-decision-gate)
- [Review checklist](#review-checklist)

## Architecture target

Build GSNpeeps as a modular monolith with two executable composition roots:

```text
cmd/api    -> REST API and health endpoint
cmd/worker -> scheduled and asynchronous jobs

transport -> application service -> repository/integration port
                                      |
                                      v
                           infrastructure adapter
```

Keep the API and worker in one repository and reuse the same domain and application
packages. Give each executable its own startup, shutdown, health, and runtime lifecycle.
Do not split the current scope into microservices.

Apply these principles:

- Keep business rules in application services, not handlers, SQL, or job runners.
- Keep domain types independent of HTTP, SQL drivers, Redis clients, and WebDAV clients.
- Depend on narrow behavior-oriented interfaces.
- Wire concrete dependencies explicitly in `cmd/api` and `cmd/worker`.
- Avoid mutable package globals, service locators, hidden singletons, and cyclic imports.
- Preserve the approved API contract and database schema as sources of truth.
- Add a new abstraction only when it protects a real boundary or improves testing.

## System context

```text
Browser
   |
   v
Nginx gateway
   |--------------------> React static files
   |
   v
Go API
   |----> PostgreSQL 16       authoritative business data and audit records
   |----> Redis 7             active sessions, rate limits, short-lived cache/locks
   `----> Nextcloud WebDAV    employee files and photos

Go worker
   |----> PostgreSQL 16       scans due work and records results
   |----> Redis 7             distributed locks/deduplication when required
   `----> Nextcloud WebDAV    file cleanup when required
```

Enforce these boundaries:

- Expose the application publicly through Nginx; do not expose database, Redis, or
  Nextcloud credentials to the browser.
- Let the backend mediate file authorization. Return only a contract-approved,
  access-controlled URL or proxy response.
- Treat PostgreSQL as the source of truth. Redis and Nextcloud must not redefine
  employee, role, approval, or attendance state.
- Keep all services on the internal Docker network unless the deployment specification
  explicitly requires a public port.

## Deployment topology

Use one backend codebase and image where practical, but start it with different commands:

```text
backend image
  +-- API process:    long-running HTTP server
  `-- worker process: long-running scheduler/job runner
```

The API process must:

1. Load and validate configuration.
2. Initialize logging and telemetry.
3. Connect to required infrastructure.
4. Construct repositories, integrations, services, middleware, handlers, and router.
5. Start the HTTP server with explicit timeouts.
6. Stop accepting requests on shutdown.
7. Drain in-flight requests before closing dependencies.

The worker process must:

1. Load and validate configuration independently.
2. Initialize logging and telemetry.
3. Connect only to dependencies required by its registered jobs.
4. Register jobs with explicit names, schedules, and timeouts.
5. Acquire a distributed lock before each singleton job when replicas can overlap.
6. Stop scheduling new jobs and let active jobs finish within a shutdown deadline.

Do not start background schedulers inside the API process.

## Canonical backend structure

Use this target structure unless the repository already has an equivalent established
location. Prefer extending an existing convention over creating parallel folder trees.

```text
backend/
|-- cmd/
|   |-- api/
|   |   `-- main.go
|   `-- worker/
|       `-- main.go
|-- internal/
|   |-- config/              environment parsing and startup validation
|   |-- domain/              entities, value objects, enums, domain errors
|   |-- dto/                 HTTP request and response shapes
|   |-- validation/          reusable input validation rules
|   |-- service/             application use cases and transaction coordination
|   |-- repository/          persistence ports, filters, and shared repository errors
|   |-- handler/             HTTP adapters grouped by resource
|   |-- middleware/          auth, RBAC, request ID, logging, recovery, rate limit
|   |-- router/              route registration and middleware composition
|   |-- worker/              job definitions and scheduling adapters
|   |-- platform/
|   |   |-- postgres/        repository implementations and transaction adapter
|   |   |-- redis/           session, rate-limit, cache, lock implementations
|   |   |-- nextcloud/       WebDAV file-store implementation
|   |   |-- export/          CSV/XLSX/PDF implementations when required
|   |   `-- logger/          approved structured logging adapter
|   `-- pkg/                 small internal cross-cutting helpers only
|       `-- response/        standard JSON success/error writer
|-- migrations/              ordered PostgreSQL migrations
|-- seeds/                   deterministic development/reference seeds
|-- tests/                   cross-package integration and end-to-end tests
|-- .env.example
|-- Dockerfile
`-- go.mod
```

Group files by feature inside a layer when the layer grows:

```text
internal/service/employee/
internal/handler/employee/
internal/platform/postgres/employee_repository.go
```

Do not create generic dumping grounds such as `common`, `utils`, or `helpers`. Put code in
the package that owns the concept. Use `internal/pkg` only for small cross-cutting
mechanisms with no business meaning.

## Dependency rules

Use this import direction:

```text
router -> middleware -> handler -> service -> domain
                          |          |
                          |          +-> repository/integration interfaces
                          v
                         dto

platform adapters -> domain + repository/integration interfaces
cmd entrypoints   -> all packages required for explicit wiring
worker jobs       -> service interfaces, not HTTP handlers
```

Follow these allowed dependencies:

| Package | May depend on | Must not depend on |
|---|---|---|
| `domain` | Go standard library | HTTP, SQL, Redis, WebDAV, DTO, handler |
| `dto`, `validation` | domain and standard library when needed | repository implementations |
| `service` | domain, ports, clock/ID abstractions | router, handler, concrete clients |
| `repository` | domain and repository-neutral filter/error types | handler, concrete SQL client |
| `handler` | DTO, validation, service interfaces, response helper | raw SQL, Redis, WebDAV |
| `middleware` | auth/RBAC service interfaces and response helper | feature repositories |
| `platform/*` | its client library, domain, relevant port | handler, router |
| `router` | handlers and middleware | business logic and database details |
| `worker` | services and job infrastructure | HTTP handlers and request DTOs |
| `cmd/*` | concrete packages needed for wiring | business logic of its own |

Reject imports that reverse the dependency direction. If two packages need each other,
move the shared contract toward the consumer or split the responsibility; do not solve the
cycle with globals.

Define interfaces at the boundary that consumes the behavior. Keep them small:

```go
type EmployeeReader interface {
    FindByID(ctx context.Context, id domain.EmployeeID) (domain.Employee, error)
}

type FileStore interface {
    Put(ctx context.Context, objectName string, content io.Reader) (StoredFile, error)
    Delete(ctx context.Context, objectName string) error
}
```

Avoid one repository or service interface containing every operation in the system.

## HTTP request lifecycle

Process requests in this order:

```text
Nginx
  -> request ID
  -> recovery
  -> structured access log
  -> security headers/CORS
  -> rate limit where applicable
  -> authentication
  -> RBAC and row-scope authorization
  -> handler decode + validate
  -> application service
  -> repository/integration adapter
  -> standard response mapper
```

The exact middleware order may vary for public routes, but:

- Generate or preserve a request ID before logging failures.
- Recover panics outside feature handlers and return the standard internal error.
- Authenticate before permission checks.
- Apply rate limiting before expensive work.
- Perform authorization again in the service when it depends on loaded resource data.
- Write one response only; return immediately after an error is written.

## Layer responsibilities

### Domain

- Represent employees, organizational structure, roles, attendance, leave, overtime,
  approvals, notifications, documents, and audit concepts.
- Enforce invariants that are intrinsic to a domain object.
- Use typed constants for stable statuses and actions.
- Avoid JSON, SQL, or framework tags unless an established project convention explicitly
  accepts them; prefer separate DTO and persistence mapping.

### DTO and validation

- Mirror the OpenAPI JSON contract with `snake_case` fields.
- Distinguish omitted fields from explicit zero values in partial updates.
- Reject unknown fields when required by the API convention.
- Normalize only explicitly approved values; do not silently reinterpret user input.
- Convert validated DTOs into service commands before invoking business logic.

### Handler

- Extract path/query/header values, decode the body, invoke validation, and call one use case.
- Translate service results into the standard response envelope.
- Map typed application errors to documented HTTP statuses and error codes.
- Avoid transactions, raw queries, permission policy, and integration orchestration.

### Service

- Implement one business use case per method.
- Enforce role permissions, ownership/row scope, status transitions, and idempotency.
- Own transaction boundaries and cross-repository orchestration.
- Receive actor identity and request metadata explicitly for auditing.
- Depend on `Clock`, ID generator, repository, session, notification, and file-store ports
  instead of constructing concrete clients.

### Repository and platform adapters

- Keep repository methods aligned with use cases, not table-shaped generic CRUD.
- Translate database/client-specific errors into stable repository or integration errors.
- Keep SQL joins and persistence mapping inside the PostgreSQL adapter.
- Use context-aware calls and propagate cancellation.
- Never return database driver rows, Redis client types, or WebDAV client types upward.

## Composition roots

Construct dependencies only in entrypoints or focused wiring functions called by them:

```text
config
  -> infrastructure clients
  -> concrete adapters
  -> application services
  -> middleware and handlers / jobs
  -> router or scheduler
```

Fail startup when a required dependency cannot be configured or reached according to the
deployment readiness policy. Log the component name, never its secret.

Prefer constructor injection:

```go
employeeService := employee.NewService(employeeRepo, auditRepo, txManager, clock)
employeeHandler := employeehttp.NewHandler(employeeService, validator)
```

Do not let constructors perform schema migrations or start goroutines. Make lifecycle
operations explicit.

## Transaction and consistency boundaries

Let the service decide when operations must be atomic. Let the PostgreSQL adapter implement
the transaction mechanism:

```go
type TransactionManager interface {
    WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}
```

Pass a transaction-aware context or unit-of-work abstraction without exposing concrete
transaction types to the service.

Use one database transaction for:

- A business state change and its audit record.
- An approval transition and the database-backed notification/event record it creates.
- Multi-table employee creation or update that must succeed as one unit.
- Compare-and-set transitions that prevent duplicate approval or attendance actions.

Do not hold a database transaction open while making slow external calls when avoidable.
For mixed infrastructure operations, define compensation or durable handoff:

- Nextcloud upload then database insert: delete the uploaded object if the insert fails.
- Database state then notification delivery: commit a durable notification/outbox record,
  then deliver asynchronously and retry.
- Session creation after login: if Redis is required for authentication, fail login clearly
  when the session cannot be established.

Design retried commands to be idempotent. Use unique constraints, idempotency keys,
compare-and-set updates, or processed-event records rather than process memory.

## Data architecture

Use PostgreSQL as the authoritative store for all durable business state.

- Use UUID identifiers according to the approved schema.
- Preserve foreign keys, unique constraints, check constraints, and indexes in migrations.
- Store timestamps consistently and convert presentation timezone at system boundaries.
- Use soft delete only for entities whose contract defines it; do not hide hard-delete
  behavior behind a global repository default.
- Keep audit records append-only.
- Use explicit projections for list/detail responses instead of returning persistence rows.
- Add indexes based on documented filters, joins, ordering, and worker scans.
- Keep migrations backward-compatible during rolling deployment where required.
- Never edit an already-applied migration; add a new migration.

Place cross-table reporting queries in a focused read repository. Do not force complex
dashboard reads through many sequential entity repository calls.

## Redis architecture

Use Redis for ephemeral coordination, not durable business truth.

Approved responsibilities include:

- Active-session verification, with keys shaped like `session:<user_id>` where required by
  the authentication contract.
- Login and sensitive-endpoint rate limiting.
- Short-lived permission/reference caches when invalidation is explicit.
- Distributed worker locks and deduplication keys.

For every key family, define:

- Prefix and identifier format.
- Value schema/version.
- TTL and renewal behavior.
- Owner responsible for deletion or invalidation.
- Behavior when Redis is unavailable.

Avoid unbounded keys and values. Do not cache authorization state without invalidating it
on role, permission, or employee-status changes.

## Nextcloud and file architecture

Access Nextcloud through a `FileStore` port implemented by the WebDAV adapter.

- Upload and download through the backend authorization boundary.
- Enforce the approved maximum file size, MIME allowlist, and filename policy.
- Generate server-controlled object names; do not trust the client filename as a path.
- Store file metadata and the approved locator/URL in PostgreSQL, not file bytes.
- Never persist or return Nextcloud credentials.
- Treat a returned file URL as access-controlled; do not create a permanent public link
  unless the product and security specifications explicitly require it.
- Apply timeouts and bounded retries only to safe/idempotent operations.
- Clean up orphaned uploads using compensation and a scheduled reconciliation job.

Use a stable hierarchy that avoids collisions, for example:

```text
employees/<employee_id>/<document_type>/<generated_id>-<safe_name>
```

Keep the final hierarchy aligned with the approved document type and retention rules.

## Background jobs

Implement these known job families independently:

- Contract-expiry reminder around H-30.
- Approval escalation from supervisor to HR after the defined elapsed time.
- Cleanup of employee photos/files beyond the approved retention threshold.

Give each job:

- Stable name and schedule.
- Explicit timeout and cancellation.
- Distributed lock when concurrent replicas would duplicate work.
- Bounded batch size and deterministic pagination.
- Idempotent processing per record.
- Retry classification for transient and permanent failures.
- Structured summary metrics: scanned, succeeded, skipped, retried, failed.

Use database conditions to claim work atomically. Do not rely only on an in-memory
`processed` flag. Keep one record failure from aborting the whole batch unless atomic batch
behavior is an explicit requirement.

## Security boundaries

Enforce security at multiple layers:

- Let Nginx terminate or forward transport security according to deployment configuration.
- Authenticate tokens in middleware and cross-check the active Redis session.
- Enforce route-level permission in middleware.
- Enforce ownership, organizational scope, and state-dependent authorization in services.
- Keep Top Management read-only except for explicitly documented final approvals.
- Record actor, action, resource, before/after state, request ID, IP, and timestamp for
  required write operations.
- Redact passwords, tokens, cookies, secrets, and sensitive employee fields from logs.
- Hash passwords with the approved adaptive password-hashing algorithm and parameters.
- Use parameterized database operations.
- Validate uploads by content and policy, not extension alone.

Do not use UI visibility as authorization. The backend must reject unauthorized direct API
requests.

## Failure handling

Classify failures before mapping them:

```text
validation/domain conflict -> stable client error
authentication/authorization -> stable security error
not found -> resource-specific not-found error
dependency timeout/unavailable -> retryable or service error
unexpected defect -> internal error + correlated server log
```

- Return only documented error codes and the standard response envelope.
- Preserve the original cause for server logging without leaking it to clients.
- Apply deadlines to PostgreSQL, Redis, and Nextcloud operations.
- Retry only transient errors and only when the operation is idempotent.
- Use bounded exponential backoff with jitter for worker delivery/reconciliation.
- Never turn an unknown infrastructure failure into `not found`.
- Keep `/health` lightweight. Separate liveness from dependency readiness if the deployment
  platform requires both semantics.

## Configuration and secrets

- Read configuration from environment variables or the approved secret mechanism.
- Validate all required values and ranges at startup.
- Keep defaults only for safe local-development behavior.
- Document variables without secrets in `.env.example`.
- Separate API and worker settings where schedules, concurrency, or timeouts differ.
- Parse durations and sizes into typed values once during startup.
- Never read environment variables throughout business code.

## Observability

Emit structured logs with:

- Timestamp, level, service/process, environment, request/job ID.
- Actor ID when available and permitted.
- Route or job name, method, status, duration, and stable error code.
- Dependency name and operation for external failures.

Add metrics for request volume/latency/error rate, database and Redis pool health,
Nextcloud latency/failures, authentication failures, rate limits, and job outcomes.

Propagate the request ID through service, repository, integration, audit, and notification
operations. For workers, generate a run ID and a per-item correlation ID.

Do not log request bodies by default because HR data is sensitive.

## Testing seams

Design the architecture so tests can replace:

- Repositories and transaction manager.
- Clock and ID generator.
- Session store, rate limiter, and distributed lock.
- File store and notification delivery.
- Export renderer.

Use:

- Unit tests for domain and service rules.
- Handler tests for decode, validation, error mapping, and response shape.
- Repository integration tests against PostgreSQL.
- Integration tests for Redis key/TTL behavior.
- Contract tests for the Nextcloud adapter using a controlled server or fake.
- Worker tests for locking, idempotency, batching, retry, and cancellation.
- API integration tests for middleware order, RBAC, and transaction-visible outcomes.

Do not mock internal implementation details. Test through the narrow boundary consumed by
the subject.

## Architecture decision gate

The approved project stack fixes Go, PostgreSQL 16, Redis 7, Nextcloud WebDAV, Docker,
Nginx, and the external API/database contracts. It does not automatically approve a
particular router, SQL abstraction, migration tool, validator, logger, scheduler, test
library, or export library.

Before introducing a dependency not already established:

1. Search the repository, `CLAUDE.md`, active prompt, specs, and lock/module files.
2. Reuse the existing dependency when it satisfies the requirement.
3. If no choice exists, present a short decision with requirements, alternatives, and
   tradeoffs.
4. Obtain approval when the choice affects architecture, contract, security, operations,
   or long-term maintenance.
5. Record the approved choice in the appropriate project guidance.

Do not copy dependency choices from another project merely because its folder structure is
similar.

## Review checklist

Before completing a backend architecture change, verify:

- [ ] The change remains inside the modular monolith boundary.
- [ ] Package imports follow the permitted direction.
- [ ] Domain and service code do not import concrete infrastructure.
- [ ] API and worker lifecycle remain separate.
- [ ] Transaction ownership and external-operation compensation are explicit.
- [ ] PostgreSQL remains the durable source of truth.
- [ ] Redis keys have TTL, invalidation, and failure behavior.
- [ ] File operations preserve backend authorization and do not expose credentials.
- [ ] RBAC includes resource/row scope, not only route permission.
- [ ] Audit and notification writes cannot be silently lost.
- [ ] Timeouts, cancellation, idempotency, and retry behavior are defined.
- [ ] Logs and responses do not leak HR data or secrets.
- [ ] New dependencies passed the architecture decision gate.
- [ ] Relevant unit, integration, contract, and worker tests exist.
