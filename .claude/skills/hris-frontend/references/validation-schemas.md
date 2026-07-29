# Validation schemas and DTOs

Use OpenAPI as the source for wire models and validation. Do not infer a payload from visible
form controls.

Implement runtime schemas with Zod and integrate form validation through React Hook Form.

## Contents

[Model layers](#model-layers) · [Schema organization](#schema-organization) ·
[Required fidelity](#required-fidelity) · [Shared enums](#shared-enums) ·
[Conditional rules](#conditional-rules) ·
[Create versus update](#create-versus-update) ·
[Error messages](#error-messages) · [Mapping](#mapping) ·
[Runtime response validation](#runtime-response-validation) · [Tests](#tests)

## Model layers

Distinguish:

```text
wire model    -> exact OpenAPI snake_case request/response
form model    -> values convenient for editing controls
view model    -> formatted labels/derived presentation
```

Keep one model when no transformation is needed. Add mapping only at a clear boundary.

## Schema organization

Organize shared schemas by resource or feature:

```text
schemas/
|-- common
|-- auth
|-- employee
|-- attendance
|-- leave
|-- overtime
`-- access
```

Feature-local schemas may remain inside the feature. Avoid duplicate enum/schema definitions.

## Required fidelity

Mirror:

- Required versus optional.
- Nullable versus omitted.
- String length and pattern.
- Numeric min/max and integer behavior.
- Date/time format and timezone expectation.
- Enum values and casing.
- Nested objects and arrays.
- Multipart field names and size/type policy.
- Partial update semantics.

Do not convert an optional field into nullable or treat an empty string as omission unless
the contract says so.

## Shared enums

Centralize approved values for:

- `karyawan`, `atasan`, `hr`, `top_management`.
- Employee status.
- WFO, WFH, WFA.
- Leave and overtime request/approval statuses.
- Approval decision/action.
- Notification status/type.
- Export/file formats.

Keep API value separate from Bahasa Indonesia label where they differ.

## Conditional rules

Validate cross-field rules including:

- Position belongs to selected department when enough reference data exists.
- Date ranges are ordered and valid.
- Overtime end is after start and duration obeys contract.
- Perjalanan Dinas destination and purpose are present when selected.
- Required supporting document for Ketidakhadiran.
- WFO location data requirements versus WFH/WFA.
- Rejection note for reject decisions.

Keep authorization and server-time/radius truth on the backend.

## Create versus update

- Create schema includes required initial state.
- Update schema follows OpenAPI partial/full replacement semantics.
- Do not blindly call `.partial()` if immutable fields or at-least-one-field constraints
  differ.
- Ensure omitted fields are not serialized as unintended null/empty values.

## Error messages

- Use Bahasa Indonesia and identify the corrective action.
- Keep consistent wording for required, invalid format, range, and mismatch.
- Do not expose internal property names when a user-facing label exists.
- Preserve backend `error.fields` as final authority.

## Mapping

If mapping case or shape:

- Implement pure recursive/field-specific functions.
- Preserve Date, File, Blob, arrays, null, and non-plain objects.
- Avoid global conversion that corrupts dictionary keys or multipart values.
- Test representative nested payloads.
- Prefer explicit mapping for sensitive or contract-critical models.

Do not double-convert data at both endpoint and hook layers.

## Runtime response validation

If the approved schema tool supports response parsing, prioritize high-risk boundaries:

- Auth response/session identity.
- Employee detail.
- Attendance result.
- Approval state/allowed actions.
- Pagination envelope.

Report contract mismatches as integration errors; do not silently default a missing security
field.

## Tests

- Representative valid OpenAPI fixture passes.
- Every required/nullable/optional boundary behaves correctly.
- Enum casing is exact.
- Conditional rules cover both branches.
- Create/update payloads omit the right fields.
- Backend `error.fields` maps to form paths.
- Wire/form/view round-trip does not lose data.

Do not add a schema library until approved.
