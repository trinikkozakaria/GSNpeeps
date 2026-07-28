# Page pattern

# Page pattern

Use this reference for every route-level screen.

## Contents

[Page composition](#page-composition) · [Route specification](#route-specification) ·
[Protected shell](#protected-shell) · [List page](#list-page) ·
[Detail page](#detail-page) · [Create/edit page](#createedit-page) ·
[Dashboard](#dashboard) · [Approval page](#approval-page) ·
[Attendance page](#attendance-page) · [Feedback states](#feedback-states) ·
[Navigation and focus](#navigation-and-focus) · [Anti-patterns](#anti-patterns)

## Page composition

Compose in this order:

1. Parse route and query parameters.
2. Resolve authentication and route capability.
3. Avoid/stop forbidden sensitive requests.
4. Set title and breadcrumbs.
5. Start approved queries.
6. Render loading/error/empty state.
7. Render feature content and permitted actions.
8. Handle mutation feedback and cache coherence.

## Route specification

Before implementation, record:

- Route path and parameters.
- Allowed roles/capabilities.
- Resource/ownership scope.
- API operations.
- URL-managed filters.
- Complete state matrix.
- Mobile and keyboard behavior.
- Test cases.

## Protected shell

Use a shared application shell with sidebar, topbar, user menu, skip link, and main landmark.
Generate navigation from centralized capability metadata. Do not mount protected shell
content until auth resolution completes.

## List page

- Put title, description, and primary action in a consistent header.
- Keep filters/page in URL.
- Use the shared data-table pattern.
- Distinguish empty data from filtered empty.
- Preserve filters through detail navigation.
- Show create/export only when permitted.

## Detail page

- Validate ID before fetching.
- Distinguish `403`, `404`, and soft-deactivated/deleted state.
- Show identity/status before secondary sections.
- Keep edit/delete/approval actions in consistent placement.
- Avoid fetching tabs/sections the user cannot access.
- Preserve originating list state in back navigation.

## Create/edit page

- Use the shared form composition.
- Prevent double submit.
- Preserve recoverable values on errors.
- Map `422` field errors.
- Handle `409` separately from validation.
- Warn for meaningful unsaved changes.

## Dashboard

- Never render fabricated metrics.
- Show metric period/filter and last-updated context.
- Give charts accessible text/table alternatives.
- Keep Top Management read-only.
- Omit Personal Metrics for Top Management.
- Treat partial widget failures explicitly; do not label missing data as zero.

## Approval page

- Show requester, type, dates/duration, document, current stage, and timeline.
- Render only backend-authorized current actions.
- Require a rejection note.
- Disable controls while deciding.
- On conflict, refetch and explain that state changed.
- Never optimistically mark an approval final.

## Attendance page

- Separate permission request, camera preview, captured fallback, location result, and submit.
- Explain camera/GPS purpose before browser prompt.
- Provide fallback upload for camera failure.
- Keep server time/radius authoritative.
- Stop resources on route leave.

## Feedback states

| State | Expected UI |
|---|---|
| Loading | Stable skeleton/status |
| Empty | Meaningful explanation and allowed next action |
| Filtered empty | Active-filter summary and clear action |
| 401 | Session cleanup and safe login redirect |
| 403 | Forbidden page/section without logout |
| 404 | Resource-specific not found |
| 409 | State changed; refetch/recover |
| 422 | Field errors or actionable validation |
| 429 | Lockout/rate-limit guidance |
| Offline/5xx | Retry path without raw internals |

## Navigation and focus

- Set a meaningful document title.
- Move focus to the main heading after route changes when the router does not.
- Restore focus after closing dialogs.
- Validate internal redirect/back targets.
- Avoid scroll loss during small URL filter changes when appropriate.

## Anti-patterns

- Fetch before auth/capability resolution.
- Hide an error by returning an empty page.
- Repeat app shell/layout in each route.
- Put every page in one client-only mega-component.
- Render Coming Soon for in-scope features.
- Render fake dashboard records while waiting for API.
