# Form and upload pattern

# Form and upload pattern

Use this reference for create/edit forms, login, attendance, approvals, permissions, and
uploads.

## Contents

[Form lifecycle](#form-lifecycle) · [Initialization](#initialization) ·
[Field composition](#field-composition) · [Validation](#validation) ·
[Submission](#submission) · [High-impact actions](#high-impact-actions) ·
[Conditional HR rules](#conditional-hr-rules) · [Upload](#upload) ·
[Camera and geolocation form](#camera-and-geolocation-form) ·
[Unsaved changes](#unsaved-changes) · [Tests](#tests)

## Form lifecycle

```text
initialize defaults
  -> edit: load resource and reset once
  -> user changes fields
  -> client validation
  -> submit once
  -> server validation/business result
  -> success navigation/feedback or recoverable errors
```

Use the approved form and schema libraries. If absent, pass the architecture decision gate.

## Initialization

- Define a default for every rendered field.
- Represent optional, nullable, empty string, boolean, and number deliberately.
- For edit, avoid overwriting user edits when background refetch occurs.
- Reset from server data only during the intentional initialization/reload event.
- Keep create and update models separate.
- Strip unchanged/undefined fields from partial updates according to OpenAPI.

## Field composition

- Use visible labels and field descriptions.
- Associate inline errors programmatically.
- Apply correct input type and autocomplete.
- Preserve keyboard and screen-reader order for conditional fields.
- Use the shared UI primitive through a form adapter.
- Keep schema definitions outside rendered components.

## Validation

Use client validation for immediate UX and backend validation as authority.

- Match required, length, format, enum, range, and conditional rules from OpenAPI.
- Use Bahasa Indonesia messages.
- Do not add a stricter business rule that the contract does not define.
- Map `error.fields` keys to the corresponding nested field.
- Focus the first invalid field after submit when practical.
- Add an error summary for long forms while retaining inline errors.

Unknown/general server errors belong in a form-level alert, not an arbitrary field.

## Submission

1. Validate through the form layer.
2. Disable submit while pending.
3. Keep cancel/back behavior explicit.
4. Prevent Enter/double-click from producing duplicate mutations.
5. Preserve recoverable input on network/server failure.
6. On `409`, explain conflict and refetch when current server state matters.
7. On `422`, apply field errors and summary.
8. On success, invalidate relevant server state before/with navigation.

Do not retry high-impact mutation automatically.

## High-impact actions

Require a clear confirmation for:

- Employee soft-deactivation/delete.
- Approval rejection and delegation.
- Permission changes.
- Dismiss/removal when recovery is limited.

State the resource, action, consequence, and required rejection note. Do not rely on color
alone for destructive meaning.

## Conditional HR rules

- Ketidakhadiran supporting document is required for every approved type.
- Perjalanan Dinas additionally requires destination and assignment purpose.
- Lembur supporting document is optional.
- Rejection requires a note.
- WFO attendance requires location/radius result; WFH/WFA do not inherit the WFO radius
  requirement.

Clear or omit hidden conditional values only according to the contract; do not silently
submit stale hidden data.

## Upload

- Accept only contract-approved document/photo types.
- Enforce 5 MB client-side for early feedback.
- Treat backend MIME/signature/size validation as authoritative.
- Display selected filename, size, replacement, removal, and progress if supported.
- Keep File/Blob/object URL local and short-lived.
- Revoke previews on replacement/unmount.
- Do not upload directly to Nextcloud.
- Preserve non-file form values if upload fails.

## Camera and geolocation form

- Request permission in response to a clear user action.
- Explain why camera/location is needed before the prompt.
- Provide the approved watermarked-photo upload fallback when live camera fails.
- Stop tracks and geolocation watches after capture/cancel/unmount.
- Do not present local time/radius as authoritative.
- Provide specific denied, unavailable, timeout, out-of-radius, and retry states.

## Unsaved changes

Warn before destructive navigation only when meaningful unsaved data exists. Avoid trapping
the user after successful submit or explicit cancel.

## Tests

- Defaults do not cause controlled/uncontrolled transitions.
- Edit initialization does not erase dirty values unexpectedly.
- Client and server field errors render and focus correctly.
- Double submit creates one mutation.
- Conflict preserves input and refreshes authoritative state.
- Conditional fields/values behave correctly.
- File size/type and backend rejection work.
- Camera/geolocation denied fallback cleans resources.
