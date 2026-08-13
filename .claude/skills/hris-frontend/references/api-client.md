# API client

Use this reference when adding an endpoint function, transport client, upload, export,
download, authentication header, error mapper, or network retry.

## Contents

[Responsibilities](#responsibilities) · [Contract mapping](#contract-mapping) ·
[Request behavior](#request-behavior) · [Response behavior](#response-behavior) ·
[Status policy](#status-policy) · [Upload](#upload) ·
[Download and export](#download-and-export) ·
[Retry and cancellation](#retry-and-cancellation) ·
[Endpoint-module rules](#endpoint-module-rules) · [Checklist](#checklist)

## Responsibilities

Keep one transport boundary responsible for:

- One shared Axios instance and interceptors.

- Validated API base URL.
- JSON serialization and response parsing.
- Bearer-token injection.
- Standard response-envelope handling.
- Timeout and abort propagation.
- Stable error normalization.
- Multipart upload and authenticated file download.
- Safe request metadata such as request/correlation ID when supported.

Keep one shared transport and place resource endpoint functions in their owning modules:

```text
src/lib/api/
|-- client
|-- errors
|-- envelope
|-- auth-header
|-- upload
`-- download

src/modules/employees/api/
src/modules/attendance/api/
src/modules/leave/api/
src/modules/overtime/api/
src/modules/notifications/api/
src/modules/access/api/
```

Use the approved extension and module convention. Do not duplicate transport setup per
module. Do not place resource statuses, permission groupings, view helpers, or a generic
`utils` dumping ground in `src/lib/api`.

## Contract mapping

Mirror the standard success response:

```json
{
  "success": true,
  "data": {},
  "message": "Operasi berhasil"
}
```

List endpoints may include:

```json
{
  "meta": {
    "page": 1,
    "limit": 20,
    "total_data": 134,
    "total_page": 7
  }
}
```

Normalize the standard error:

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Data belum valid",
    "fields": {
      "email": "Format email tidak valid"
    }
  }
}
```

Represent frontend errors with stable fields:

```text
status
code
message
fields
cause (internal/logging only)
```

Distinguish HTTP errors, network/offline failures, timeout/abort, invalid response shape,
and configuration failures. Do not collapse all failures into "server error".

## Request behavior

- Read the base URL from validated public configuration; never hardcode an environment URL.
- Inject the current Bearer token at one boundary.
- Set JSON content headers only for JSON bodies; let the runtime construct multipart
  boundaries.
- Preserve `AbortSignal` or the approved equivalent from caller to transport.
- Omit optional query parameters only according to the OpenAPI contract; do not remove
  meaningful `false`, `0`, or empty values blindly.
- Encode path and query values safely.
- Never log Authorization headers, passwords, document bodies, photo blobs, or sensitive
  employee payloads.

The approved contract has no refresh endpoint. Do not add automatic refresh or replay a
failed mutation after `401`.

## Response behavior

- Verify the envelope before returning `data`.
- Return `meta` intact for list hooks.
- Preserve `message` when useful for mutation feedback.
- Treat an HTTP success with `success: false` as a contract/error condition.
- Treat malformed JSON or envelope mismatch as an integration error with correlated logs.
- Keep wire fields in `snake_case`; map to a view model explicitly when required.

Do not perform toast, redirect, or cache invalidation inside endpoint functions.

## Status policy

| Failure | Client behavior |
|---|---|
| `400` | Show request/parameter error appropriate to the page |
| `401` | End local session, clear protected cache, redirect safely to login |
| `403` | Preserve session and render forbidden/no-action state |
| `404` | Render resource-specific not-found state |
| `409` | Explain conflict and refetch concurrency-sensitive state |
| `422` | Map `error.fields` to form controls and show summary |
| `429` | Show lockout/rate-limit feedback and any approved retry timing |
| `5xx` | Show retryable service error without leaking internals |
| offline/timeout | Show connection-specific recovery |

## Upload

- Validate extension, MIME, and 5 MB limit for early UX feedback.
- Let the backend perform authoritative file validation.
- Use the exact multipart field names from OpenAPI.
- Surface upload progress only if the approved transport supports it reliably.
- Allow cancellation where feasible.
- Keep files and object URLs out of persistent/global state.
- Clear file input after success or explicit reset.

Do not upload directly to Nextcloud or expose its credentials.

## Download and export

- Request protected exports through the authenticated API boundary.
- Read filename from a safe response header when defined; otherwise generate a
  product-appropriate fallback.
- Validate expected content type when practical.
- Create a short-lived object URL, trigger download, then revoke it.
- Handle authorization and expired-session errors before treating the body as a file.
- Do not cache salary/identity exports in persistent browser storage.

## Retry and cancellation

Retry only safe read requests for transient network/selected server failures. Use a small
bounded count and backoff. Never automatically retry:

- Login/logout.
- Create/update/delete.
- Attendance check-in/out.
- Approval decision/delegation.
- Permission changes.
- Uploads unless the protocol and idempotency guarantee make it safe.
- `401`, `403`, `409`, `422`, or business-rule errors.

Silently ignore an abort caused by navigation/unmount; surface a real timeout or connection
failure.

## Endpoint-module rules

- Export one function per OpenAPI operation.
- Use resource language from the API path while keeping code identifiers in English.
- Accept a command/query object rather than UI event objects.
- Keep endpoint functions deterministic and independent of rendered components.
- Add contract tests or fixtures for success and documented errors.
- Update OpenAPI first when the contract must change.

## Checklist

- [ ] Method, path, query, payload, response, and status match OpenAPI.
- [ ] Token injection and redaction are centralized.
- [ ] Cancellation reaches the network call.
- [ ] `401` does not attempt refresh.
- [ ] Field errors remain available to the form.
- [ ] Upload/download cleanup is implemented.
- [ ] Retry policy is safe for the operation.
- [ ] Endpoint code has no toast/navigation/UI side effect.
