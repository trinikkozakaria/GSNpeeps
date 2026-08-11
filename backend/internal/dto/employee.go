package dto

import (
	"bytes"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

type OptionalUUID struct {
	Value *uuid.UUID
	Set   bool
}

type EmployeeAddressRequest struct {
	Street   string  `json:"jalan" validate:"required,min=1,max=255"`
	Village  *string `json:"kelurahan" validate:"omitempty,max=100"`
	District *string `json:"kecamatan" validate:"omitempty,max=100"`
	City     string  `json:"kota" validate:"required,min=1,max=100"`
	Province string  `json:"provinsi" validate:"required,min=1,max=100"`
}

type EmployeeKTPRequest struct {
	Number string `json:"nomor_ktp" validate:"required,len=16,numeric"`
}

type EmployeeContractRequest struct {
	Number    string `json:"nomor_kontrak" validate:"required,min=1,max=50"`
	Type      string `json:"jenis_kontrak" validate:"required,oneof=PKWT PKWTT"`
	StartDate string `json:"tanggal_mulai" validate:"required,datetime=2006-01-02"`
	EndDate   string `json:"tanggal_berakhir" validate:"required,datetime=2006-01-02"`
}

// EmployeeBPJSRequest menulis satu baris `employee_bpjs`; kedua nomor opsional dan
// independen karena satu employee hanya memiliki satu baris BPJS.
type EmployeeBPJSRequest struct {
	HealthNumber     *string `json:"nomor_kesehatan" validate:"omitempty,max=20"`
	EmploymentNumber *string `json:"nomor_ketenagakerjaan" validate:"omitempty,max=20"`
}

type EmployeeNPWPRequest struct {
	Number string `json:"nomor_npwp" validate:"required,min=1,max=25"`
}

type EmergencyContactRequest struct {
	Name         string  `json:"nama" validate:"required,min=1,max=150"`
	Relationship *string `json:"hubungan" validate:"omitempty,max=50"`
	Phone        string  `json:"nomor_telepon" validate:"required,min=1,max=20"`
}

type EducationRequest struct {
	Level          *string `json:"jenjang" validate:"omitempty,max=20"`
	Institution    *string `json:"institusi" validate:"omitempty,max=150"`
	GraduationYear *int    `json:"tahun_lulus" validate:"omitempty,min=1900,max=2200"`
}

// PositionHistoryRequest mengacu ke department/position master; label diturunkan server
// dari posisi saat penulisan, bukan dikirim client.
type PositionHistoryRequest struct {
	DepartmentID *uuid.UUID `json:"department_id"`
	PositionID   *uuid.UUID `json:"position_id"`
	StartDate    string     `json:"tanggal_mulai" validate:"required,datetime=2006-01-02"`
	EndDate      *string    `json:"tanggal_selesai" validate:"omitempty,datetime=2006-01-02"`
}

type CurrentSalaryRequest struct {
	Period    string  `json:"periode" validate:"required,datetime=2006-01"`
	BasePay   float64 `json:"gaji_pokok" validate:"required,min=0"`
	Allowance float64 `json:"tunjangan" validate:"omitempty,min=0"`
	Deduction float64 `json:"potongan" validate:"omitempty,min=0"`
}

type CreateEmployeeRequest struct {
	NIP               string                    `json:"nip" validate:"required,min=1,max=20"`
	Name              string                    `json:"nama" validate:"required,min=1,max=150"`
	Email             string                    `json:"email" validate:"required,email,max=150"`
	Gender            string                    `json:"jenis_kelamin" validate:"required,oneof=L P"`
	BirthDate         string                    `json:"tanggal_lahir" validate:"required,datetime=2006-01-02"`
	JoinDate          string                    `json:"tanggal_join" validate:"required,datetime=2006-01-02"`
	DepartmentID      uuid.UUID                 `json:"department_id" validate:"required"`
	PositionID        uuid.UUID                 `json:"position_id" validate:"required"`
	SupervisorID      *uuid.UUID                `json:"atasan_id"`
	MaritalStatus     *string                   `json:"status_pernikahan" validate:"omitempty,oneof=lajang menikah cerai"`
	Role              domain.RoleName           `json:"role" validate:"required,oneof=karyawan atasan hr top_management"`
	Address           EmployeeAddressRequest    `json:"alamat" validate:"required"`
	KTP               EmployeeKTPRequest        `json:"ktp" validate:"required"`
	Contract          EmployeeContractRequest   `json:"kontrak" validate:"required"`
	BPJS              *EmployeeBPJSRequest      `json:"bpjs" validate:"omitempty"`
	NPWP              *EmployeeNPWPRequest      `json:"npwp" validate:"omitempty"`
	EmergencyContacts []EmergencyContactRequest `json:"kontak_darurat" validate:"omitempty,dive"`
	Education         []EducationRequest        `json:"pendidikan" validate:"omitempty,dive"`
	PositionHistory   []PositionHistoryRequest  `json:"riwayat_jabatan" validate:"omitempty,dive"`
	CurrentSalary     *CurrentSalaryRequest     `json:"gaji_berjalan" validate:"omitempty"`
}

func (value *OptionalUUID) UnmarshalJSON(data []byte) error {
	value.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		value.Value = nil
		return nil
	}
	var parsed uuid.UUID
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	value.Value = &parsed
	return nil
}

