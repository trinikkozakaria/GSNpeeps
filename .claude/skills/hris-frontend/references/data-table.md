# Data table

Use this reference for employee, attendance, approval, notification, role, permission, and
audit-log lists.

## Contents

[Ownership](#ownership) · [URL state](#url-state) ·
[Table composition](#table-composition) · [Required states](#required-states) ·
[Columns](#columns) · [Row actions and RBAC](#row-actions-and-rbac) ·
[Responsive behavior](#responsive-behavior) · [Pagination](#pagination) ·
[Search and filters](#search-and-filters) · [Testing](#testing)

## Ownership

Treat the backend as authoritative for pagination, filtering, and supported sorting. Use:

```text
meta.page
meta.limit
meta.total_data
meta.total_page
```

Do not download all records for client-side filtering or pagination.

## URL state

Store shareable state in query parameters:

```text
page
limit
search
sort (only when contract supports it)
feature-specific filters
```

- Parse and validate URL values.
- Reset `page` to 1 when search/filter/limit changes.
- Remove default/empty parameters consistently.
- Preserve filters through detail/back navigation.
- Debounce search and cancel stale requests.

## Table composition

Separate:

- Toolbar/search/filter controls.
- Column definitions.
- Table renderer.
- Pagination.
- Row action menu.
- Loading/empty/error states.

Keep feature-specific cells and actions in the feature. Keep the generic renderer unaware
of employee roles and API endpoints.

## Required states

- Initial skeleton with stable column layout.
- Page transition without misleading flicker.
- Empty dataset with useful primary action where allowed.
- Filtered empty with clear-filters action.
- Error with status-appropriate message and retry.
- Read-only action state.
- Partial/stale state after concurrency conflict followed by refetch.

Do not turn an error into "Tidak ada data".

## Columns

- Use visible, concise Bahasa Indonesia headers.
- Keep row/header semantics intact.
- Define sort affordance only for server-supported fields.
- Communicate sort direction by text/ARIA, not icon alone.
- Use stable row IDs.
- Format dates/numbers centrally.
- Show statuses with text/icon plus color.
- Do not fetch or render sensitive hidden columns.

## Row actions and RBAC

- Derive actions from centralized capabilities and current allowed resource actions.
- Karyawan/Atasan receive only scoped actions returned by the API.
- Top Management receives read-only actions except the documented HR-request decision case.
- Disable actions during mutation.
- Confirm destructive/high-impact actions.
- Refetch after `409 ALREADY_DECIDED`.

## Responsive behavior

Choose deliberately per dataset:

- Horizontal scroll with a labelled region and sticky identity/action columns.
- Priority columns plus accessible detail disclosure.
- Card/list layout on small screens.

Do not squeeze every column into unreadable widths. Preserve action names and data
relationships at 200% zoom.

## Pagination

- Support previous/next and first/last only when useful.
- Announce current page and visible item range.
- Disable unavailable controls.
- Keep focus predictable after page changes.
- Scroll/focus to table heading when appropriate.
- Handle deletion of the final row on the last page by navigating to the prior valid page.

## Search and filters

- Provide visible labels or accessible names.
- Show active filters and a reset action.
- Debounce around the approved UX interval; do not hardcode a library dependency.
- Cancel superseded requests.
- Avoid firing one request per keystroke.
- Validate date ranges and enum values before request.

## Testing

- URL parse/serialization and page reset.
- Debounce and cancellation.
- Empty versus filtered-empty versus error.
- Server metadata drives pagination.
- Role/action matrix.
- Responsive strategy and keyboard access.
- `409` refetch behavior.
- No hidden sensitive-field request.
