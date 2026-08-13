package domain

import (
	"time"

	"github.com/google/uuid"
)

// TimeLayout adalah format jam kontrak (JSON Schema `format: time`).
const TimeLayout = "15:04:05"

// OvertimeRequestSummary memetakan schema OvertimeRequestSummary pada OpenAPI. Nama wire
// `waktu_mulai`, `waktu_selesai`, dan `total_jam` dipetakan dari kolom `jam_mulai`,
// `jam_selesai`, dan `durasi_jam`.
type OvertimeRequestSummary struct {
	ID           uuid.UUID     `json:"id"`
	EmployeeID   uuid.UUID     `json:"employee_id"`
	EmployeeName string        `json:"nama_karyawan"`
	Date         string        `json:"tanggal"`
	StartTime    string        `json:"waktu_mulai"`
	EndTime      string        `json:"waktu_selesai"`
	TotalHours   float64       `json:"total_jam"`
	Status       RequestStatus `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
}

type OvertimeRequestDetail struct {
	OvertimeRequestSummary
	Reason          string            `json:"alasan"`
	DocumentURL     *string           `json:"dokumen_url"`
	ApprovalHistory []ApprovalHistory `json:"approval_history"`
}

type OvertimeRequestPage struct {
	Items []OvertimeRequestSummary
	Total int
	Page  int
	Limit int
}

// CreateOvertimeRequest adalah perintah pengajuan lembur. Dokumen pendukung opsional,
// berbeda dengan ketidakhadiran.
type CreateOvertimeRequest struct {
	UserID    uuid.UUID
	Date      string
	StartTime string
	EndTime   string
	Reason    string
	Document  *UploadedFile
}

// OvertimeRequestRow adalah baris yang siap disimpan. Durasi tidak disertakan karena
// dihitung database sebagai generated column.
type OvertimeRequestRow struct {
	UserID      uuid.UUID
	Date        string
	StartTime   string
	EndTime     string
	Reason      string
	DocumentURL *string
	Status      RequestStatus
}

type OvertimeRequestFilter struct {
	Scope  LeaveRequestScope
	Status *RequestStatus
	Start  *string
	End    *string
	Page   int
	Limit  int
}

// OvertimeRecapItem memetakan schema OvertimeRecapItem. Kompensasi lembur tidak dihitung
// di sini; PRD menetapkan perhitungan dilakukan manual di luar sistem.
type OvertimeRecapItem struct {
	EmployeeID   uuid.UUID `json:"employee_id"`
	EmployeeName string    `json:"nama_karyawan"`
	Department   string    `json:"departemen"`
	TotalRequest int       `json:"total_pengajuan"`
	TotalHours   float64   `json:"total_jam"`
}

type OvertimeRecapFilter struct {
	Start        *string
	End          *string
	DepartmentID *uuid.UUID
}
