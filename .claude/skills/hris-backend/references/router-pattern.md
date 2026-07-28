# Router pattern

Use the approved router. Keep route registration declarative and dependency injection in `cmd/api`, not inside handlers or route functions.

## Dependencies

```go
type HTTPDependencies struct {
    Auth        *AuthHandler
    Employees   *EmployeeHandler
    Profile     *ProfileHandler
    Dashboard   *DashboardHandler
    Attendance  *AttendanceHandler
    Reports     *ReportHandler
    Leave       *LeaveHandler
    Overtime    *OvertimeHandler
    Access      *AccessHandler
    Notification *NotificationHandler

    Authenticate Middleware
    Authorize    Authorizer
}
```

Include only handlers that exist. Do not register nil placeholders.

## Route groups

Public:

- `GET /health`
- `POST /api/v1/auth/login`

Authenticated:

- `POST /api/v1/auth/logout`
- All master, employee, profile, dashboard, attendance, report, leave, overtime, access, and notification operations defined in OpenAPI.

Do not register:

- Refresh-token routes.
- Undocumented reset/change-password routes.
- Notification create routes.
- Audit mutation routes.

## Conceptual registration

```go
func RegisterRoutes(r Router, d HTTPDependencies) {
    r.GET("/health", d.Health)

    api := r.Group("/api/v1")
    api.POST("/auth/login", d.Auth.Login, d.LoginRateLimit)
    api.POST("/auth/logout", d.Auth.Logout, d.Authenticate)

    protected := api.Group("", d.Authenticate)
    protected.GET("/karyawan", d.Employees.List, d.Authorize.Roles("hr", "top_management"))
    protected.POST("/karyawan", d.Employees.Create, d.Authorize.Roles("hr"))
    // Register the remaining operations from OpenAPI.
}
```

Adapt syntax to the approved router. Route policy is unchanged.

## Authorization

- Attach coarse role/capability guards at route registration.
- Keep ownership, direct-report, recipient, and approval-stage enforcement in services/repositories.
- Top Management read routes and HR-request decisions must be explicit.
- Unknown roles fail closed.

## Global middleware

Wrap in the order documented by `middleware-pattern.md`. Public endpoints still receive recovery, request ID, logging, CORS, and relevant rate/body limits.

## Server

Use `http.Server` or the approved equivalent with read header, read, write, idle, and shutdown timeouts. Handle SIGINT/SIGTERM and close resources gracefully.

## Verification

- Every route specifies method and path.
- Wrong methods return the intended 405 behavior.
- Exactly 42 operations match OpenAPI.
- Public routes do not require auth.
- Protected routes cannot bypass auth/RBAC.
- No undocumented route appears in the registry.

