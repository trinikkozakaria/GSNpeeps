# Operations runbook

This runbook describes the current Docker Compose candidate. Commands are run from the repository root. Never store real secrets in the repository or command history.

## Required configuration

Provision these values through an access-controlled runtime `.env` or the deployment platform's secret store:

- `POSTGRES_PASSWORD`
- `JWT_SECRET`
- `NEXTCLOUD_ADMIN_PASSWORD`

Set these non-secret or environment-specific values as needed:

- `APP_PORT` (default `8080`)
- `APP_ENV`, `LOG_LEVEL`
- `CORS_ALLOWED_ORIGIN`
- `POSTGRES_DB`, `POSTGRES_USER`
- `NEXTCLOUD_ADMIN_USER`
- `SEED_PASSWORD` only for explicitly approved synthetic/local seeding; leave it empty in production

Restrict secret-file permissions, rotate any value exposed to logs or terminals, and use a high-entropy independent value for each secret.

## Startup and verification

Validate interpolation before changing services:

```powershell
docker compose config --quiet
```

Build and start the candidate:

```powershell
docker compose up -d --build
docker compose ps
Invoke-RestMethod http://127.0.0.1:8080/health
```

Expected health data reports `status`, `db`, and `redis` as `ok`. Compose waits for PostgreSQL, runs migrations, waits for Redis and the API health check, and then starts Nginx. Only Nginx should publish a host port.

Do not run the seed profile in production. For an approved empty local environment only:

```powershell
docker compose --profile tools run --rm seed
```

## TLS

The repository's local Compose stack serves HTTP. Terminate TLS at the approved deployment ingress or reverse proxy. Define the hostname, DNS ownership, certificate issuer, renewal mechanism, HTTP-to-HTTPS redirect, and trusted-proxy boundary before production. Do not request a live certificate until the deployment target is authorized.

## Logs and minimum monitoring

```powershell
docker compose logs --since 15m backend-api
docker compose logs --since 15m cron-worker
docker compose logs --since 15m nginx
docker compose ps
```

At minimum alert on repeated failed health checks, API 5xx rate, authentication rate-limit spikes, migration failure, worker restarts/failures, PostgreSQL/Redis unavailability, Nextcloud errors, and disk/volume capacity. Application logs go to container stdout/stderr; centralize them in the deployment platform and restrict access because audit metadata may contain personal identifiers.

## Failure handling

### Health failure

1. Check `docker compose ps` and recent API logs.
2. Confirm PostgreSQL and Redis health without printing credentials.
3. Confirm the latest migration container exited successfully.
4. Keep Nginx out of rotation until the combined health response is healthy.

### Worker failure

1. Inspect worker logs and restart count.
2. Resolve the dependency or configuration failure before restarting only the worker.
3. Verify lock and idempotency behavior by monitoring duplicate-event constraints and job results.
4. Do not manually replay a job unless its idempotency key and intended recipients are understood.

### Nextcloud outage

Employee metadata in PostgreSQL and file content in Nextcloud form one logical record. During outage, block or fail file operations explicitly; do not mark an upload successful without confirmed storage completion. After recovery, verify file metadata and WebDAV content together.

### Redis loss or restart

Redis loss removes ephemeral cache/rate-limit state. PostgreSQL-backed session revocation remains authoritative, but rate-limit counters may restart. Verify login/logout behavior after Redis recovery and watch for a temporary authentication spike.

## Consistent backup and restore

Back up PostgreSQL and the Nextcloud data/config/database set within one maintenance window. Record timestamps and checksums. Prevent file mutations during the consistency window or use storage/database snapshots that share a recovery point. A PostgreSQL dump alone is not a complete employee-document backup.

Restore into an isolated environment first:

1. Restore PostgreSQL to the recorded recovery point.
2. Restore the matching Nextcloud data, configuration, and its own database.
3. Start dependencies without public ingress.
4. Run migrations only after confirming the restored schema version.
5. Verify health, synthetic login, employee metadata-to-file links, and representative downloads.
6. Open ingress only after application and storage consistency checks pass.

The commands and credentials depend on the deployment backup system; document the exact tested procedure and recovery time before changing the release decision to GO.

### Local restore-drill evidence (2026-08-03)

The disposable Compose project was quiesced for storage copying. `pg_dump -Fc` was restored into a separately named database and representative table counts matched the source (`employees=5`, `notifications=23`, `audit_logs=71`). The Nextcloud volume was copied read-only into a separately named restore volume and both sides contained 26,878 files. The temporary database and volume were then removed, Nextcloud restarted, and application health recovered. This proves the local procedure, but it does not replace a timed restore test using the production backup platform and its encryption, retention, and access controls.

## Application rollback

Record the previous known-good image digest and commit before rollout. If rollback is needed:

1. Remove the candidate from ingress.
2. Stop state-changing workers.
3. Decide migration rollback versus forward-fix per migration. Never assume an older image can safely use a newer schema.
4. If a safe Goose down migration was rehearsed, run exactly one approved step and verify its data impact. Otherwise restore the consistent PostgreSQL/Nextcloud backup or deploy a forward-fix.
5. Redeploy the recorded image digests, not mutable tags.
6. Restart Redis when forced session/cache invalidation is required.
7. Verify combined health, four-role authentication/authorization, worker status, and file access before restoring ingress.

Container rollback does not reverse database writes or Nextcloud file changes. Treat those as explicit compensation or restore work.

### Local rollback-drill evidence (2026-08-03)

The disposable ingress was cut over from the candidate API to a retained pre-candidate image while PostgreSQL remained on the tested compatible schema. Health through Nginx returned `200`. The pre-candidate container was then removed, the candidate API restarted, and health again returned `200`. A separate migration rehearsal successfully moved schema v7 down to v6 and forward to v7. Production rollout must still use immutable registry digests and must repeat the drill using the actual ingress and backup platform.

## Credential rotation

Rotate one dependency at a time with an approved maintenance plan. Update the secret store, recreate only the affected services, verify health and authentication, then revoke the previous credential. Rotating `JWT_SECRET` invalidates all signed access tokens and requires a planned user re-login. PostgreSQL and Nextcloud rotations must update both the dependency account and application secret atomically.
