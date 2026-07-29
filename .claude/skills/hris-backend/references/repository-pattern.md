# Repository pattern

Repositories isolate PostgreSQL access. Define interfaces from service needs and keep implementations in the approved data-access layer.

## Sentinel errors

```go
var (
    ErrNotFound = errors.New("repository: not found")
    ErrConflict = errors.New("repository: conflict")
)
```

## Interface example

```go
type EmployeeRepository interface {
    List(ctx context.Context, page PageQuery, filter EmployeeFilter) (Page[EmployeeSummary], error)
    FindByID(ctx context.Context, id uuid.UUID) (EmployeeDetail, error)
    FindByIDForUpdate(ctx context.Context, id uuid.UUID) (Employee, error)
    Create(ctx context.Context, employee EmployeeAggregate) (uuid.UUID, error)
    Update(ctx context.Context, id uuid.UUID, changes EmployeeChanges) (Employee, error)
    SoftDelete(ctx context.Context, id uuid.UUID, at time.Time) error
    IsDirectReport(ctx context.Context, supervisorID, employeeID uuid.UUID) (bool, error)
}
```

Do not expose a generic CRUD repository that forces services to load or mutate too much.

## Query rules

- Pass `context.Context` first.
- Use explicit selected columns.
- Use parameters for all values.
- Whitelist any dynamic sort column/direction.
- Default employee queries exclude `deleted_at IS NOT NULL` and nonactive rows when appropriate.
- Apply owner/direct-report/recipient scopes inside queries where practical.
- Use stable order plus bounded pagination.
- Avoid N+1 with deliberate joins, aggregation, or batching.
- Use `COUNT(*)` and list queries consistently with identical filters.

Conceptual list query:

```sql
SELECT e.id, e.nip, e.nama, p.nama AS jabatan, d.nama AS departemen, e.status
FROM employees e
LEFT JOIN positions p ON p.id = e.position_id
LEFT JOIN departments d ON d.id = e.department_id
WHERE e.deleted_at IS NULL
  AND ($1 = '' OR e.nama ILIKE $1 OR e.nip ILIKE $1)
  AND ($2::uuid IS NULL OR e.department_id = $2)
  AND ($3 = '' OR e.status = $3)
ORDER BY e.created_at DESC, e.id
LIMIT $4 OFFSET $5;
```

Adapt placeholders to the approved driver.

## Concurrency

Use row locks or conditional updates for:

- Failed-login increment/account lock.
- Leave/overtime decision.
- Delegation vs escalation.
- Leave-balance deduction.
- Notification insert/idempotency.

Check affected rows and return `ErrConflict` for a lost race.

## Transactions

Repositories participate in a transaction supplied by the service transaction manager. Do not start hidden nested transactions.

## Error mapping

- Map driver not-found and unique/FK/check violations to stable repository errors.
- Wrap other infrastructure errors with operation context.
- Never compare raw database error strings.

## Anti-patterns

```go
// Wrong: unscoped sensitive read.
SELECT * FROM employees

// Wrong: injectable sorting.
query += " ORDER BY " + userSort

// Wrong: business approval routing in repository.
if requester.Role == "hr" { status = "menunggu_top_management" }
```
