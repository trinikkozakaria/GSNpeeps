# Response helper

Keep one response package for JSON envelopes and application-error mapping.

## Types

```go
type SuccessBody struct {
    Success bool   `json:"success"`
    Data    any    `json:"data"`
    Message string `json:"message,omitempty"`
    Meta    *PaginationMeta `json:"meta,omitempty"`
}

type PaginationMeta struct {
    Page       int   `json:"page"`
    Limit      int   `json:"limit"`
    TotalData  int64 `json:"total_data"`
    TotalPage  int   `json:"total_page"`
}

type ErrorBody struct {
    Success bool      `json:"success"`
    Error   ErrorInfo `json:"error"`
}

type ErrorInfo struct {
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Fields  map[string]string `json:"fields,omitempty"`
}
```

## Helpers

Provide the project equivalents of:

```go
Success(w, status, data, message)
Page(w, status, data, meta, message)
Error(w, status, code, message, fields)
FromError(ctx, w, err)
```

The writer must set JSON content type, write status once, encode once, and handle encoding failure through logging rather than a second response.

## Central error catalog

Map application errors with `errors.Is`/typed inspection:

| Application error | HTTP | Code |
|---|---:|---|
| invalid credentials | 401 | `INVALID_CREDENTIALS` |
| account locked | 429 | `ACCOUNT_LOCKED` |
| unauthorized/session invalid | 401 | `UNAUTHORIZED` |
| forbidden | 403 | `FORBIDDEN` |
| employee/resource missing | 404 | `NOT_FOUND` |
| duplicate identity | 409 | `CONFLICT` |
| already decided | 409 | `ALREADY_DECIDED` |
| out of WFO radius | 422 | `OUT_OF_RADIUS` |
| duplicate check-in | 422 | `DUPLICATE_CHECKIN` |
| insufficient leave | 422 | `INSUFFICIENT_LEAVE_BALANCE` |
| file too large | 413 | `FILE_TOO_LARGE` |
| unsupported file | 415 | `UNSUPPORTED_FILE_TYPE` |
| default unknown | 500 | `INTERNAL_ERROR` |

Use the exact endpoint-specific OpenAPI mapping; this table is a routing guide, not permission to add codes.

## Validation

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Data belum valid",
    "fields": {
      "tanggal_mulai": "Tanggal mulai wajib diisi"
    }
  }
}
```

## File responses

XLSX/PDF exports are streamed with their documented media type and safe filename; do not wrap them in JSON.

## Rules

- Never expose internal error strings or stack traces.
- Log unknown errors once with request ID.
- Never serialize persistence models containing sensitive fields.
- Test every catalog entry, validation fields, pagination calculations, and encoding behavior.

