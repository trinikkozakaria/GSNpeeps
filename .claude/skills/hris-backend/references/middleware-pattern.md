# Middleware pattern

Use the approved HTTP stack with the conceptual signature:

```go
type Middleware func(http.Handler) http.Handler
```

## Recommended order

```text
recovery
-> request-id
-> access-log
-> CORS
-> body-limit / route rate-limit
-> authentication
-> capability guard
-> handler
```

## Typed context

```go
type identityKey struct{}

type Identity struct {
    UserID     uuid.UUID
    EmployeeID uuid.UUID
    Role       string
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
    identity, ok := ctx.Value(identityKey{}).(Identity)
    return identity, ok
}
```

Do not place permission lists in JWT. Permissions are dynamic and must be resolved through the authorization service/cache.

## Authentication

```go
type TokenVerifier interface {
    Verify(ctx context.Context, token string) (Identity, TokenFingerprint, error)
}

type SessionValidator interface {
    Validate(ctx context.Context, userID uuid.UUID, fp TokenFingerprint) error
}
```

Authentication must:

1. Parse `Authorization: Bearer`.
2. Validate algorithm, signature, required claims, and 8-hour expiry.
3. Cross-check Redis `session:<user_id>`.
4. Optionally confirm active employee/account according to the approved policy.
5. Put typed identity in context.
6. Fail closed on invalid/missing session or Redis failure.

## Authorization

Capability middleware handles coarse route access. Services still enforce:

- self ownership.
- direct-report relationship.
- current approval stage.
- HR full access.
- Top Management read-only and HR-request approval exception.

Unknown roles or missing identity fail closed.

## Recovery/request logging

- Recovery catches panic, logs stack internally with request ID, and returns generic `INTERNAL_ERROR`.
- Request ID accepts a valid incoming value or generates one and echoes it.
- Access logging records method, path template, status, duration, and safe actor ID.
- Redact Authorization, cookies, query PII, bodies, and file names when sensitive.

## Rate limiting

- Login lockout is five failed attempts per account and is not replaced by an IP limiter.
- Default authenticated limit is 120 requests/minute/user.
- Store counters in Redis.
- Trust forwarded IP headers only from the configured Nginx proxy boundary.

## Anti-patterns

```go
// Wrong: string context key.
context.WithValue(ctx, "user", identity)

// Wrong: permissions copied into long-lived JWT.
claims.Permissions = allPermissions

// Wrong: Redis unavailable but request continues.
if err != nil { next.ServeHTTP(w, r) }
```

