# Release readiness report

Date: 2026-08-04 (Asia/Jakarta)  
Baseline commit: `386e899748d3abfe38658cd1c3180b1b3b1d14c1`  
Candidate state: local working tree; not committed, tagged, pushed, merged, or deployed  
Decision: **NO-GO pending the open release gates below**

## Scope and contract evidence

- OpenAPI exposes 46 operations. The backend router exposes the same 46 path/method pairs: zero missing and zero undocumented routes.
- Frontend application consumers cover 45 operations; the remaining operation is the service `/health` endpoint. No frontend consumer points to an undocumented operation.
- All 524 local OpenAPI references resolve.
- The production frontend uses same-origin `/api/v1` through Nginx. No development mock, seed password, example account, or E2E password marker was found in the production bundle.
- The live E2E suites use synthetic accounts and read the seed password only from the `E2E_SEED_PASSWORD` process environment.

## Database and seed evidence

A disposable PostgreSQL 16 database was used for integration verification.

- Clean migration from version 0 to version 7 passed.
- Rollback by one migration and forward migration back to version 7 passed.
- The resulting contract contains 26 application tables (27 including Goose metadata).
- The seed ran twice without duplicating its records and produced four roles, five synthetic users (including a Karyawan without a supervisor), exactly one Top Management user, one synthetic office location, and two synthetic leave types for deterministic workflow tests.
- All 32 backend database integration tests passed against the isolated database.

The disposable database was test-only. PostgreSQL created an anonymous data volume automatically; after all integration and race tests passed, the container identity and dedicated host port were verified and both the container and its anonymous volume were removed. The application PostgreSQL volume was not touched.

## Test and build evidence

| Area | Result |
|---|---|
| Frontend unit/component | 168/168 passed across 23 files |
| Browser baseline | 44/44 passed on Chromium desktop and mobile projects |
| Live Docker auth/RBAC | 10/10 passed after the final image rebuild |
| Live account lockout/recovery | 1/1 passed with the seed password restored |
| Isolated HR employee lifecycle | 1/1 passed: create/list/filter/view/update/document upload/list/PDF export/soft-deactivate |
| Isolated attendance workflow | 1/1 passed: WFO radius, sequence/duplicate errors, WFH/WFA, camera-denied fallback |
| Isolated attendance reporting | 1/1 passed: HR/Top feed and report, HR XLSX/PDF export, Top read-only, monthly dashboard boundary |
| Photo retention repeat-run | Passed: first run deleted one photo, later runs deleted zero, attendance row remained |
| Isolated approval/overtime workflow | 1/1 passed: all role routes, reject note, delegation, concurrency, required/optional documents |
| Isolated notification/access/audit | 1/1 passed: exact recipient, row scope, read/dismiss, permission-cache invalidation, audit visibility |
| Protected cache user switch | 1/1 passed: HR data absent after Karyawan switch and no forbidden employee API call |
| Automated accessibility smoke | 5/5 passed: Axe critical/serious, keyboard order, narrow viewport, reduced motion, four role landings |
| Dialog keyboard regression | 1/1 passed: initial focus, forward/reverse focus wrap, Escape cancellation, trigger-focus restoration, and no permission mutation |
| Storage outage | 1/1 passed: Nextcloud failure returns a safe error without internal detail leakage |
| Backend DB integration | 32/32 passed, including escalation and repeat-run idempotency |
| Frontend production build | Passed; 328 modules transformed |
| Backend vet/build/tests | Passed |
| Backend race detector | Passed for all packages, including integration tests |
| Backend vulnerability scan | `govulncheck`: zero reachable vulnerabilities |
| Docker build/startup | Passed with Go 1.26.5 builder; API became healthy |
| Combined health | `status=ok`, `db=ok`, `redis=ok` |
| Backup/restore drill | PostgreSQL counts matched; Nextcloud restore copy matched 26,878 files |
| Graceful shutdown | API and worker exited 0; worker stop logged; API recovered to health 200 |

The live suite proves login/navigation/logout for Karyawan, Atasan, HR, and Top Management. It also proves anonymous access is rejected, Karyawan and Atasan cannot read the employee database, HR and Top Management can read it, Top Management cannot mutate permissions, and logout invalidates the prior token. An authenticated Karyawan navigating directly to the employee route is redirected to the official forbidden page before any employee API request is issued. A separate opt-in scenario proves that the fifth consecutive failure locks the account and revokes its active session; self-reset then unlocks the account, restores the seed password, and verifies login again.

## Security and privacy findings

### Resolved

