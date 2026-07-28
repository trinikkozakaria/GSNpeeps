# API mocking

# API and browser mocking

Use the approved network-interception/test mechanism. Do not add a mock library silently.

## Contents

[Source of truth](#source-of-truth) · [Structure](#structure) ·
[Required scenarios](#required-scenarios) · [Factories](#factories) ·
[Stateful behavior](#stateful-behavior) · [Browser APIs](#browser-apis) ·
[Production safety](#production-safety) · [Contract drift](#contract-drift) ·
[Tests](#tests)

## Source of truth

- Mirror method, path, query, payload, envelope, status, and error codes from OpenAPI.
- Return wire-format `snake_case`.
- Keep mock conversion behavior identical to the real API path.
- Update OpenAPI first when a contract changes.

## Structure

Organize by resource:

```text
test/mocks/
|-- handlers/
|   |-- auth
|   |-- employees
|   |-- attendance
|   |-- leave
|   |-- overtime
|   |-- notifications
|   `-- access
|-- factories/
|-- scenarios/
`-- browser-apis/
```

Adapt the location to approved tooling without merging all handlers into one file.

## Required scenarios

For each important operation, model:

- Success.
- Loading/controlled delay.
- Empty and filtered empty.
- `400`, `401`, `403`, `404`, `409`, `422`, `429`, and `500` where documented.
- Offline, timeout, and aborted request.
- Role/row-scope filtering.
- Concurrency conflict for approval.

Use scenario overrides so a component/page test can select one behavior without mutating
unrelated global fixtures.

## Factories

- Generate deterministic synthetic UUIDs, Indonesian-style names, emails, dates, and
  organizational relations.
- Seed pseudo-random data if a generator is approved.
- Support explicit overrides.
- Keep manager/direct-report and approval-stage relationships internally consistent.
- Avoid impossible combinations unless testing corrupt-response handling.
- Never use production employee, salary, identity, photo, or document data.

## Stateful behavior

When useful for local development, keep list/detail/create/update flows consistent in one
session. Reset state between tests. Do not let test ordering determine results.

Model:

- Pagination metadata.
- Soft-deactivated employee behavior.
- Notification unread/read/dismiss coherence.
- Approval decision becoming unavailable.
- Permission changes affecting returned capabilities.

## Browser APIs

Mock deterministically:

- Camera success, denied, unavailable, stopped tracks.
- Geolocation in radius, out of radius, denied, timeout.
- Clock/timezone presentation while keeping server response authoritative.
- File selection, oversize/type rejection, upload failure.
- Blob download and object URL cleanup.
- Online/offline and visibility state for polling.

## Production safety

- Load dev mocks only behind an explicit non-production mode.
- Keep test handlers and factories out of production bundles.
- Fail production build/check if mock mode or mock credentials are enabled.
- Never provide a hidden UI toggle that activates mocks in production.

## Contract drift

Add checks that representative mock responses pass approved response schemas. During real
API integration, treat mismatches as either backend contract violations or stale mocks;
do not patch the component around inconsistent shapes.

## Tests

- Handler path/query behavior matches OpenAPI.
- Role and ownership filtering works.
- Factory output is deterministic and schema-valid.
- Scenario override resets after test.
- Production bundle excludes mock code.
- Browser resources are cleaned after mocked failure.
