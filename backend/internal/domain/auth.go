package domain

import (
	"time"

	"github.com/google/uuid"
)

type RoleName string

const (
	RoleEmployee      RoleName = "karyawan"
	RoleSupervisor    RoleName = "atasan"
	RoleHR            RoleName = "hr"
	RoleTopManagement RoleName = "top_management"
)

func (r RoleName) Valid() bool {
	switch r {
	case RoleEmployee, RoleSupervisor, RoleHR, RoleTopManagement:
		return true
	default:
		return false
	}
}

type LoginAccount struct {
	ID               uuid.UUID
	EmployeeID       uuid.UUID
	RoleID           uuid.UUID
	Name             string
	Email            string
	PasswordHash     string
	Role             RoleName
	EmployeeStatus   string
	EmployeeDeleted  *time.Time
	FailedLoginCount int
	AccountLocked    bool
	PhotoURL         *string
}

func (a LoginAccount) Active() bool {
	return a.EmployeeStatus == "aktif" && a.EmployeeDeleted == nil
}

type AuthUser struct {
	ID         uuid.UUID `json:"id"`
	EmployeeID uuid.UUID `json:"employee_id"`
	Name       string    `json:"nama"`
	Email      string    `json:"email"`
	Role       RoleName  `json:"role"`
	PhotoURL   *string   `json:"foto_profil_url"`
}

type Identity struct {
	UserID     uuid.UUID
	EmployeeID uuid.UUID
	Role       RoleName
}

type AuditEntry struct {
	UserID    *uuid.UUID
	Action    string
	Module    string
	DataID    *uuid.UUID
	Detail    map[string]any
	IPAddress string
	CreatedAt time.Time
}
