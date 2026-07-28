# Frontend testing

# Frontend testing

Use the approved test stack. Test behavior and contracts rather than implementation details.

## Contents

[Test pyramid](#test-pyramid) · [Role matrix](#role-matrix) ·
[Error matrix](#error-matrix) ·
[Browser capability tests](#browser-capability-tests) ·
[Accessibility tests](#accessibility-tests) · [Test data](#test-data) ·
[Resource cleanup](#resource-cleanup) · [Quality gate](#quality-gate)

## Test pyramid

### Unit

- Formatters and wire/view mappings.
- Permission/capability helpers.
- Validation schemas and conditional rules.
- Query-key factories.
- Error normalization.
- URL filter parsing/serialization.
- Notification event/view helpers.

### Component

- Shared primitives and form adapters.
- Create/edit forms and server field errors.
- Data table, filters, pagination, and action variants.
- Dialog focus and confirmation.
- Upload, camera, geolocation, and cleanup.
- Notifications and approval timeline/decision controls.

### Integration

- Route guard plus data loader.
- API endpoint/hook/page with OpenAPI-aligned handlers.
- Cache invalidation and conflict refetch.
- Auth expiry cleanup.
- Role navigation and no-sensitive-fetch behavior.

### E2E

- Login, logout, and lockout.
- HR employee CRUD/soft-deactivation and documents.
- Profile and personal metrics.
- WFO/WFH/WFA attendance including failures.
- Every leave/overtime approval route, delegation, and conflict.
- Notification read/dismiss/deep link.
- HR permission administration and Top Management read-only audit view.
- Employee and attendance exports.

## Role matrix

Run representative routes/actions for:

- Karyawan.
- Atasan with direct and unrelated employees.
- HR.
- Top Management including HR-request final approval exception.
- Anonymous user.

Assert forbidden routes do not issue sensitive requests, not merely that controls are hidden.

## Error matrix

Exercise documented:

- Loading and cancellation.
- Empty and filtered empty.
- `400`, `401`, `403`, `404`, `409`, `422`, `429`, and `500`.
- Offline and timeout.
- Malformed contract response for critical boundaries.

Assert `401` clears protected cache while `403` preserves the session.

## Browser capability tests

- Camera allowed, denied, unavailable, and stream cleanup.
- Fallback photo upload.
- Geolocation in/out of WFO radius, denied, unavailable, and timeout.
- WFH/WFA without inherited WFO radius requirement.
- Object URL creation/revocation.
- Polling stop on logout/unmount.
- Reduced motion and narrow viewport.

## Accessibility tests

- Keyboard navigation and visible focus.
- Accessible names, labels, errors, landmarks, and headings.
- Dialog focus trap/return.
- Live-region behavior.
- Status not encoded by color alone.
- 200% zoom and mobile layout.
- Automated scan plus manual critical-flow review.

## Test data

- Use deterministic synthetic data only.
- Preserve valid organization, supervisor, role, and approval relations.
- Never use production names, emails, salary, identity, photos, or documents.
- Reset shared mock state between tests.

## Resource cleanup

After each test, restore:

- Timers and clock.
- Network handlers.
- MediaStream tracks.
- Geolocation watches.
- Event listeners.
- Object URLs.
- Server/client caches and approved persisted state.

## Quality gate

Run repository-defined:

```text
format/check
lint
unit/component tests
integration tests
production build
relevant E2E
accessibility scan
```

Do not claim a command passed if it was unavailable or not run. Report skipped checks and
why.
