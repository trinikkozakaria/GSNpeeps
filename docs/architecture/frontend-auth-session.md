# Frontend Auth Session Decision

## Status

Superseded (2026-08-10) by **Bearer token in a browser-written cookie**. The previous
decision was "in memory only"; it was rejected in practice because every page reload ended
the session.

## Decision

The frontend stores the access token in the cookie `gsnpeeps_session`, written by
JavaScript because the API does not send `Set-Cookie` on `POST /auth/login`. The cookie
value is `{"token":"…","expires_at":<epoch ms>}` with `path=/`, `SameSite=Strict`,
`Max-Age` equal to the remaining token lifetime, and `Secure` whenever the page is served
over HTTPS. It cannot be `HttpOnly`, because the same JavaScript must read it to build the
`Authorization` header.

After a successful login, the frontend:

1. Keeps the token in memory and writes the cookie.
2. Calls `GET /api/v1/auth/me` with that token to resolve the authoritative identity.
3. Calculates local expiry from the contract value `expires_in = 28800`.
4. Clears auth state, protected query data, and the cookie on logout, expiry, or `401`.

On reload the provider seeds the in-memory token from the cookie and re-verifies it with
`GET /api/v1/auth/me` before rendering an authenticated shell. A revoked or rejected token
clears the cookie and returns the user to `/login`, so a stale cookie can never grant access
on its own.

## Options considered

| Option | Benefit | Risk/cost | Decision |
|---|---|---|---|
| In-memory | Smallest XSS exposure window; no token at rest | Every reload forces a new login | Superseded |
| JS-written cookie | Survives reload; expiry rides on `Max-Age`; `SameSite=Strict` limits cross-site sends | Readable by injected JavaScript, same as `localStorage` | Selected |
| `localStorage` | Survives reload and browser restart | Same XSS exposure, but no automatic expiry and no `SameSite` semantics | Rejected |
| Backend `httpOnly` cookie | JavaScript cannot read the token | Requires backend `Set-Cookie`, CSRF design, and an API/security-contract change | Preferred target |

## Security and lifecycle

- Backend JWT and Redis session validation remain authoritative; the cookie is a client
  convenience, never proof of authorization.
- The token is XSS-readable. This is the accepted cost of reload persistence until the
  backend issues the cookie itself.
- The cookie expires with the token, so an abandoned tab does not leave a usable token at rest.
- Frontend role checks provide UX only.
- No refresh endpoint, refresh cookie, or silent token replay exists.
- A `401` ends local auth and deletes the cookie; a `403` preserves it.
- Logout clears the token and cookie even if the network request fails.
- No permissions, employee payloads, or passwords are persisted.

## Revisit trigger

Move to a backend-issued `HttpOnly; Secure; SameSite=Strict` cookie with CSRF protection as
soon as the API contract allows it. At that point the frontend drops token handling from
JavaScript entirely and this decision is retired.
