# Component pattern

Use this reference whenever creating or reviewing a React component.

## Contents

[Component categories](#component-categories) · [Component contract](#component-contract) ·
[Data and behavior](#data-and-behavior) ·
[Effects and resource cleanup](#effects-and-resource-cleanup) ·
[Props and events](#props-and-events) · [Rendering](#rendering) ·
[Boundaries](#boundaries) · [Reuse threshold](#reuse-threshold) ·
[Tests](#tests) · [Anti-patterns](#anti-patterns)

## Component categories

| Category | Responsibility | Examples |
|---|---|---|
| Primitive | Domain-neutral appearance and interaction | Button, Input, Dialog |
| Form adapter | Connect primitive to approved form state | FormInput, FormSelect |
| Feature component | GSNpeeps business presentation | EmployeeStatus, DecisionDialog |
| Page composition | Route guard, data states, feature assembly | EmployeeListPage |
| Hook | Reusable state/effect behavior | useEmployeeList |

Keep network operations out of primitives. Keep route navigation out of reusable domain
formatters and endpoint modules.

## Component contract

- Use JavaScript: `.jsx` for files rendering JSX and `.js` for non-JSX modules.
- Name components in PascalCase and hooks with `use`.
- Accept explicit props; avoid passing an unbounded page object.
- Use semantic defaults and allow safe extension through `className`/approved style props.
- Forward refs only where consumers genuinely need DOM focus/measurement integration.
- Preserve native attributes and accessibility behavior.
- Document non-obvious props and side effects using the approved documentation convention.

Use named exports by default for reusable components/hooks and keep one primary component per
file. Do not add TypeScript or duplicate `.tsx` variants.

## Data and behavior

- Keep API/cache logic in feature hooks or page composition.
- Pass prepared data and callbacks to presentational components.
- Keep feature status labels and mappings in the owning feature.
- Derive values during render when possible.
- Use effects only to synchronize with external systems.
- Keep mutation pending/disabled behavior explicit.
- Model loading, empty, error, readonly, and unavailable states intentionally.

## Effects and resource cleanup

Every effect that acquires a resource must release it:

| Resource | Cleanup |
|---|---|
| MediaStream | Stop every track |
| Geolocation watch | Clear watch |
| Timer/interval | Clear timer |
| Event listener | Remove same listener |
| Network request | Abort/cancel |
| Object URL | Revoke |
| Subscription | Unsubscribe |

Handle development strict-mode remounting without duplicating resources.

## Props and events

- Use `is*`, `has*`, or `can*` for booleans.
- Use `on*` for callback props and `handle*` for local handlers.
- Pass IDs/actions rather than exposing cache/client objects.
- Keep callback payloads stable and documented.
- Avoid boolean-prop explosions; prefer a small variant/state model when combinations grow.
- Do not mutate props.

## Rendering

- Use stable server IDs for list keys.
- Never use array index for reordered/inserted/deleted lists.
- Avoid unsafe HTML rendering.
- Format dates, numbers, and currency through centralized locale utilities.
- Render status with text/icon plus color.
- Keep headings hierarchical and interactive elements native where possible.

## Boundaries

- Primitive must not import a form library, API client, auth store, or feature module.
- Form adapter must delegate visual styling to primitives.
- Feature components may use domain labels and feature hooks.
- Page composition owns route-specific orchestration.
- Components must not guess API fields.

## Reuse threshold

Create a shared abstraction when:

- At least two coherent consumers need the same behavior, or
- A strong consistency/accessibility/security requirement justifies it immediately.

Keep one-off product composition local. Do not build a mega-component with dozens of
configuration props merely to avoid small readable duplication.

## Tests

Test public behavior:

- Accessible name/role and keyboard operation.
- Props and callback payload.
- Loading/disabled/error/empty variants.
- Cleanup of acquired resources.
- Role/capability variants for feature components.
- Mobile wrapping/overflow for critical layouts.

Avoid asserting internal hook calls or CSS implementation details.

## Anti-patterns

- Fetch directly inside a primitive.
- Define validation schemas inline in page markup.
- Couple a button to global auth state.
- Use an effect to calculate display text.
- Leave streams, timers, or object URLs alive after unmount.
- Create local colors/sizing when a token/component variant exists.
- Duplicate the same accessible control in multiple features.
