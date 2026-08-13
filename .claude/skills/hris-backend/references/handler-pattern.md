# Handler pattern

Handlers are thin `net/http` adapters: parse, validate with go-playground/validator, call one
service method, and write a standard response. Extract path variables with `mux.Vars(r)`.

## Contents

- List/create templates
- Parsing rules
- Multipart and export handling
- Rules and anti-patterns

## Template

```go
type EmployeeHandler struct {
    service EmployeeService
    validate Validator
}

func NewEmployeeHandler(service EmployeeService, validate Validator) *EmployeeHandler {
    return &EmployeeHandler{service: service, validate: validate}
}

func (h *EmployeeHandler) List(w http.ResponseWriter, r *http.Request) {
    page, filter, err := parseEmployeeListQuery(r.URL.Query())
    if err != nil {
        response.Validation(w, err)
        return
    }

    identity, ok := middleware.IdentityFromContext(r.Context())
    if !ok {
        response.Unauthorized(w)
        return
    }

    result, err := h.service.List(r.Context(), identity, page, filter)
    if err != nil {
        response.FromError(r.Context(), w, err)
        return
    }

    response.Page(w, http.StatusOK, result.Items, result.Meta, "OK")
}
```

Create:

```go
func (h *EmployeeHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateEmployeeRequest
    if err := decodeJSON(r, &req); err != nil {
        response.Validation(w, err)
        return
    }
    if err := h.validate.Struct(req); err != nil {
        response.Validation(w, err)
        return
    }

    identity, ok := middleware.IdentityFromContext(r.Context())
    if !ok {
        response.Unauthorized(w)
        return
    }

    out, err := h.service.Create(r.Context(), identity, req)
    if err != nil {
        response.FromError(r.Context(), w, err)
        return
    }
    response.Success(w, http.StatusCreated, out, "Karyawan berhasil ditambahkan")
}
```

## Parsing rules

- Apply a body-size limit before decoding.
- Reject empty bodies, multiple JSON values, and unknown fields.
- Close multipart/file resources reliably.
- Parse UUID/date/boolean/pagination with explicit errors.
- Use bounded `limit`.
- Extract identity only from typed context, never request body/headers other than Bearer auth middleware.

## Multipart and files

- Use the exact OpenAPI field names.
- Enforce 5 MB per file plus reasonable multipart overhead.
- Validate type/signature before service/storage.
- Pass a streaming reader/file metadata to the service; do not load large files unnecessarily.
- Never accept a client-supplied Nextcloud path.

## Export

For XLSX/PDF, set the documented media type and a sanitized `Content-Disposition`, then stream. Do not wrap file bytes in JSON.

## Rules

1. No SQL, Redis, WebDAV, approval routing, or business calculation in handlers.
2. No manual domain-error branching scattered across handlers; use central mapping.
3. Return immediately after writing a response.
4. Do not leak whether another user's resource exists.
5. Test every documented status, content type, and malformed input case.

## Anti-patterns

```go
// Wrong: persistence and business logic in handler.
db.ExecContext(r.Context(), "UPDATE employees ...")

// Wrong: ignores decode error.
_ = json.NewDecoder(r.Body).Decode(&req)

// Wrong: identity supplied by client.
actorID := req.UserID
```
