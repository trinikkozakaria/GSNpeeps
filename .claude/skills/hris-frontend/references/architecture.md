# Frontend architecture

Use this reference when establishing the frontend foundation, placing a new module,
reviewing imports, or changing routing, state, API, build, and delivery boundaries.

## Contents

- [Architecture target](#architecture-target)
- [Runtime context](#runtime-context)
- [Canonical structure](#canonical-structure)
- [React Router structure](#react-router-structure)
- [Module anatomy](#module-anatomy)
- [Dependency direction](#dependency-direction)
- [Module boundaries](#module-boundaries)
- [Data ownership](#data-ownership)
- [Route architecture](#route-architecture)
- [Provider composition](#provider-composition)
- [API and model boundaries](#api-and-model-boundaries)
- [Authentication boundary](#authentication-boundary)
- [Performance and delivery](#performance-and-delivery)
- [Error and state architecture](#error-and-state-architecture)
- [Configuration](#configuration)
- [Architecture extension gate](#architecture-extension-gate)
- [Review checklist](#review-checklist)

## Architecture target

Build a browser-delivered React application that communicates with the Go backend through
`/api/v1`. The approved baseline produces static production assets served by Nginx.

The final foundation is React JavaScript/JSX, Vite, React Router, Tailwind CSS, Axios,
TanStack Query, React Hook Form, Zod, Vitest, Testing Library, Playwright, and pnpm.

```text
Browser
  -> Nginx
      |-> static frontend assets
      `-> /api/v1 proxy -> Go API
```

Use a business-module-oriented frontend:

```text
route -> page composition -> module component/hook -> API boundary
                      |                |
                      v                v
                 shared UI       server-state boundary
```

Apply these principles:

- Keep route files thin and module behavior cohesive.
- Make shared primitives domain-neutral and module composites domain-aware.
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

Use this target:

```text
frontend/
|-- public/
|-- src/
|   |-- app/                    provider/bootstrap composition
|   |-- routes/                 React Router route tree, guards, navigation
|   |-- modules/               one folder per business capability
|   |   |-- auth/
|   |   |-- dashboard/
|   |   |-- employees/
|   |   |-- profile/
|   |   |-- attendance/
|   |   |-- attendance-reports/
|   |   |-- leave/
|   |   |-- overtime/
|   |   |-- approvals/
|   |   |-- notifications/
|   |   |-- access/
|   |   `-- audit/
|   |-- components/
|   |   |-- ui/                domain-neutral primitives
|   |   |-- form/              approved form-layer adapters
|   |   |-- layout/            shell, sidebar, topbar, containers
|   |   |-- data-table/        cross-module table primitives/composition
|   |   |-- charts/            cross-module chart wrappers when justified
|   |   `-- feedback/          loading, empty, error, forbidden
|   |-- lib/
|   |   |-- api/               transport, envelope, auth header, error mapping
|   |   `-- query/             shared TanStack Query client/config
|   |-- hooks/                  truly cross-module hooks only
|   |-- stores/                 only if a global client-state library is later approved
|   |-- schemas/                shared contract-derived validation
|   |-- mocks/                  approved development/test network mocks
|   |-- styles/                 tokens and global styles
|   |-- test/                   test render, handlers, fixtures
|   `-- main.jsx                Vite entry
|-- .env.example
|-- Dockerfile
|-- package.json
|-- pnpm-lock.yaml
`-- vite.config.js
```

Do not commit generated directories such as `dist`, `node_modules`, coverage,
workbench output, caches, or debug logs. Do not keep backup routing files such as
`middleware-bck.*`. Remove obsolete code through the approved Git workflow.

## React Router structure

```text
src/
|-- app/
|   |-- App.jsx
|   `-- providers.jsx
|-- routes/
|   |-- guards/
|   |-- navigation/
|   `-- router.jsx
`-- main.jsx
```

Keep route elements thin and compose pages from `src/modules`. Configure Nginx fallback to
`index.html` for direct client-route navigation. Do not create Next.js route groups,
middleware, proxy, server actions, or a second route tree.

## Module anatomy

Keep module-local code inside its module. Create only directories required by that module:

```text
modules/employees/
|-- api/
|   |-- employee-api.js
|   `-- employee-query-keys.js
|-- components/
|   |-- EmployeeFilters.jsx
|   |-- EmployeeForm.jsx
|   `-- EmployeeTable.jsx
|-- hooks/
|   |-- useEmployee.js
|   `-- useEmployees.js
|-- pages/
|   |-- EmployeeCreatePage.jsx
|   |-- EmployeeDetailPage.jsx
|   |-- EmployeeEditPage.jsx
|   `-- EmployeeListPage.jsx
|-- schemas/
|   `-- employee-schema.js
|-- utils/
|   `-- employee-view-model.js
|-- tests/
`-- index.js
```

The same rule applies to dashboard-style modules:

```text
modules/dashboard/
|-- api/
|-- components/
|   |-- hr/
|   |   |-- EmployeeDistributionChart.jsx
|   |   `-- WorkforceSummaryCards.jsx
|   |-- employee/
|   |   `-- PersonalMetricCards.jsx
|   |-- DashboardFilters.jsx
|   `-- MetricCard.jsx
|-- hooks/
|-- pages/
|   |-- HrDashboardPage.jsx
|   `-- PersonalDashboardPage.jsx
|-- tests/
`-- index.js
```

Use role subdirectories only when the rendered composition is materially different. Prefer
capability checks and shared module components when the difference is only visibility or one
action. Never create complete `Admin`, `HR`, `Supervisor`, and `Employee` application trees.

Directory responsibilities:

| Directory | Responsibility |
|---|---|
| `pages/` | Route-level composition, guards, data states, and module assembly |
| `components/` | Presentational or interactive UI owned only by the module |
| `hooks/` | Module orchestration, queries, mutations, and browser behavior |
| `api/` | Module endpoint calls, query keys, and contract/view-model adapters |
| `schemas/` | Module request/form validation derived from OpenAPI |
| `utils/` | Pure module-specific formatting or calculation helpers |
| `constants/` | Stable module labels/options when a separate file is justified |
| `tests/` | Module unit, component, integration, fixture, and handler tests |
| `index.*` | Small public module API; never a dump of every private export |

Do not require every directory in every module. For example, a read-only notification module
may need `api`, `components`, `hooks`, `pages`, and `tests` but no form schema.

Use root `lib`, `hooks`, and `schemas` only for genuine cross-module infrastructure.
Use `src/lib/api` as the single shared transport location; do not also create `src/api`.

## Dependency direction

Use this import direction:

```text
routing layer -> modules -> shared components
                       |-> module API
                       `-> shared lib

lib/api -> transport, envelope, errors, environment config
shared components -> styles/tokens
```

Apply these boundaries:

| Area | May depend on | Must not depend on |
|---|---|---|
| `components/ui` | React, tokens, small UI utilities | modules, API, auth store, form library |
| `components/form` | UI primitives and approved form adapter | module API and business rules |
| `components/layout` | UI, navigation capabilities | module repositories or endpoint details |
| `modules/*` | shared UI, API, schemas, approved state layers | another module's internals |
| `lib/api` | environment, transport, envelope, stable errors | route components and module UI |
| routing layer | module public exports, guards, layouts | module-private files |
| `stores` | client/session state | duplicated server resource collections |
| `schemas` | contract-derived values | rendered components |

Expose a small public module API through an index module only if the build configuration
handles it safely. Do not use a barrel that creates cycles or unnecessarily expands bundles.

## Module boundaries

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

Keep product statuses, labels, capabilities, and module-specific formatters within the
owning module. Share only stable concepts used by multiple modules.

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
  -> routing layer
  -> notification/toast layer
  -> application
```

Adjust order for the approved libraries, but make dependencies explicit. Do not let module
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

Do not create a refresh call: the approved contract provides login, self-reset, logout,
current-user restoration, and own-password change, but no refresh operation.

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

- Split code at React Router/module boundaries with dynamic `import()` where beneficial.
- Lazy-load camera, chart, export-preview, and other heavy dependencies.
- Avoid importing all business modules into the initial shell.
- Size and compress images/icons appropriately.
- Prevent repeated request waterfalls by composing queries deliberately.
- Virtualize only when real dataset/UI constraints justify it.
- Keep production configuration compatible with the approved delivery mode; the baseline is
  static Nginx assets.
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

Use reusable feedback components for visual consistency. Keep module-specific recovery
actions near the module. Never show raw stack traces or transport internals.

## Configuration

- Validate public configuration at application startup.
- Expose only non-secret values to the browser.
- Keep the API base URL centralized.
- Prefer same-origin `/api/v1` in production through Nginx when deployment supports it.
- Keep `.env.example` free of real credentials and production employee data.
- Fail visibly in development/build when required configuration is missing.

## Architecture extension gate

The foundation stack is final. Table/chart/icon libraries, a global client store, network
mock layer, component workbench, formatter, and additional linter remain optional.

Before introducing an optional dependency:

1. Inspect the repository for an existing approved choice.
2. Read the active foundation prompt and architecture decision record.
3. Reuse an existing coherent stack.
4. If absent, propose the smallest compatible option with tradeoffs.
5. Obtain approval when the decision affects long-term architecture or delivery.
6. Record the decision and update this skill where library-specific rules become stable.

Do not introduce Next.js, TypeScript, npm/yarn, Jest, or parallel HTTP/query/form/router
libraries. Treat supplied example projects as layout ideas only.

## Review checklist

- [ ] The code lives in the correct module/shared layer.
- [ ] Imports follow the dependency direction and have no cycle.
- [ ] The API contract is derived from OpenAPI without guessed fields/endpoints.
- [ ] Auth resolution precedes protected rendering and sensitive requests.
- [ ] Route/menu checks use centralized capabilities.
- [ ] Server, URL, form, local, and global state have correct owners.
- [ ] All relevant loading/empty/error states exist.
- [ ] Mobile, keyboard, focus, zoom, and non-color status behavior are covered.
- [ ] Protected cache and transient resources are cleaned on logout/unmount.
- [ ] The approved production build/delivery mode works through Nginx.
- [ ] New non-baseline dependencies passed the architecture extension gate.
