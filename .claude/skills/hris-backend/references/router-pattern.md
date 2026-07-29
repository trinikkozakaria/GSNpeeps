# Router pattern

Use `github.com/gorilla/mux` on top of `net/http`. Keep route registration declarative and
dependency injection in `cmd/api`, not inside handlers or route functions.

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
- `POST /api/v1/auth/reset-password`

Authenticated:

- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`
- `PATCH /api/v1/auth/me/password`
- All master, employee, profile, dashboard, attendance, report, leave, overtime, access, and notification operations defined in OpenAPI.

Do not register:

- Refresh-token routes.
- Undocumented reset/change-password routes.
- Notification create routes.
- Audit mutation routes.

## Conceptual registration

```go
func RegisterRoutes(r *mux.Router, d HTTPDependencies) {
    r.HandleFunc("/health", d.Health).Methods(http.MethodGet)

    api := r.PathPrefix("/api/v1").Subrouter()
    api.Handle("/auth/login", d.LoginRateLimit(d.Auth.Login)).Methods(http.MethodPost)
    api.Handle("/auth/reset-password", d.ResetRateLimit(d.Auth.ResetPassword)).
        Methods(http.MethodPost)

    protected := api.NewRoute().Subrouter()
    protected.Use(d.Authenticate)
    protected.HandleFunc("/auth/logout", d.Auth.Logout).Methods(http.MethodPost)
    protected.HandleFunc("/auth/me", d.Auth.Me).Methods(http.MethodGet)
    protected.HandleFunc("/auth/me/password", d.Auth.ChangePassword).Methods(http.MethodPatch)
    protected.Handle("/karyawan", d.Authorize.Roles("hr", "top_management")(d.Employees.List)).
        Methods(http.MethodGet)
    protected.Handle("/karyawan", d.Authorize.Roles("hr")(d.Employees.Create)).
        Methods(http.MethodPost)
    // Register the remaining operations from OpenAPI.
}
```

`ResetRateLimit` is conceptual middleware naming; reuse the established middleware signature.

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
- Exactly 46 operations match OpenAPI.
- Public routes do not require auth.
- Protected routes cannot bypass auth/RBAC.
- No undocumented route appears in the registry.
