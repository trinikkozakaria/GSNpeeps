# Reusable component catalog

# Reusable components

Use a layered catalog so visual controls remain independent from form and business logic.

## Contents

[Two-layer control pattern](#two-layer-control-pattern) ·
[Field anatomy](#field-anatomy) · [Primitive catalog](#primitive-catalog) ·
[Layout catalog](#layout-catalog) · [Domain composites](#domain-composites) ·
[Variants](#variants) · [Required states](#required-states) ·
[Creation workflow](#creation-workflow) · [Anti-patterns](#anti-patterns)

## Two-layer control pattern

### Layer 1: UI primitive

Place domain-neutral controls in `components/ui`.

- Own visual states, semantic element, focus ring, disabled state, and variants.
- Accept native attributes and a ref when necessary.
- Expose `aria-*`, `id`, `name`, and event props.
- Do not import the form library, API client, auth state, or feature code.

### Layer 2: form adapter

Place approved form-library integration in `components/form`.

- Read field value/error/touched state from the form layer.
- Connect label, description, and error IDs.
- Delegate all control visuals to the UI primitive.
- Avoid duplicating Tailwind classes already owned by the primitive.
- Support nested field paths and server errors.

If no form library has been approved, preserve the separation concept with controlled props;
do not add a dependency silently.

## Field anatomy

Every form control should compose:

```text
Field
  |-- visible Label
  |-- optional marker / requirement text
  |-- Control
  |-- Description
  `-- Error message
```

Connect with `htmlFor`, `id`, `aria-describedby`, and `aria-invalid`. Do not use placeholder
text as the only label.

## Primitive catalog

Prefer shared implementations for:

- Button and IconButton.
- Input, PasswordInput, NumberInput, Textarea.
- Select/Combobox, Checkbox, RadioGroup, Switch.
- Card, MetricCard, Badge, StatusBadge, Alert.
- Dialog, ConfirmDialog, Dropdown/Menu, Tabs.
- Spinner, Skeleton, EmptyState, ErrorState, ForbiddenState.
- Tooltip only for supplemental information, never required instructions.

Use native controls where they satisfy UX and accessibility. Select more complex custom
controls only after keyboard and screen-reader behavior are covered.

## Layout catalog

- AppShell, Sidebar, Topbar, UserMenu.
- PageContainer, PageHeader, Breadcrumbs.
- ResponsiveStack/Grid.
- Section and DetailList.

Keep route capabilities in centralized navigation data; do not hardcode role checks inside
the visual Sidebar primitive.

## Domain composites

Feature-aware reusable components may include:

- DataTable, Pagination, FilterBar, SearchInput.
- FileUpload, FilePreview, ExportMenu.
- NotificationBell and NotificationItem.
- ApprovalTimeline and DecisionDialog.
- AttendanceCamera, LocationStatus, WorkModeSelector.
- EmployeeStatus and ContractExpiryBadge.

Keep these in their feature when not genuinely cross-feature.

## Variants

- Extend components through approved design tokens and a finite variant API.
- Reuse semantic names such as `default`, `danger`, `success`, and `warning`.
- Keep size and state variants consistent.
- Do not create separate components differing only by copy, icon, or one color.
- Ensure destructive variants still require the feature-level confirmation policy.

## Required states

Primitives must support appropriate:

- Default, hover, focus-visible, active.
- Disabled and read-only.
- Invalid and described error.
- Loading/busy without changing accessible name unexpectedly.
- High zoom and reduced motion.

## Creation workflow

1. Search the existing catalog and feature components.
2. Extend a compatible component before creating another.
3. Define semantics, keyboard behavior, states, and variants.
4. Build the primitive without feature/form coupling.
5. Add a form adapter only if a form use exists.
6. Add tests and approved component-workbench examples if that tool exists.
7. Verify all current consumers before changing a primitive contract.

## Anti-patterns

- Import form state inside `ui/Input`.
- Style the same input separately in every form adapter.
- Put employee-specific labels inside a generic Select.
- Build a clickable `div` instead of Button/Link.
- Add a primitive without focus, disabled, invalid, and accessible-name behavior.
- Require a component workbench when none is approved.