- Upgraded `golang.org/x/text` to 0.39.0 and `golang.org/x/sync` to 0.21.0, removing the reachable `GO-2026-5970` finding.
- Pinned the backend builder to Go 1.26.5, which includes the standard-library fixes used by the verified image.
- Nginx now replaces, rather than appends to, client-supplied `X-Forwarded-For`. A spoof attempt was verified to record the ingress peer address instead of the supplied address.
- Only Nginx publishes the application port. PostgreSQL, Redis, Nextcloud, API, worker, and frontend remain on Docker networks without host application ports.
- Source and production-bundle scans found no embedded seed password or E2E password variable. Runtime secrets remain environment-provided.
- Employee soft-deactivation no longer refetches detail/document endpoints during the navigation boundary. List, detail, and document caches are invalidated with targeted refetch behavior, preventing a transient expected `404` from being surfaced as a failed request.
- Overtime now accepts the browser-native `HH:MM` time value as well as `HH:MM:SS`, normalizing both to the canonical seconds form before persistence. Unit and real-stack E2E coverage prove the integration fix.
- Audit-log update was rejected by the database append-only trigger and left zero tampered rows. Notification foreign-row operations return the privacy-preserving `403` required by D-032.

### Accepted only after explicit review

`pnpm audit --prod` reports one high advisory, `GHSA-qwww-vcr4-c8h2`, against React Router 7.18.1. The advisory affects unstable React Server Components action paths. This application is a static Vite browser SPA using `createBrowserRouter`; it does not use RSC or server actions, so the vulnerable execution path is not present. The published fix requires the React Router 8.3 major line. This remains an audit finding until an owner either:

1. accepts the documented non-applicability for this candidate, or
2. authorizes and verifies the major-version migration.

## Docker candidate

The active local application services were rebuilt from this working tree. Migration completed successfully; the API, worker, frontend, and Nginx were recreated while PostgreSQL, Redis, and Nextcloud data volumes were preserved. The health endpoint passed after recreation, followed by the final 10/10 live role tests.

State-changing employee, attendance, approval, notification, access, failure-injection, and restore checks ran on a separate Compose project published only on port 18080. It used dedicated PostgreSQL, Redis, and Nextcloud volumes and generated runtime secrets. After the workflows passed, every isolated container, network, and volume was removed. The development database on port 8080 was not used for those mutations.

Redis and PostgreSQL outages each changed combined health from HTTP `200` to `503` and recovered to `200` after restart. A Nextcloud outage returned a sanitized upload failure and recovered after restart. The restore drill matched source/restored PostgreSQL counts (`employees=5`, `notifications=23`, `audit_logs=71`) and copied 26,878 Nextcloud files into a temporary restore volume before deleting all restore targets.

Twenty sequential read probes on the isolated stack measured employee-list latency at 15.2 ms average / 23.7 ms p95 and dashboard latency at 15.6 ms average / 20.4 ms p95. This is a small-dataset sanity measurement, not a production load test. The production build's main application chunk is 32.49 kB gzip and its shared vendor chunk is 149.40 kB gzip.

A larger disposable dataset added 5,000 employees (5,005 total). `EXPLAIN ANALYZE` measured the count query at 0.52 ms, the 100-row joined list at 3.79 ms with a 48 kB top-N sort, and the 5,005-row organization query at 6.06 ms with a 662 kB in-memory sort. A 100-request run at concurrency 10 completed with zero failures: employee list p95 58.4 ms and dashboard p95 130.4 ms. The dashboard run transferred about 15.9 MB because the organization chart intentionally contains the full active hierarchy.

Cancellation was exercised while PostgreSQL held an exclusive table lock. One dashboard query was observed waiting in `pg_stat_activity`; terminating the client removed the waiting query before the lock was released, and health remained `200`. The worker was tested the same way: its escalation query was observed waiting, SIGTERM reduced the waiter count from one to zero, and the worker exited `0` in approximately 416 ms.

An older local API image was started against the restored schema, then cut over through the disposable Nginx ingress while the candidate API was stopped. Rollback health returned `200`; the older image was removed, the candidate was restarted, and candidate health returned `200`. Combined with the earlier successful migration v7 -> v6 -> v7 rehearsal, this covers both image and safe rollback-one recovery locally.

## Open release gates

These checks must not be represented as passed:

- Manual screen-reader review, full critical-workflow keyboard review, and explicit 200% zoom review. Automated Axe, narrow viewport, reduced-motion, login keyboard order, and the permission-dialog focus regression passed. The corresponding live HR dialog smoke is implemented but was not rerun because Docker API access was denied by the execution environment.
- A production capacity target/SLA and load test in the actual deployment environment. Local 5,005-row query-plan and concurrency-10 load sanity passed.
- Production deployment target, TLS termination, and certificate lifecycle.
- Formal disposition of the React Router advisory above.

## Go/no-go decision

Current decision is **NO-GO**. There is no observed authentication bypass, contract mismatch, broken build, reachable backend dependency vulnerability, unhealthy Docker service, failed critical state workflow, uncancelled blocked query, or failed local rollback. The remaining manual accessibility, production capacity/SLA, production TLS/target, and advisory-disposition gates require an owner or deployment environment. A tag, push, merge, release, or deployment requires separate authorization.
