# Audit Log

GSNpeeps records authentication, writes, approvals, downloads, permission changes, and system actions in `audit_logs`.

## Contents

- Contract and repository input
- Service transaction usage
- Action/module conventions
- Redaction
- Database protection
- Anti-patterns

## Contract

```text
id         UUID PK
user_id    UUID FK users.id, nullable for system actor
aksi       VARCHAR(30)
modul      VARCHAR(50)
data_id    UUID nullable
detail     JSONB nullable
ip_address VARCHAR(45)
created_at TIMESTAMP/TIMESTAMPTZ
```

Use exact types from the approved migration/Database Schema. The table has no update/delete lifecycle.

## Repository input

Keep the interface dependency-neutral:

```go
type AuditEntry struct {
    UserID    *uuid.UUID
    Action    string
    Module    string
    DataID    *uuid.UUID
    Detail    map[string]any
    IPAddress string
    CreatedAt time.Time
}

type AuditWriter interface {
    Append(ctx context.Context, entry AuditEntry) error
}
```

The implementation performs one INSERT with explicit columns.

## Service usage

Create audit data in the service because it knows the actor, business action, safe before/after values, and transaction.

```go
err := s.tx.Within(ctx, func(txCtx context.Context) error {
    before, err := s.employees.FindByIDForUpdate(txCtx, employeeID)
    if err != nil {
        return err
    }
    after, err := s.employees.Update(txCtx, employeeID, changes)
    if err != nil {
        return err
    }
    return s.audit.Append(txCtx, AuditEntry{
        UserID: &actor.UserID,
        Action: "UPDATE",
        Module: "karyawan",
        DataID: &employeeID,
        Detail: map[string]any{
            "before": sanitizeEmployee(before),
            "after":  sanitizeEmployee(after),
        },
    })
})
```

For critical auditable mutations, keep the audit INSERT in the same PostgreSQL transaction. If a cross-system operation cannot share the transaction, document the failure/retry policy; never silently lose audit evidence.

## Action/module conventions

Actions include:

- `LOGIN`, `LOGOUT`, `CREATE`, `UPDATE`, `DELETE`.
- `APPROVE`, `REJECT`, `DELEGATE`, `AUTO_ESCALATE`.
- `DOWNLOAD`, `PERMISSION_UPDATE`, `ACCOUNT_LOCK`.

Modules use stable API/domain names such as:

- `auth`, `karyawan`, `absensi`, `ketidakhadiran`, `lembur`.
- `permission`, `notifikasi`, `laporan_kehadiran`.

Use `user_id = NULL` for cron/system actions.

## Redaction

Never store:

- Password/password hash.
- JWT, Authorization header, Redis session values.
- Nextcloud credentials or document/photo content.
- Full salary/identity data when IDs and changed field names are sufficient.

Prefer an allowlist sanitizer:

```go
func sanitizeEmployeeChange(e Employee) map[string]any {
    return map[string]any{
        "id":            e.ID,
        "department_id": e.DepartmentID,
        "position_id":   e.PositionID,
        "atasan_id":     e.SupervisorID,
        "status":        e.Status,
    }
}
```

## Database protection

- Grant only INSERT and required SELECT to the runtime DB role.
- Revoke UPDATE and DELETE.
- Do not expose update/delete endpoints.
- Test direct UPDATE/DELETE failure with the application role.

## Anti-patterns

```go
// Wrong: global persistence hook cannot reliably know business actor/action.
func (e *Employee) AfterUpdate(...) error { ... }

// Wrong: serializes password and all PII.
Detail: map[string]any{"after": user}

// Wrong: records success before the business write commits.
audit.Append(ctx, entry)
employees.Update(ctx, employee)
```
