# PostgreSQL migration pattern

Use `github.com/pressly/goose/v3`, the approved migration tool.

## Contents

- Location and naming
- GSNpeeps schema/dependency rules
- Notification and Audit Log examples
- Index checklist
- Verification and anti-patterns

## Location and naming

- Store SQL migrations in `backend/migrations/`.
- Use the approved chronological/version naming convention.
- Keep one coherent schema change per migration.
- Never edit a migration already applied in another environment; add a new migration.
- Provide a meaningful up and down path when reversal is safe.

Use timestamped SQL files and Goose Up/Down directives.

## GSNpeeps schema rules

- PostgreSQL 16.
- UUID primary keys default to `gen_random_uuid()` from `pgcrypto`.
- Use exact column types from Database Schema v1.1; do not silently substitute timestamp semantics.
- Default FK behavior is `ON DELETE RESTRICT` unless the schema states otherwise.
- Add NOT NULL, UNIQUE, CHECK, generated values, and indexes explicitly.
- Add `created_at`/`updated_at` where specified.
- Implement employee and notification soft-delete fields exactly.
- Do not store file binary data.
- Do not create a refresh-token table.

## Dependency order

Create the 26 contract tables in FK-safe groups:

1. `office_locations`, `departments`, `positions`, `roles`, `employees`, `users`.
2. Ten employee-detail tables.
3. `attendances`, leave tables.
4. Overtime tables.
5. `permissions`, `audit_logs`.
6. `notifications`.

Down migrations reverse this dependency order.

## Example: notification table

The final SQL must match the approved schema; this illustrates critical constraints:

```sql
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    tipe VARCHAR(30) NOT NULL,
    pesan VARCHAR(255) NOT NULL,
    referensi_id UUID,
    referensi_tipe VARCHAR(30),
    event_key VARCHAR(100) NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    dismissed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (recipient_user_id, event_key)
);

CREATE INDEX idx_notifications_recipient
    ON notifications (recipient_user_id, is_read, dismissed_at);
CREATE INDEX idx_notifications_event_key
    ON notifications (event_key);
```

## Audit protection

After creating `audit_logs`, revoke UPDATE and DELETE from the runtime application DB role. Keep only required INSERT/SELECT privileges. Make the role name configurable in deployment/migration execution; never hardcode a production credential.

## Index checklist

Index:

- Foreign keys used in joins/scopes.
- Employee NIP, department, supervisor, and status.
- Contract end date for H-30.
- Attendance `(user_id, tanggal)`.
- Leave/overtime status and requester.
- Audit user/time.
- Notification recipient/unread/dismiss/event.

Do not create duplicate or speculative indexes; review representative query plans.

## Verification

On a disposable database:

1. Migrate from empty to latest.
2. Verify 26 tables and critical constraints.
3. Roll back one migration safely.
4. Migrate up again.
5. Test FK/UNIQUE/CHECK behavior.
6. Verify Audit Log UPDATE/DELETE denial.
7. Run application integration tests.

## Anti-patterns

```sql
-- Wrong: ambiguous FK behavior.
employee_id UUID REFERENCES employees(id)

-- Wrong: drops production data as a routine rollback.
DROP SCHEMA public CASCADE;

-- Wrong: adding any session-refresh table not defined by GSNpeeps.
CREATE TABLE undocumented_session_table (...);
```
