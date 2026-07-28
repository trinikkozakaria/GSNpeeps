# Frontend architecture

Use this reference when establishing the frontend foundation, placing a new feature,
reviewing imports, or changing routing, state, API, build, and delivery boundaries.

## Contents

- [Architecture target](#architecture-target)
- [Runtime context](#runtime-context)
- [Canonical structure](#canonical-structure)
- [Dependency direction](#dependency-direction)
- [Feature boundaries](#feature-boundaries)
- [Data ownership](#data-ownership)
- [Route architecture](#route-architecture)
- [Provider composition](#provider-composition)
- [API and model boundaries](#api-and-model-boundaries)
- [Authentication boundary](#authentication-boundary)
- [Performance and delivery](#performance-and-delivery)
- [Error and state architecture](#error-and-state-architecture)
- [Configuration](#configuration)
- [Architecture decision gate](#architecture-decision-gate)
- [Review checklist](#review-checklist)

## Architecture target

Build a client-side React application that produces static production assets served by
Nginx and communicates with the Go backend through `/api/v1`.

```text
Browser
  -> Nginx
      |-> static frontend assets
      `-> /api/v1 proxy -> Go API
```

Use a feature-oriented modular frontend:

```text
route -> page composition -> feature component/hook -> API boundary
                      |                |
                      v                v
                 shared UI       server-state boundary
```

Apply these principles:

- Keep route modules thin and feature behavior cohesive.
- Make shared primitives domain-neutral and feature composites domain-aware.
- Keep the OpenAPI contract at the API boundary.
- Keep server data, URL state, form state, and local UI state in their proper owners.
- Centralize authentication, response errors, route capabilities, and navigation.
- Prefer explicit dependencies and composition over global access and hidden side effects.
- Produce static assets compatible with the approved Nginx deployment.

## Runtime context

GSNpeeps is an internal HR application containing sensitive employee, salary, identity,
attendance-photo, and approval data.

```text
public shell
  -> authenticate
  -> resolve current user and role
  -> mount protected shell
  -> render only permitted navigation and routes
  -> fetch only data permitted for that route and scope
```

Never render a protected page optimistically while authentication is unresolved. Never
prefetch a route merely because its menu is hidden; verify capability before sensitive data
requests begin.

The backend remains authoritative for:

- Authentication and active Redis session.
- Role, ownership, subordinate relation, and approval-stage authorization.
- Server time, attendance radius result, and business calculations.
- File validation and Nextcloud access.

## Canonical structure

Use this target after foundational tooling is approved:

```text
frontend/
|-- public/
|-- src/
|   |-- app/                    startup and provider composition
|   |-- routes/                 route definitions, guards, lazy boundaries
|   |-- features/
|   |   |-- auth/
|   |   |-- dashboard/
|   |   |-- employees/
|   |   |-- profile/
|   |   |-- attendance/
|   |   |-- leave/
|   |   |-- overtime/
|   |   |-- notifications/
|   |   `-- access/
|   |-- components/
|   |   |-- ui/                domain-neutral primitives
|   |   |-- form/              approved form-layer adapters
|   |   |-- layout/            shell, sidebar, topbar, containers
|   |   `-- feedback/          loading, empty, error, forbidden
|   |-- api/                    client, endpoint modules, error mapping
|   |-- hooks/                  truly cross-feature hooks only
|   |-- stores/                 approved global client state only
|   |-- schemas/                shared contract-derived validation
|   |-- styles/                 tokens and global styles
|   |-- test/                   test render, handlers, fixtures
|   `-- main.<approved-ext>
|-- .env.example
|-- Dockerfile
|-- package.json
`-- <approved-lockfile>
```

Keep feature-local code inside the feature:

```text
features/employees/
|-- api/
|-- components/
|-- hooks/
|-- schemas/
|-- routes/
|-- tests/
`-- index.<approved-ext>
```

Use root `api`, `hooks`, and `schemas` only for genuine cross-feature infrastructure.
Avoid duplicate locations such as both `lib/api` and `api` for the same responsibility.

## Dependency direction

Use this import direction:

```text
app/routes -> features -> shared components
                   |-> api
                   |-> shared schemas/hooks

api -> contract models + environment config
shared components -> styles/tokens
```

Apply these boundaries:

| Area | May depend on | Must not depend on |
|---|---|---|
| `components/ui` | React, tokens, small UI utilities | features, API, auth store, form library |
| `components/form` | UI primitives and approved form adapter | feature API and business rules |
| `components/layout` | UI, navigation capabilities | feature repositories or endpoint details |
| `features/*` | shared UI, API, schemas, approved state layers | another feature's internals |
| `api` | environment, transport, contract models | route components and feature UI |
| `routes` | feature public exports, guards, layouts | feature-private files |
| `stores` | client/session state | duplicated server resource collections |
| `schemas` | contract-derived values | rendered components |

Expose a small public feature API through an index module only if the build configuration
handles it safely. Do not use a barrel that creates cycles or unnecessarily expands bundles.

## Feature boundaries

Treat these as the primary product modules:

- Authentication.
- Employee database and documents.
- Personal profile and personal metrics.
- HR/Top Management dashboard.
- Attendance, live feed, report, and export.
- Leave and its approval flow.
- Overtime and its approval flow.
- Notifications.
- Roles, permissions, and audit log.

Keep product statuses, labels, capabilities, and feature-specific formatters within the
owning feature. Share only stable concepts used by multiple features.

Do not implement Hiring Progress, Recruitment Cost, or Benefit beyond an approved Coming
Soon state.

## Data ownership

Use this decision table:

| State | Owner |
|---|---|
| API resources, lists, detail, metrics | Approved server-state/cache layer |
| Page, search, sort, filters, active tab when shareable | URL |
| Input values and field errors | Approved form layer or local form state |
| Dialog open state and transient toggles | Local component |
| Cross-route UI preference | Approved client store/context |
| Auth identity/token metadata | Approved auth/session boundary |
| Derived totals/labels | Compute from source data; do not duplicate |

Never copy an employee list or notification list into global client state. Never persist
passwords, photo blobs, documents, salary data, or entire protected responses.

## Route architecture

Define each route with:

- Path and route parameter schema.
- Required authentication.
- Required capability/role and any UX scope.
- Data dependencies.
- Loading, empty, forbidden, not-found, conflict, and server-error behavior.
- Mobile and keyboard behavior.
- Document title.

Centralize route and navigation metadata so the sidebar, breadcrumbs, and route guard use
the same capability definition. Keep dynamic ownership and approval-stage checks on the
backend; the frontend may adapt controls using returned allowed actions.

Validate route and query parameters before issuing requests. Preserve list filters when
navigating list -> detail -> back.

## Provider composition

Mount global providers in one visible composition root:

```text
error boundary
  -> auth bootstrap
  -> server-state provider
  -> router
  -> notification/toast layer
  -> application
```

Adjust order for the approved libraries, but make dependencies explicit. Do not let feature
imports register interceptors, handlers, or global listeners as hidden side effects.

Clean up listeners, timers, network requests, media streams, and object URLs during
unmount/logout.

## API and model boundaries

- Read endpoint, method, query, payload, envelope, and error code from OpenAPI.
- Keep wire fields in `snake_case`.
- Use a separate view model only for meaningful display transformation.
- Centralize JSON, multipart, and authenticated download behavior.
- Preserve abort/cancellation from route/component to transport.
- Keep API endpoint functions free of toast, navigation, and component state.
- Map transport failures into one stable frontend error type.

Do not create a refresh call: the approved GSNpeeps contract provides login and logout only.

## Authentication boundary

Define and document the approved token-storage strategy before implementation. Account for:

- Eight-hour JWT expiry.
- Active-session invalidation by the backend/Redis.
- Logout from the current tab and, if required, other open tabs.
- Full protected-cache and sensitive-state cleanup.
- Reload behavior without inventing a refresh endpoint.
- `401` ending the local session.
- `403` showing forbidden without treating the session as invalid.

Do not persist a Bearer token silently. Do not rely on route hiding as security.

## Performance and delivery

- Split code at route/feature boundaries when supported by the approved router/build tool.
- Lazy-load camera, chart, export-preview, and other heavy dependencies.
- Avoid importing all feature modules into the initial shell.
- Size and compress images/icons appropriately.
- Prevent repeated request waterfalls by composing queries deliberately.
- Virtualize only when real dataset/UI constraints justify it.
- Keep production configuration runtime/deployment compatible with static Nginx assets.
- Verify direct-route fallback behavior in Nginx for client-side routing.

Do not sacrifice accessibility or correctness for micro-optimizations.

## Error and state architecture

Every data surface must deliberately cover:

```text
initial/loading
success with data
success with no data
filtered empty
validation error
401 session ended
403 forbidden
404 not found
409 conflict/already decided
422 field validation
429 lockout/rate limit
5xx/network/offline
```

Use reusable feedback components for visual consistency. Keep feature-specific recovery
actions near the feature. Never show raw stack traces or transport internals.

## Configuration

- Validate public configuration at application startup.
- Expose only non-secret values to the browser.
- Keep the API base URL centralized.
- Prefer same-origin `/api/v1` in production through Nginx when deployment supports it.
- Keep `.env.example` free of real credentials and production employee data.
- Fail visibly in development/build when required configuration is missing.

## Architecture decision gate

The approved documents establish React, Tailwind CSS, static production assets, and Nginx.
They do not yet establish language, bundler, router, package manager, HTTP client, form
library, validation library, server-state library, client store, table library, mock layer,
test runner, browser E2E tool, component workbench, formatter, or linter.

Before introducing any of those:

1. Inspect the repository for an existing approved choice.
2. Read the active foundation prompt and architecture decision record.
3. Reuse an existing coherent stack.
4. If absent, propose one compatible stack with alternatives and tradeoffs.
5. Obtain approval when the decision affects long-term architecture or delivery.
6. Record the decision and update this skill where library-specific rules become stable.

Treat the supplied example project as a pattern reference only. Do not inherit its product
names, routes, refresh-token flow, framework, file extensions, or libraries.

## Review checklist

- [ ] The code lives in the correct feature/shared layer.
- [ ] Imports follow the dependency direction and have no cycle.
- [ ] The API contract is derived from OpenAPI without guessed fields/endpoints.
- [ ] Auth resolution precedes protected rendering and sensitive requests.
- [ ] Route/menu checks use centralized capabilities.
- [ ] Server, URL, form, local, and global state have correct owners.
- [ ] All relevant loading/empty/error states exist.
- [ ] Mobile, keyboard, focus, zoom, and non-color status behavior are covered.
- [ ] Protected cache and transient resources are cleaned on logout/unmount.
- [ ] Static production build works through Nginx.
- [ ] New dependencies passed the architecture decision gate.
