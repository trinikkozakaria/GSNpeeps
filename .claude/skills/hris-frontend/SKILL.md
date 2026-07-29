---
name: hris-frontend
description: Implement and review the GSNpeeps React and Tailwind frontend, including application architecture, API integration, authentication UX, role-aware navigation, employee dashboards, forms, tables, attendance camera and GPS flows, approvals, notifications, access administration, accessibility, responsive behavior, mocks, and frontend tests. Use for any browser UI, component, page, route, state, validation, styling, or frontend integration task in this repository.
---

# GSNpeeps Frontend

Implement a secure, accessible, responsive React frontend for the GSNpeeps employee
database, HR dashboard, attendance, approval, notification, and access-management scope.

## Workflow

1. Read `CLAUDE.md`, the active prompt, and `../../specs/product-requirements.md`.
2. Read `../../specs/access-matrix.md`, `../../specs/api-data-summary.md`, and
   `docs/openapi.yaml` before defining routes, payloads, response models, or actions.
3. Inspect `frontend/package.json`, `pnpm-lock.yaml`, Vite configuration, design tokens,
   React Router composition, and existing components.
4. Preserve the approved stack; only optional task-specific additions use the extension gate.
5. Load only the references relevant to the current task.
6. Define scope, route capability, data states, API errors, responsive behavior, and tests.
7. Build the smallest complete vertical slice using shared primitives and business modules
   under `src/modules`.
8. Verify every affected role, direct forbidden navigation, keyboard use, mobile layout,
   sensitive-cache cleanup, and documented API failures.
9. Run the repository-defined format, lint, test, production build, and relevant E2E checks.

## Invariants

- Display the product name **GSNpeeps**.
- Use React JavaScript/JSX, Vite, React Router, Tailwind CSS, Axios, TanStack Query,
  React Hook Form, Zod, Vitest, Testing Library, Playwright, and pnpm.
- Do not introduce Next.js, TypeScript, npm/yarn lockfiles, Jest, or parallel HTTP/query/form/router tools.
- Treat backend authorization as authoritative; frontend guards are UX only.
- Never fetch, cache, or render data outside the authenticated role/scope.
- Centralize API calls, error mapping, auth lifecycle, query keys, and navigation capabilities.
- Mirror OpenAPI exactly; do not create a second frontend contract or guessed endpoint.
- GSNpeeps has login/logout/current-user/password operations, an eight-hour JWT, and no
  refresh endpoint in the approved contract.
- Keep API wire fields in `snake_case`; introduce a view-model mapping only when it solves
  a real presentation need.
- Keep server state out of global client state and keep shareable filters in the URL.
- Prevent protected-content flashes and sensitive requests before auth/capability resolution.
- Keep interactions keyboard accessible, responsive, and understandable without color alone.
- Provide camera failure fallback without changing server-authoritative time/radius.
- Show Coming Soon only for Hiring Progress, Recruitment Cost, and Benefit.
- Treat non-baseline optional libraries as an architecture extension decision.
- Give every business capability its own `src/modules/<module>` boundary. Keep its pages,
  components, hooks, API adapters, schemas, utilities, and tests together; create only the
  subdirectories the module actually needs.
- Keep role-specific variants inside the owning module. Do not create parallel applications
  or duplicate complete module trees for HR, supervisor, employee, or top management.
- Keep shared transport and framework-neutral infrastructure in `src/lib`; keep resource
  endpoints, components, hooks, schemas, and tests in the owning business module.
- Use `src/routes` with React Router; `src/app` is provider/bootstrap composition only.

## Reference routing

- Read `references/architecture.md` for package boundaries, dependency direction, React
  Router composition, runtime configuration, and the extension gate.
- Read `references/api-client.md` for JSON, multipart, downloads, cancellation, response
  envelopes, and error normalization.
- Read `references/auth-rbac.md` for login/logout, token lifecycle, route protection,
  capability navigation, role scope, and session cleanup.
- Read `references/accessibility.md` for keyboard, focus, semantics, live feedback, charts,
  camera/GPS, responsive layouts, and verification.
- Read `references/client-state.md` before adding context, reducer, or a global store.
- Read `references/component-pattern.md` before creating or changing React components.
- Read `references/data-table.md` for server pagination, filters, URL state, actions, and
  responsive table behavior.
- Read `references/form-pattern.md` for create/edit forms, field errors, conditional fields,
  uploads, confirmation, and double-submit protection.
- Read `references/mocking.md` for OpenAPI-aligned fixtures, network handlers, browser APIs,
  and production exclusion.
- Read `references/page-pattern.md` for route guards and complete loading/error/empty states.
- Read `references/reusable-components.md` before adding a shared primitive, form control,
  layout component, or feature composite.
- Read `references/server-state.md` for query keys, caching, invalidation, cancellation,
  retry, polling, and logout cleanup.
- Read `references/tooling.md` before adding or changing the package manager, formatter,
  linter, component workbench, generator, build command, or dependency.
- Read `references/validation-schemas.md` for OpenAPI-derived schemas, nullable/optional
  fields, enums, conditional rules, and wire/view-model mapping.
- Read `references/testing.md` for unit, component, integration, E2E, accessibility,
  browser-permission, role-matrix, and sensitive-data tests.

## Completion report

Report:

- Routes/components/hooks/schemas changed.
- API contract and role assumptions used.
- Commands run and exact results.
- Accessibility and responsive checks performed.
- Mock/fixture changes.
- Known limitations or unresolved contract decisions.
