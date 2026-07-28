# Authentication and RBAC UX

Use this reference for login/logout, auth bootstrap, route guards, navigation, conditional
actions, session expiry, and role-aware UI.

## Contents

[Session contract](#session-contract) · [Bootstrap](#bootstrap) ·
[Route protection](#route-protection) ·
[Centralized capabilities](#centralized-capabilities) · [Role UX](#role-ux) ·
[Action rendering](#action-rendering) · [Error behavior](#error-behavior) ·
[Logout](#logout) · [Testing matrix](#testing-matrix) ·
[Anti-patterns](#anti-patterns)

## Session contract

- GSNpeeps provides login and logout only.
- JWT lifetime is eight hours.
- The backend cross-checks the active Redis session.
- There is no approved refresh endpoint or refresh-token flow.
- Follow the approved token-storage decision; do not choose persistence silently.

Model auth lifecycle explicitly:

```text
unknown -> authenticated
unknown -> anonymous
authenticated -> expired/invalid -> anonymous
authenticated -> logout -> anonymous
```

Do not render protected content while state is `unknown`.

## Bootstrap

At application start:

1. Resolve the token/session using the approved storage strategy.
2. Validate identity through an approved endpoint if the contract provides one.
3. Resolve current role/capabilities before mounting protected routes.
4. Clear invalid session data and protected cache.
5. Render the protected shell only after resolution.

If the API contract lacks a current-user/bootstrap operation, record the gap and ask for a
contract decision. Do not invent `/auth/me` or `/auth/refresh`.

## Route protection

Use two UX layers:

1. A route-level auth/capability guard before route data loaders execute.
2. Component-level action visibility/disabled behavior for allowed pages.

The guard must:

- Preserve a validated internal return path for post-login navigation.
- Reject unsafe external/open-redirect targets.
- Avoid sensitive requests for a forbidden route.
- Show a stable loading state during auth initialization.
- Render a dedicated forbidden state for authenticated users without access.

Frontend checks never replace backend authorization.

## Centralized capabilities

Define navigation and page/action rules once:

```text
capability
  -> allowed roles
  -> route
  -> navigation visibility
  -> page actions
```

Prefer capabilities or allowed actions over scattered role-string conditions. Where the
backend returns allowed actions, use them for state-dependent controls.

Do not claim the user can act merely because a menu item is visible.

## Role UX

### Karyawan

- Access own profile, personal metrics, attendance, own leave/overtime requests, and own
  notifications.
- Do not show HR dashboard, employee administration, or AKSES.

### Atasan

- Include Karyawan capabilities.
- Show approval work for direct reports only.
- Do not imply access to employees outside the returned scope.

### HR

- Show employee administration, HR dashboard, monitoring, master leave types, role and
  permission administration, and audit log.
- Show final approval actions when HR is the active stage.

### Top Management

- Show monitoring routes in read-only mode.
- Do not show Personal Metrics.
- Do not show edit/delete/permission mutation controls.
- Show final decision controls only for requests owned by HR when the backend marks the
  action available.

## Action rendering

Use:

```text
hidden   -> action is irrelevant or disallowed and discovery is not useful
disabled -> action is visible but temporarily unavailable; explain why
enabled  -> capability and current resource state allow it
```

Do not hide meaningful read-only data merely because mutation controls are unavailable.
Status-dependent authorization must be rechecked after mutations.

## Error behavior

- `401`: clear token/session metadata, all protected query cache, sensitive client state,
  pending uploads, and derived permission state; redirect safely to login.
- `403`: keep the authenticated session; show forbidden or remove the disallowed action.
- `409 ALREADY_DECIDED`: refetch approval detail/list and explain that another decision won.
- `429 ACCOUNT_LOCKED`: show the documented lockout message without exposing account
  existence beyond the API response policy.

Avoid redirect loops. Keep login failure on the login page.

## Logout

1. Disable duplicate logout action.
2. Call the backend logout endpoint when a session exists.
3. Clear local auth state even if network logout cannot complete, while reporting the
   limitation appropriately.
4. Cancel in-flight protected requests.
5. Clear protected server cache and user-scoped client state.
6. Release media streams and object URLs.
7. Navigate with history replacement to login.

If multi-tab coordination is approved, broadcast logout without persisting sensitive
payloads.

## Testing matrix

Test:

- Anonymous direct protected URL.
- Authentication initialization without protected flash.
- Each role's navigation.
- Each role's direct route access.
- Karyawan/Atasan row scope using API mocks.
- Top Management read-only controls and HR-request decision exception.
- `401` cleanup and redirect.
- `403` without logout.
- Expiry during a form or upload.
- Logout during in-flight requests.
- Unsafe return-path rejection.

## Anti-patterns

- Store the JWT in browser persistence without an approved threat-model decision.
- Add a refresh cookie/endpoint because another project uses one.
- Scatter `role === "hr"` across components.
- Fetch protected page data before capability resolution.
- Treat hidden buttons as security.
- Log out on every `403`.
- Cache permissions after logout or across users.
