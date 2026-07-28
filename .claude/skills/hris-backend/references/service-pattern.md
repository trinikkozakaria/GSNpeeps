# Service/use-case pattern

Services own GSNpeeps business rules, authorization scope, transaction boundaries, audit entries, and domain events. They depend on interfaces, not database/router implementations.

## Contents

- Interface and dependency injection
- Transactional decision example
- Error and transaction rules
- Core GSNpeeps invariants
- Anti-patterns

## Interface example

```go
type LeaveService interface {
    Submit(ctx context.Context, actor Identity, req CreateLeaveRequest) (LeaveRequest, error)
    Decide(ctx context.Context, actor Identity, id uuid.UUID, req DecisionRequest) (LeaveRequest, error)
    Delegate(ctx context.Context, actor Identity, id uuid.UUID, note *string) (LeaveRequest, error)
}
```

## Constructor injection

```go
type leaveService struct {
    tx            TransactionManager
    requests      LeaveRepository
    balances      LeaveBalanceRepository
    organization  OrganizationRepository
    audit         AuditWriter
    events        EventWriter
    clock         Clock
}
```

Do not access globals or instantiate dependencies inside methods.

## Decision example

```go
func (s *leaveService) Decide(
    ctx context.Context,
    actor Identity,
    id uuid.UUID,
    req DecisionRequest,
) (LeaveRequest, error) {
    var result LeaveRequest

    err := s.tx.Within(ctx, func(txCtx context.Context) error {
        current, err := s.requests.FindForUpdate(txCtx, id)
        if err != nil {
            return mapLeaveRepositoryError(err)
        }
        if err := authorizeLeaveDecision(txCtx, actor, current); err != nil {
            return err
        }
        next, err := transitionLeave(current, actor.Role, req)
        if err != nil {
            return err
        }

        updated, err := s.requests.ApplyDecision(txCtx, current, next, actor.UserID)
        if err != nil {
            return mapDecisionConflict(err)
        }
        if next.IsFinalApproval() && current.UsesBalance() {
            if err := s.balances.DeductOnce(txCtx, current); err != nil {
                return err
            }
        }
        if err := s.audit.Append(txCtx, decisionAudit(actor, current, updated)); err != nil {
            return err
        }
        if err := s.events.Append(txCtx, decisionEvent(actor, current, updated)); err != nil {
            return err
        }
        result = updated
        return nil
    })
    if err != nil {
        return LeaveRequest{}, fmt.Errorf("decide leave: %w", err)
    }
    return result, nil
}
```

## Error rules

- Map repository not-found/conflict to application errors.
- Wrap infrastructure errors with operation context.
- Preserve `errors.Is` behavior.
- Never return raw DB/Redis/WebDAV errors to handlers.
- Never use panic for not-found/invalid transition.

## Transaction rules

Use one transaction for related PostgreSQL state:

- Nested employee creation/update.
- Login counter/account lock plus audit where applicable.
- Approval history/status/leave balance/audit/event.
- Permission update/audit.
- Notification insert idempotency.

For Nextcloud/Redis operations that cannot join a DB transaction, define ordering, compensation, and retry behavior.

## Core invariants

- Karyawan with supervisor -> Atasan -> HR.
- Karyawan without supervisor and Atasan -> HR.
- HR -> Top Management.
- Reject requires a note.
- SLA escalation only Atasan -> HR.
- Leave balance deducts once at final approval.
- WFO radius only for WFO.
- Notifications use deterministic event keys.
- Top Management is read-only except HR-request decisions.

## Anti-patterns

```go
// Wrong: service knows HTTP.
return http.StatusForbidden

// Wrong: repository error leaks unchanged.
return s.repo.FindByID(ctx, id)

// Wrong: detached request context.
ctx = context.Background()
```
