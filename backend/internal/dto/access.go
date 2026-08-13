package dto

import "github.com/google/uuid"

// UpdatePermissionRequest memetakan schema UpdatePermissionRequest pada OpenAPI. Field
// `is_allowed` memakai pointer agar body tanpa field tersebut ditolak validasi, bukan
// diperlakukan diam-diam sebagai `false`.
type UpdatePermissionRequest struct {
	RoleID    uuid.UUID `json:"role_id" validate:"required"`
	Module    string    `json:"modul" validate:"required,max=100"`
	Action    string    `json:"aksi" validate:"required,oneof=create read update delete approve export"`
	IsAllowed *bool     `json:"is_allowed" validate:"required"`
}
