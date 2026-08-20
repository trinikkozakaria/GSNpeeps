# Object storage and workers

## Object storage

- Browser uploads to Backend API; backend uploads to object storage through the
  `filestore.Store` port (`internal/platform/filestore`), which selects the active adapter
  via `STORAGE_DRIVER`:
  - `minio` (default since 2026-08-20; `internal/platform/minio`) — MinIO, S3-compatible,
    uses a technical access key/secret key pair scoped to one bucket.
  - `nextcloud` (back-compat; `internal/platform/webdav`) — Nextcloud, uses a technical
    WebDAV account.
  See `docs/architecture/minio-integration.md` for the migration decision and action list.
- Validate size, extension, MIME, and signature before storage. This validation happens in
  the handler/service layer and applies identically regardless of the active driver.
- Generate server-owned paths under a stable per-employee/per-document hierarchy and reject
  traversal (`internal/pkg/objectpath.SafePath`, shared by both adapters).
- Store only URL/path in PostgreSQL.
- Use timeouts and clean orphan files when a DB transaction fails.
- Never expose MinIO or Nextcloud credentials to handlers, responses, logs, or frontend.
- Do not add a third storage adapter or an alternate upload path without a recorded product
  decision; both existing adapters must keep satisfying the same narrow interfaces consumed
  by services and handlers (`service.DocumentStore`, `handler`'s `mediaReader`,
  `worker.PhotoDeleter`).

## Workers

- Use the same image/codebase through `cmd/worker`.
- Implement contract H-30, Atasan-to-HR auto-escalation, and three-month photo retention as separate repeat-safe jobs.
- Claim batches safely for multiple workers.
- Use deterministic event keys and database uniqueness for idempotency.
- For contract H-30, notify the active direct supervisor when present plus every active HR
  except the contract subject; when no eligible HR exists, notify the single active Top
  Management user. Deduplicate recipients before insertion.
- Support graceful shutdown, bounded retries, and aggregate logs without PII.
- Never delete an attendance row during photo cleanup; clear `foto_url` after successful file
  handling. This holds for either storage driver — the retention job depends only on
  `worker.PhotoDeleter.Delete`, not on which adapter implements it.
