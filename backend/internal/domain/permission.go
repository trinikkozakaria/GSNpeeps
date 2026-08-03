package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Modul yang dapat diatur melalui matriks permission. Nilai identik dengan kolom `modul`
// pada tabel permissions.
const (
	ModuleEmployee         = "karyawan"
	ModuleDashboard        = "dashboard"
	ModuleAttendance       = "absensi"
	ModuleAttendanceReport = "laporan_kehadiran"
	ModuleLeave            = "ketidakhadiran"
	ModuleOvertime         = "lembur"
	ModuleNotification     = "notifikasi"
	ModuleAccess           = "akses"
	ModuleAudit            = "audit"
)

// Aksi mengikuti enum `Permission.aksi` pada OpenAPI (D-031).
const (
	ActionCreate  = "create"
	ActionRead    = "read"
	ActionUpdate  = "update"
	ActionDelete  = "delete"
	ActionApprove = "approve"
	ActionExport  = "export"
)

// permissionCatalog adalah pasangan modul/aksi yang benar-benar dimiliki produk. Katalog ini
// sama persis dengan baris yang di-seed migration, sehingga update permission tidak dapat
// membuat kapabilitas yang tidak berarti apa pun.
var permissionCatalog = map[string][]string{
	ModuleEmployee:         {ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionExport},
	ModuleDashboard:        {ActionRead},
	ModuleAttendance:       {ActionCreate, ActionRead},
	ModuleAttendanceReport: {ActionRead, ActionExport},
	ModuleLeave:            {ActionCreate, ActionRead, ActionUpdate, ActionApprove},
	ModuleOvertime:         {ActionCreate, ActionRead, ActionApprove},
	ModuleNotification:     {ActionRead, ActionUpdate, ActionDelete},
	ModuleAccess:           {ActionRead, ActionUpdate},
	ModuleAudit:            {ActionRead},
}

// roleDescriptions memberi `deskripsi` pada schema RoleSummary. Kolom deskripsi tidak ada di
// tabel roles, sehingga label ditetapkan di domain agar konsisten di seluruh response.
var roleDescriptions = map[RoleName]string{
	RoleEmployee:      "Melihat data diri, metrik personal, absensi, dan pengajuan sendiri.",
	RoleSupervisor:    "Seluruh kemampuan Karyawan ditambah approval bawahan langsung.",
	RoleHR:            "Mengelola data karyawan, dashboard, monitoring, approval final, dan akses.",
	RoleTopManagement: "Pemantauan read-only seluruh menu dan approval final pengajuan HR.",
}

// KnownCapability menyatakan apakah pasangan modul/aksi ada pada katalog.
func KnownCapability(module, action string) bool {
	return slices.Contains(permissionCatalog[module], action)
}

// RoleDescription mengembalikan deskripsi Bahasa Indonesia satu role.
func RoleDescription(role RoleName) string {
	return roleDescriptions[role]
}

// RoleSummary memetakan schema RoleSummary pada OpenAPI.
type RoleSummary struct {
	ID          uuid.UUID `json:"id"`
	Name        RoleName  `json:"nama"`
	Description string    `json:"deskripsi"`
}

// Permission memetakan schema Permission pada OpenAPI.
type Permission struct {
	ID        uuid.UUID `json:"id"`
	RoleID    uuid.UUID `json:"role_id"`
	Module    string    `json:"modul"`
	Action    string    `json:"aksi"`
	IsAllowed bool      `json:"is_allowed"`
}

// PermissionChange adalah satu perubahan matriks yang diminta HR.
type PermissionChange struct {
	RoleID    uuid.UUID
	Module    string
	Action    string
	IsAllowed bool
}

// ValidatePermissionChange menegakkan invariant produk pada matriks permission.
//
// Aturan yang ditegakkan:
//
//   - Modul/aksi harus ada pada katalog.
//   - Top Management read-only: hanya boleh memegang aksi baca, approval pengajuan HR, dan
//     pengelolaan notifikasinya sendiri. Memberi mutation AKSES kepada Top Management
//     bertentangan dengan kontrak dan ditolak tanpa keputusan produk baru.
//   - Karyawan dan Atasan tidak boleh memegang modul AKSES maupun Audit dalam bentuk apa pun.
//   - HR tidak boleh mencabut aksesnya sendiri ke modul AKSES; pencabutan tersebut mengunci
//     satu-satunya role yang dapat memulihkan matriks.
func ValidatePermissionChange(role RoleName, change PermissionChange) error {
	if !role.Valid() {
		return ErrNotFound
	}
	if !KnownCapability(change.Module, change.Action) {
		return ErrInvalidRequest
	}

	if !change.IsAllowed {
		if role == RoleHR && change.Module == ModuleAccess {
			return ErrPermissionInvariant
		}
		return nil
	}

	switch role {
	case RoleTopManagement:
		if !topManagementMayHold(change.Module, change.Action) {
			return ErrPermissionInvariant
		}
	case RoleEmployee, RoleSupervisor:
		if change.Module == ModuleAccess || change.Module == ModuleAudit {
			return ErrPermissionInvariant
		}
	}
	return nil
}

func topManagementMayHold(module, action string) bool {
	// Notifikasi selalu milik sendiri, sehingga menandai dibaca dan dismiss bukan mutation
	// terhadap data organisasi.
	if module == ModuleNotification {
		return true
	}
	return action == ActionRead || action == ActionApprove
}

// AuditLogEntry memetakan schema AuditLog pada OpenAPI. `UserID` dan `UserName` kosong untuk
// aktor sistem seperti auto-escalation dan scheduler (D-029).
type AuditLogEntry struct {
	ID         uuid.UUID      `json:"id"`
	UserID     *uuid.UUID     `json:"user_id"`
	UserName   *string        `json:"nama_user"`
	Action     string         `json:"aksi"`
	Module     string         `json:"modul"`
	ResourceID *uuid.UUID     `json:"resource_id"`
	Detail     map[string]any `json:"detail"`
	IPAddress  *string        `json:"ip_address"`
	CreatedAt  time.Time      `json:"created_at"`
}

// AuditLogFilter membatasi pembacaan audit log.
type AuditLogFilter struct {
	UserID    *uuid.UUID
	Action    *string
	Module    *string
	StartDate *time.Time
	EndDate   *time.Time
	Page      int
	Limit     int
}

// AuditLogPage adalah hasil pembacaan audit log yang sudah dipaginasi.
type AuditLogPage struct {
	Items []AuditLogEntry
	Page  int
	Limit int
	Total int
}
