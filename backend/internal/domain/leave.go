package domain

import (
	"time"

	"github.com/google/uuid"
)

// LeaveType memetakan schema LeaveType pada OpenAPI (D-023).
type LeaveType struct {
	ID               uuid.UUID `json:"id"`
	Code             string    `json:"kode"`
	Name             string    `json:"nama"`
	AnnualQuota      int       `json:"kuota_tahunan"`
	RequiresDocument bool      `json:"memerlukan_dokumen"`
	IsActive         bool      `json:"is_active"`
}

type CreateLeaveType struct {
	Code             string
	Name             string
	AnnualQuota      int
	RequiresDocument bool
}

type UpdateLeaveType struct {
	Name             *string
	AnnualQuota      *int
	RequiresDocument *bool
	IsActive         *bool
}

func (u UpdateLeaveType) Empty() bool {
	return u.Name == nil && u.AnnualQuota == nil && u.RequiresDocument == nil && u.IsActive == nil
}

// LeaveBalance adalah saldo cuti per user per tahun. `Remaining` adalah kolom generated.
type LeaveBalance struct {
	UserID    uuid.UUID
	Year      int
	Opening   int
	Used      int
	Remaining int
}

// LeaveRequestSummary memetakan schema LeaveRequestSummary.
type LeaveRequestSummary struct {
	ID           uuid.UUID     `json:"id"`
	EmployeeID   uuid.UUID     `json:"employee_id"`
	EmployeeName string        `json:"nama_karyawan"`
	LeaveType    string        `json:"jenis_izin"`
	StartDate    string        `json:"tanggal_mulai"`
	EndDate      string        `json:"tanggal_selesai"`
	TotalDays    int           `json:"jumlah_hari"`
	Status       RequestStatus `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
}

// LeaveRequestDetail menambahkan alasan, dokumen, dan riwayat approval.
// `keterangan_lokasi` adalah nama wire untuk kolom `keperluan_tugas` (D-024).
type LeaveRequestDetail struct {
	LeaveRequestSummary
	Reason          string            `json:"alasan"`
	DocumentURL     *string           `json:"dokumen_url"`
	Destination     *string           `json:"lokasi_tujuan"`
	DestinationNote *string           `json:"keterangan_lokasi"`
	ApprovalHistory []ApprovalHistory `json:"approval_history"`
}

type LeaveRequestPage struct {
	Items []LeaveRequestSummary
	Total int
	Page  int
	Limit int
}

// CreateLeaveRequest adalah perintah pengajuan ketidakhadiran yang sudah tervalidasi handler.
type CreateLeaveRequest struct {
	UserID          uuid.UUID
	LeaveTypeID     uuid.UUID
	StartDate       string
	EndDate         string
	Reason          string
	Destination     *string
	DestinationNote *string
	Document        *UploadedFile
}

// LeaveRequestRow adalah baris yang siap disimpan repository.
type LeaveRequestRow struct {
	UserID          uuid.UUID
	LeaveTypeID     uuid.UUID
	StartDate       string
	EndDate         string
	TotalDays       int
	Reason          string
	DocumentURL     *string
	Destination     *string
	DestinationNote *string
	Status          RequestStatus
}

// UploadedFile adalah berkas multipart yang sudah divalidasi tipe dan ukurannya.
type UploadedFile struct {
	FileName  string
	Extension string
	MediaType string
	Content   []byte
}

// LeaveRequestScope membatasi baris yang boleh dibaca satu identitas.
type LeaveRequestScope struct {
	// RequesterUserID membatasi ke pengajuan milik sendiri.
	RequesterUserID *uuid.UUID
	// SupervisorEmployeeID membatasi ke bawahan langsung pada tahap atasan.
	SupervisorEmployeeID *uuid.UUID
	// Stage membatasi ke status menunggu tertentu.
	Stage *RequestStatus
	// RequesterRole membatasi ke pengajuan milik role tertentu, dipakai Top Management
	// yang hanya boleh melihat pengajuan HR.
	RequesterRole *RoleName
}

type LeaveRequestFilter struct {
	Scope  LeaveRequestScope
	Status *RequestStatus
	Page   int
	Limit  int
}

// TotalLeaveDays menghitung jumlah hari kalender inklusif pada rentang pengajuan.
// Kalender libur nasional belum tersedia (G-010) sehingga seluruh hari dihitung.
func TotalLeaveDays(start, end time.Time) int {
	return int(end.Sub(start).Hours()/24) + 1
}
