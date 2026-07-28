# Nextcloud and workers

## Nextcloud

- Browser uploads to Backend API; backend uses a technical WebDAV account.
- Validate size, extension, MIME, and signature before storage.
- Generate server-owned paths under a GSNpeeps root and reject traversal.
- Store only URL/path in PostgreSQL.
- Use HTTP timeouts and clean orphan files when a DB transaction fails.
- Never expose WebDAV credentials to handlers, responses, logs, or frontend.

## Workers

- Use the same image/codebase through `cmd/worker`.
- Implement contract H-30, Atasan-to-HR auto-escalation, and three-month photo retention as separate repeat-safe jobs.
- Claim batches safely for multiple workers.
- Use deterministic event keys and database uniqueness for idempotency.
- Support graceful shutdown, bounded retries, and aggregate logs without PII.
- Never delete an attendance row during photo cleanup; clear `foto_url` after successful file handling.

