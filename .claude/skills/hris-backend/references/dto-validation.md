# DTO and validation

DTOs are typed request/response contracts separate from domain and persistence models.
Validation tags and custom validators use `github.com/go-playground/validator/v10`.

## Contents

- Create and partial-update requests
- Query and response DTOs
- Validation layers
- Error format
- Rules

## Create request

Use non-pointer values for required fields and pointers only for genuinely optional values.

```go
type CreateEmployeeRequest struct {
    NIP           string     `json:"nip" validate:"required,max=20"`
    Name          string     `json:"nama" validate:"required,max=150"`
    Email         string     `json:"email" validate:"required,email,max=150"`
    Gender        string     `json:"jenis_kelamin" validate:"required,oneof=L P"`
    BirthDate     Date       `json:"tanggal_lahir" validate:"required"`
    DepartmentID  uuid.UUID  `json:"department_id" validate:"required"`
    PositionID    uuid.UUID  `json:"position_id" validate:"required"`
    SupervisorID  *uuid.UUID `json:"atasan_id"`
    JoinDate      Date       `json:"tanggal_join" validate:"required"`
    Address       AddressRequest `json:"alamat" validate:"required"`
    KTP           KTPRequest     `json:"ktp" validate:"required"`
    Contract      ContractRequest `json:"kontrak" validate:"required"`
}
```

`Date` represents the project's approved date-only type. Do not add request fields absent from OpenAPI.

## Partial update

`PUT /karyawan/{id}` is partial in the API Contract. Use pointers or an optional-field type to distinguish omitted, null, empty, and zero.

```go
type UpdateEmployeeRequest struct {
    Name         *string    `json:"nama" validate:"omitempty,max=150"`
    DepartmentID *uuid.UUID `json:"department_id"`
    PositionID   *uuid.UUID `json:"position_id"`
    SupervisorID OptionalUUID `json:"atasan_id"`
    Status       *string    `json:"status" validate:"omitempty,oneof=aktif nonaktif"`
}
```

Choose one nullable-field convention and use it consistently, especially when a nullable supervisor must be cleared.

## Query DTO

```go
type PageQuery struct {
    Page  int
    Limit int
}

type EmployeeListFilter struct {
    Search       string
    DepartmentID *uuid.UUID
    Status       *string
}
```

Handler parsing must apply defaults, a maximum limit, UUID validation, and sort allowlists if sorting is approved.

## Response DTO

Never serialize persistence entities directly when they contain internal/sensitive fields.

```go
type LoginResponse struct {
    Token     string            `json:"token"`
    ExpiresIn int               `json:"expires_in"`
    Role      string            `json:"role"`
    User      LoginUserResponse `json:"user"`
}

type LoginUserResponse struct {
    ID   uuid.UUID `json:"id"`
    Name string    `json:"nama"`
}
```

Do not include `password_hash`, failed-login counters, account lock internals, Redis values, or permission caches.

## Validation layers

Handler/DTO:

- Required fields, format, length, enum, UUID, date syntax, coordinate range.
- Multipart presence, file size/type.

Service:

- Department-position relationship.
- Supervisor not self and active.
- Uniqueness and conflict.
- Leave quota/balance and date rules.
- Approval actor/stage.
- WFO distance and attendance state.

Database:

- NOT NULL, FK, UNIQUE, CHECK, and concurrency-safe constraints.

## Error format

Map validation to:

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Data belum valid",
    "fields": {
      "email": "Format email tidak valid",
      "tanggal_selesai": "Tanggal selesai harus setelah tanggal mulai"
    }
  }
}
```

Use JSON field paths for nested errors. Messages are Bahasa Indonesia.

## Rules

1. Reject unknown JSON fields unless compatibility policy says otherwise.
2. Bound body and collection sizes.
3. Normalize email/search before use without changing displayed names.
4. Validate file extension, MIME, and signature; client MIME alone is insufficient.
5. Keep OpenAPI schemas synchronized with DTOs.
6. Never use `map[string]any` as a request DTO.