type UpdateEmployeeRequest struct {
	Name          *string          `json:"nama" validate:"omitempty,min=1,max=150"`
	Email         *string          `json:"email" validate:"omitempty,email,max=150"`
	Gender        *string          `json:"jenis_kelamin" validate:"omitempty,oneof=L P"`
	BirthDate     *string          `json:"tanggal_lahir" validate:"omitempty,datetime=2006-01-02"`
	JoinDate      *string          `json:"tanggal_join" validate:"omitempty,datetime=2006-01-02"`
	DepartmentID  *uuid.UUID       `json:"department_id"`
	PositionID    *uuid.UUID       `json:"position_id"`
	SupervisorID  OptionalUUID     `json:"atasan_id"`
	MaritalStatus *string          `json:"status_pernikahan" validate:"omitempty,oneof=lajang menikah cerai"`
	Status        *string          `json:"status" validate:"omitempty,oneof=aktif nonaktif"`
	Role          *domain.RoleName `json:"role" validate:"omitempty,oneof=karyawan atasan hr top_management"`
	// Field detail berikut mengganti seluruh baris lama saat disertakan (replace-all);
	// pointer nil berarti bagian tersebut tidak diubah sama sekali.
	BPJS              *EmployeeBPJSRequest       `json:"bpjs" validate:"omitempty"`
	NPWP              *EmployeeNPWPRequest       `json:"npwp" validate:"omitempty"`
	EmergencyContacts *[]EmergencyContactRequest `json:"kontak_darurat" validate:"omitempty,dive"`
	Education         *[]EducationRequest        `json:"pendidikan" validate:"omitempty,dive"`
	PositionHistory   *[]PositionHistoryRequest  `json:"riwayat_jabatan" validate:"omitempty,dive"`
	CurrentSalary     *CurrentSalaryRequest      `json:"gaji_berjalan" validate:"omitempty"`
}

func (request UpdateEmployeeRequest) Empty() bool {
	return request.Name == nil &&
		request.Email == nil &&
		request.Gender == nil &&
		request.BirthDate == nil &&
		request.JoinDate == nil &&
		request.DepartmentID == nil &&
		request.PositionID == nil &&
		!request.SupervisorID.Set &&
		request.MaritalStatus == nil &&
		request.Status == nil &&
		request.Role == nil &&
		request.BPJS == nil &&
		request.NPWP == nil &&
		request.EmergencyContacts == nil &&
		request.Education == nil &&
		request.PositionHistory == nil &&
		request.CurrentSalary == nil
}
