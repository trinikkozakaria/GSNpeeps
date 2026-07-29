# Frontend Auth Session Decision

## Status

Accepted for the Auth/RBAC foundation: **Bearer token in memory only**.

## Decision

The frontend keeps the access token in a module-owned memory reference for the current tab.
It does not write the token to `localStorage`, `sessionStorage`, IndexedDB, a browser cookie,
or a persisted TanStack Query cache.

After a successful login, the frontend:

1. Keeps the token in memory.
2. Calls `GET /api/v1/auth/me` with that token to resolve the authoritative identity.
3. Calculates local expiry from the contract value `expires_in = 28800`.
4. Clears auth state and protected query data on logout, expiry, or `401`.

A browser reload loses the in-memory token and returns the user to `/login`. This is
intentional until the HTTP contract approves a more durable session mechanism.

## Options considered

| Option | Benefit | Risk/cost | Decision |
|---|---|---|---|
| In-memory | Smallest XSS exposure window; no token at rest | Reload requires login again; no multi-tab session restore | Selected |
| `sessionStorage` | Survives reload in one tab | Token remains readable by injected JavaScript and persists at rest for the tab | Rejected for now |
| `localStorage` | Survives reload and browser restart | Long-lived XSS-accessible token and cross-session stale-data risk | Rejected |
| Backend `httpOnly` cookie | JavaScript cannot read the token | Requires CSRF design and an approved API/security-contract change | Not in current contract |

## Security and lifecycle

- Backend JWT and Redis session validation remain authoritative.
- Frontend role checks provide UX only.
- No refresh endpoint, refresh cookie, or silent token replay exists.
- A `401` ends local auth; a `403` preserves it.
- Logout clears the token even if the network request fails.
- No permissions, employee payloads, or passwords are persisted.
- Multi-tab synchronization is intentionally absent because it would require a separately
  approved browser communication and persistence decision.

## Revisit trigger

Revisit this decision only if reload persistence becomes a confirmed requirement. Prefer an
approved backend `httpOnly`, `Secure`, `SameSite` cookie design with CSRF protection over
silently introducing browser storage.

