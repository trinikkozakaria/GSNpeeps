# Backend testing

Test behavior at the narrowest reliable level, then prove integrations with disposable
infrastructure. Use Go `testing` with Testify assertions/require and Testify mocks only where
a hand-written fake is not clearer.

## Contents

- Unit and integration tests
- Concurrency and worker tests
- Authorization matrix
- Rules and commands

## Unit tests

Use hand-written mocks/fakes or the approved mocking tool consistently. Unit tests do not access real DB, Redis, network, clock, or filesystem.

Example:

```go
func TestLeaveService_Decide_AlreadyDecided(t *testing.T) {
    repo := &fakeLeaveRepository{
        findForUpdate: func(context.Context, uuid.UUID) (LeaveRequest, error) {
            return LeaveRequest{Status: StatusApproved}, nil
        },
    }
    svc := newTestLeaveService(repo)

    _, err := svc.Decide(context.Background(), supervisorIdentity(), uuid.New(), approveRequest())

    if !errors.Is(err, ErrAlreadyDecided) {
        t.Fatalf("expected ErrAlreadyDecided, got %v", err)
    }
}
```

Prefer table-driven tests for validation, role matrices, transitions, and error mapping.

## Integration tests

Use disposable:

- PostgreSQL 16.
- Redis 7.
- Fake/test WebDAV server.

Run real migrations, build the HTTP composition root, and call it with `httptest` or the approved equivalent. Never use a shared development database.

Cover:

- Migration constraints and rollback.
- Redis login/session invalidation and expiry.
- Employee transaction rollback.
- Multipart upload and orphan cleanup.
- Attendance queries and export media types.
- Audit privileges.
- Notification uniqueness and dismissed retry.

## Concurrency tests

Run competing operations for:

- Failed-login increments to the fifth lock.
- Approval vs approval.
- Approval vs delegation.
- Approval vs auto-escalation.
- Leave-balance deduction.
- Duplicate notification writer.

Assert one valid winner and the documented conflict for others.

## Worker tests

Inject deterministic clock. Run each job repeatedly and concurrently:

- Contract H-30.
- H-30 before/at/after boundary and recipient resolution: active supervisor, every active HR
  except the subject, single Top Management fallback, and missing-fallback failure.
- Auto-escalation.
- Photo retention.
- Notification consumption.

Assert no duplicate effects and safe retry after partial failure.

## Authorization tests

Build a matrix for all 46 operations with:

- Public/unauthenticated.
- Karyawan owner/non-owner.
- Atasan direct-report/non-report.
- HR.
- Top Management read/mutation/HR approval.

Forbidden actions must produce no unintended DB, file, notification, or audit side effect.

## Rules

1. Name tests `TestType_Method_Scenario`.
2. Use synthetic data only.
3. Make tests independent and parallel only when isolation is guaranteed.
4. Use deterministic UUID/clock where assertions depend on them.
5. Do not hide flaky tests with sleeps/retries.
6. Do not claim a numeric coverage target unless the project explicitly sets one.

## Commands

Run the project equivalents of:

```text
go test ./...
go test -race ./...
integration test target
coverage report
OpenAPI lint
migration verification
```

Report tests that require Docker or another unavailable dependency instead of marking them passed.
