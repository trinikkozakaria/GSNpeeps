# Server state and cache

# Server state and cache

Use the approved server-state library or repository-native mechanism. Do not introduce one
silently.

## Contents

[Query-key design](#query-key-design) · [Query hooks](#query-hooks) ·
[Cache policy](#cache-policy) · [Mutation policy](#mutation-policy) ·
[Retry](#retry) · [Search and navigation](#search-and-navigation) ·
[Notifications and polling](#notifications-and-polling) ·
[Auth lifecycle](#auth-lifecycle) · [Error rendering](#error-rendering) ·
[Tests](#tests)

## Query-key design

Use a centralized hierarchical factory:

```text
["session", user_id]
["employees", "list", user_scope, normalized_filters]
["employees", "detail", user_scope, employee_id]
["dashboard", "metrics", user_scope, normalized_filters]
["notifications", "list", user_id, normalized_filters]
["notifications", "unread-count", user_id]
```

- Include every response-affecting parameter.
- Include authenticated user/scope where cache separation matters.
- Normalize filter objects so equivalent queries share a key.
- Never build keys ad hoc in components.

## Query hooks

Keep one resource concern per hook/module:

- Accept validated query parameters.
- Call an endpoint function from the API boundary.
- Forward cancellation.
- Return transport status without hiding useful error details.
- Apply projections/selectors only for stable derived views.
- Disable the query until required ID, auth, and capability are ready.

Do not toast or redirect inside generic read hooks.

## Cache policy

Choose stale and retention durations from data volatility and sensitivity:

- Reference/master data may remain fresh longer.
- Live attendance and unread notifications require fresher data.
- Approval detail must refetch around decisions.
- Sensitive employee detail should not remain longer than needed.

Do not use infinity/default persistence without explicit justification.

## Mutation policy

After success, invalidate or update all affected surfaces:

| Mutation | Required coherence |
|---|---|
| Employee create/update/delete | list, detail, dashboard metrics as applicable |
| Attendance check-in/out | personal metric, live feed, attendance report |
| Leave/overtime submit | own list, approver list, detail, notifications |
| Approval/delegation | detail, all relevant queues, metrics, notifications |
| Notification read/dismiss | list and unread count |
| Permission update | role/permission data and auth capability resolution |

Prefer authoritative refetch for concurrency-sensitive operations. Use optimistic updates
only when rollback is safe and business conflict is unlikely.

Never optimistically decide approvals, edit permissions, or confirm attendance.

## Retry

Base retry on operation and normalized error:

- Retry a small number of safe reads for transient network/selected `5xx` failures.
- Do not retry `401`, `403`, `404`, `409`, `422`, or `429` blindly.
- Do not retry mutations automatically.
- Stop retries when the request is aborted or the user logs out.
- Use bounded backoff and avoid synchronized polling.

## Search and navigation

- Debounce search input.
- Cancel the previous request when filters change.
- Preserve prior list data during pagination only if the UI labels it correctly and avoids
  stale actions.
- Reset page when a filter changes.
- Prefetch detail only after capability is known and sensitivity permits it.

## Notifications and polling

Poll only if required by the product behavior and no push mechanism exists.

- Pause or slow polling when the document is hidden where appropriate.
- Coordinate unread count and list.
- Refetch on focus only when staleness warrants it.
- Prevent multiple components from starting duplicate polling loops.
- Stop all polling on logout.

## Auth lifecycle

On logout or `401`:

1. Cancel protected requests.
2. Clear all protected/user-scoped queries.
3. Remove persisted cache if an approved persistence layer exists.
4. Reset mutation state containing sensitive payloads.
5. Ensure the next user cannot see stale data.

Do not rely on cache expiry alone.

## Error rendering

Expose normalized errors to the page so it can distinguish forbidden, not found, conflict,
rate limit, offline, and server failure. Never convert every error into an empty list.

## Tests

- Query keys differ for different users/scopes and filters.
- Disabled queries do not fetch before auth/capability/ID.
- Stale search is cancelled.
- Mutation invalidates every dependent surface.
- Approval conflict refetches authoritative state.
- Logout clears cache and polling.
- The next authenticated fixture cannot read previous-user data.
