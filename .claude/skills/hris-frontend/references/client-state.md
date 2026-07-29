# Client state

Use this reference before adding context, reducer, a global store, persistence, or
cross-component UI state.

## State placement

Choose the smallest correct owner:

| State | Preferred owner |
|---|---|
| Input value, dirty/touched, field errors | Form layer |
| Search, page, filter, sort, shareable tab | URL |
| Employee/list/detail/metrics/notifications | Server-state cache |
| Dialog open, hover, temporary selection | Local component |
| Sidebar collapse or cross-layout preference | Context/store if genuinely shared |
| Auth/session metadata | Dedicated approved auth boundary |
| Camera stream/photo preview | Local attendance component |
| Multi-step draft | Local/reducer; global only if navigation requires it |

Move state upward only when multiple consumers need the same source.

## Global-state admission test

Add state to a global store only when all are true:

1. Multiple distant branches require it.
2. URL, server cache, form state, composition, or local state is unsuitable.
3. Lifecycle and reset ownership are defined.
4. Sensitive-data and persistence risks are understood.
5. A global state solution is added only after the architecture extension gate; TanStack
   Query remains server state and React Hook Form remains form state.

Document state shape, actions, selectors, initialization, and reset.

## Store design

- Group actions with the state they mutate.
- Expose focused selectors to avoid unnecessary rerenders.
- Keep derived values computed rather than stored.
- Use explicit initial state and one complete reset action.
- Avoid network calls inside generic store setters.
- Keep feature-specific state inside its feature.
- Avoid a single application-wide store containing every concern.

## Persistence

Do not persist:

- Passwords or tokens without an explicit approved security decision.
- Employee identity, salary, BPJS, NPWP, KTP, or address responses.
- Photo/file blobs or object URLs.
- Approval documents.
- Permissions or role-derived UI state across logout.

Persist only harmless preferences when approved, with a version and migration strategy.

## Auth cleanup

On logout or `401`:

- Reset auth identity/capability state.
- Reset user-scoped UI state.
- Clear protected server-state cache separately.
- Cancel pending operations.
- Release streams/object URLs.
- Remove approved persisted user-scoped keys.

Make reset idempotent so multiple `401` responses cannot corrupt state.

## Multi-tab behavior

If required, coordinate logout/session expiry through the approved browser mechanism.
Broadcast only an event and timestamp, not token or employee data. Test stale tabs and
simultaneous logout.

## Anti-patterns

- Copy every query response into a global store.
- Store filters locally when the URL should preserve them.
- Store derived `isHr`, counts, or formatted labels alongside their sources.
- Put a file object in persistent storage.
- Add a state library solely to open one dialog.
- Read global state imperatively throughout components when a selector/hook exists.
