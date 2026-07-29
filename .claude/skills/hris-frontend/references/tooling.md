# Frontend tooling

Use only choices approved in the repository configuration or architecture decision record.

## Contents

[Decision inventory](#decision-inventory) ·
[Required command interface](#required-command-interface) ·
[Package manager](#package-manager) ·
[Formatter and linter](#formatter-and-linter) · [Tests](#tests) ·
[Component workbench](#component-workbench) · [Mock tooling](#mock-tooling) ·
[Static production delivery](#static-production-delivery) ·
[Dependency decision](#dependency-decision) · [CI quality gate](#ci-quality-gate)

## Decision inventory

Before changing tooling, inspect:

- `package.json` scripts and dependencies.
- Package-manager lockfile.
- Language/compiler configuration.
- Bundler and React plugin.
- Tailwind/PostCSS configuration.
- Router.
- Formatter and linter.
- Unit/component/E2E configuration.
- Mock and component-workbench configuration.
- Dockerfile and Nginx static delivery.

Do not assume the example project's Next.js, JavaScript, Axios, TanStack Query, Zustand, Zod,
MSW, Storybook, oxlint, or oxfmt choices.

If Next.js is proposed, record whether production uses static export or a Node/standalone
runtime. Static export must remain compatible with the approved Nginx topology. A Node
runtime requires a separate deployment decision and updated container/proxy design.

## Required command interface

Provide repository-defined equivalents for:

```text
dev
build
format
format-check
lint
test
test-watch (optional local)
test-e2e
preview (when supported)
```

Use clear exit codes and make CI use non-interactive commands.

## Package manager

- Select one package manager.
- Commit exactly its expected lockfile.
- Use reproducible/frozen installs in CI and container builds.
- Do not mix lockfiles or global-only dependencies.
- Keep engine/runtime version documented and aligned with CI/container.
- Review dependency purpose and production versus development placement.

## Formatter and linter

- Keep one formatting source of truth.
- Keep one primary lint path unless an approved tool has a distinct non-overlapping purpose.
- Integrate framework, hooks, accessibility, and import rules supported by the chosen stack.
- Prefer narrow justified suppression with a comment.
- Do not disable important rules globally to silence generated debt.
- Ensure editor settings call the repository tools rather than competing formatters.

## Tests

- Keep unit/component tests in the approved runner.
- Keep browser E2E in one approved tool.
- Make test environment, DOM cleanup, mocks, and path aliases match production code.
- Produce useful CI artifacts only when needed.
- Avoid adding a second runner for one test.

## Component workbench

Add Storybook or another workbench only after approval and when its maintenance cost is
justified.

If approved:

- Reuse app tokens/styles/providers.
- Cover primitive, form-adapter, loading, empty, error, long-content, and accessibility
  variants.
- Use network mocks, never production APIs.
- Keep workbench-only code out of the production bundle.
- Run an approved static build/check in CI.

Do not make story files mandatory before the workbench decision is recorded.

## Mock tooling

- Keep dev/test mocks gated explicitly.
- Exclude them from production output.
- Validate representative fixtures against OpenAPI-derived schemas.
- Fail production checks if mock mode is enabled.

## Static production delivery

The production build must:

- Emit static assets compatible with Nginx.
- Use correct asset base paths.
- Support client-route fallback if a client router is selected.
- Use same-origin/proxied `/api/v1` where deployment configuration defines it.
- Contain no public secret or development mock.
- Handle cache headers so new HTML does not reference deleted hashed assets.

Do not commit generated output such as `.next`, `dist`, `node_modules`, Storybook output,
coverage, tool caches, or debug logs. Do not preserve obsolete routing code as
`middleware-bck.*`; version control is the backup.

## Dependency decision

For a new major dependency, record:

- Requirement it solves.
- Existing-stack capability.
- Alternatives considered.
- Bundle/runtime/maintenance/security impact.
- Static-build compatibility.
- Testing and migration cost.

Prefer existing approved capability when sufficient. Do not add overlapping HTTP, state,
form, table, date, icon, or UI libraries.

## CI quality gate

Run in a deterministic order:

1. Frozen install.
2. Format check.
3. Lint.
4. Unit/component tests.
5. Production build.
6. Relevant E2E/accessibility checks.
7. Optional bundle/workbench checks when approved.

Report exact failures; do not bypass checks by weakening configuration.
