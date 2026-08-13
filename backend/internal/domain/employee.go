package domain

import (
	"time"

	"github.com/google/uuid"
)

type Department struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"nama"`
}

type Position struct {
	ID           uuid.UUID `json:"id"`
	DepartmentID uuid.UUID `json:"department_id"`
	Name         string    `json:"nama"`
}

type EmployeeSummary struct {
	ID         uuid.UUID `json:"id"`
	NIP        string    `json:"nip"`
	Name       string    `json:"nama"`
	Email      string    `json:"email"`
	Department string    `json:"departemen"`
	Position   string    `json:"jabatan"`
	Status     string    `json:"status"`
}

type EmployeeAddress struct {
	Street   string  `json:"jalan"`
	Village  *string `json:"kelurahan"`
	District *string `json:"kecamatan"`
	City     string  `json:"kota"`
	Province string  `json:"provinsi"`
}

type EmployeeKTP struct {
	Number  string  `json:"nomor_ktp"`
	FileURL *string `json:"file_url"`
}

type EmployeeContract struct {
	Number    string  `json:"nomor_kontrak"`
	StartDate string  `json:"tanggal_mulai"`
	EndDate   string  `json:"tanggal_berakhir"`
	Type      string  `json:"jenis_kontrak"`
	FileURL   *string `json:"file_url"`
	Status    string  `json:"status"`
}

// EmployeeBPJS memetakan satu nomor kepesertaan BPJS. Satu employee menyimpan satu baris
// `employee_bpjs`, tetapi kontrak mengirimkannya sebagai koleksi per jenis kepesertaan.
type EmployeeBPJS struct {
	Type   string `json:"jenis"`
	Number string `json:"nomor"`
}

type EmployeeNPWP struct {
	Number  string  `json:"nomor_npwp"`
	FileURL *string `json:"file_url"`
}

type EmergencyContact struct {
	Name         string  `json:"nama"`
	Relationship *string `json:"hubungan"`
	Phone        string  `json:"nomor_telepon"`
}

type EducationHistory struct {
	Level          *string `json:"jenjang"`
	Institution    *string `json:"institusi"`
	EntryYear      *int    `json:"tahun_masuk"`
	GraduationYear *int    `json:"tahun_lulus"`
}

type PositionHistory struct {
	Department *Department `json:"departemen,omitempty"`
	Position   *Position   `json:"jabatan,omitempty"`
	StartDate  string      `json:"tanggal_mulai"`
	EndDate    *string     `json:"tanggal_selesai"`
}

// CurrentSalary hanya berisi periode bulan berjalan sesuai PRD; histori gaji penuh tidak
// pernah dikirim melalui endpoint detail maupun profil.
type CurrentSalary struct {
	Period      string  `json:"periode"`
	BasePay     float64 `json:"gaji_pokok"`
	Allowance   float64 `json:"tunjangan"`
	Deduction   float64 `json:"potongan"`
	TakeHomePay float64 `json:"take_home_pay"`
}

type EmployeeDocument struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"jenis_dokumen"`
	FileName  string    `json:"nama_file"`
	FileURL   string    `json:"file_url"`
	CreatedAt time.Time `json:"created_at"`
}

type NewEmployeeDocument struct {
	EmployeeID uuid.UUID
	Type       string
	FileName   string
	FileURL    string
}

type EmployeeDetail struct {
	EmployeeSummary
	PhotoURL          *string            `json:"foto_profil_url"`
	Gender            string             `json:"jenis_kelamin"`
	BirthDate         string             `json:"tanggal_lahir"`
	JoinDate          string             `json:"tanggal_join"`
	DepartmentID      *uuid.UUID         `json:"department_id"`
	PositionID        *uuid.UUID         `json:"position_id"`
	SupervisorID      *uuid.UUID         `json:"atasan_id"`
	MaritalStatus     *string            `json:"status_pernikahan"`
	Address           *EmployeeAddress   `json:"alamat"`
	KTP               *EmployeeKTP       `json:"ktp"`
	Contracts         []EmployeeContract `json:"kontrak"`
	BPJS              []EmployeeBPJS     `json:"bpjs"`
	NPWP              *EmployeeNPWP      `json:"npwp,omitempty"`
	EmergencyContacts []EmergencyContact `json:"kontak_darurat"`
	Education         []EducationHistory `json:"pendidikan"`
	PositionHistory   []PositionHistory  `json:"riwayat_jabatan"`
	CurrentSalary     *CurrentSalary     `json:"gaji_berjalan,omitempty"`
}

type EmployeeFilter struct {
	Search       string
	DepartmentID *uuid.UUID
	Status       string
	Page         int
	Limit        int
}

// ExportFormat adalah format berkas export yang disetujui kontrak; lihat keputusan D-017.
type ExportFormat string

const (
	ExportFormatXLSX ExportFormat = "xlsx"
	ExportFormatPDF  ExportFormat = "pdf"
)

func (f ExportFormat) Valid() bool {
	return f == ExportFormatXLSX || f == ExportFormatPDF
}

// EmployeeExportQuery memakai filter yang sama dengan list, ditambah `id` opsional untuk
// mengekspor satu karyawan.
type EmployeeExportQuery struct {
	Format     ExportFormat
	EmployeeID *uuid.UUID
	Filter     EmployeeFilter
}

// ExportFile adalah berkas hasil export yang siap di-stream ke client.
type ExportFile struct {
	FileName    string
	ContentType string
	Content     []byte
}

type EmployeePage struct {
	Items []EmployeeSummary
	Total int
	Page  int
	Limit int
}

type EmployeeChanges struct {
	Name          *string
	Email         *string
	Gender        *string
	BirthDate     *string
	JoinDate      *string
	DepartmentID  *uuid.UUID
	PositionID    *uuid.UUID
	SupervisorID  *uuid.UUID
	SupervisorSet bool
	MaritalStatus *string
	Status        *string
	Role          *RoleName
	// Field detail berikut memakai pointer (objek/slice) sebagai penanda "disertakan pada
	// request"; nil berarti tidak diubah. Slice yang disertakan (termasuk slice kosong)
	// menggantikan seluruh baris lama untuk employee tersebut (replace-all semantics).
	BPJS              *CreateEmployeeBPJS
	NPWP              *CreateEmployeeNPWP
	EmergencyContacts *[]CreateEmergencyContact
	Education         *[]CreateEducation
	PositionHistory   *[]CreatePositionHistory
	CurrentSalary     *CreateCurrentSalary
}

type EmployeeMutationResult struct {
	EmployeeID uuid.UUID
	UserID     *uuid.UUID
}

type CreateEmployee struct {
	NIP               string
	Name              string
	Email             string
	PasswordHash      string
	Gender            string
	BirthDate         string
	JoinDate          string
	DepartmentID      uuid.UUID
	PositionID        uuid.UUID
	SupervisorID      *uuid.UUID
	MaritalStatus     *string
	Role              RoleName
	Address           EmployeeAddress
	KTPNumber         string
	Contract          CreateEmployeeContract
	BPJS              *CreateEmployeeBPJS
	NPWP              *CreateEmployeeNPWP
	EmergencyContacts []CreateEmergencyContact
	Education         []CreateEducation
	PositionHistory   []CreatePositionHistory
	CurrentSalary     *CreateCurrentSalary
}

type CreateEmployeeContract struct {
	Number    string
	Type      string
	StartDate string
	EndDate   string
}

// CreateEmployeeBPJS menulis satu baris `employee_bpjs`; kedua nomor bersifat opsional dan
// independen, selaras dengan bentuk penyimpanan (satu baris per employee).
type CreateEmployeeBPJS struct {
	HealthNumber     *string
	EmploymentNumber *string
}

type CreateEmployeeNPWP struct {
	Number string
}

type CreateEmergencyContact struct {
	Name         string
	Relationship *string
	Phone        string
}

type CreateEducation struct {
	Level          *string
	Institution    *string
	EntryYear      *int
	GraduationYear *int
}

// CreatePositionHistory menerima referensi department/position; label `jabatan` yang
// tersimpan diturunkan dari posisi pada saat penulisan agar tetap akurat sebagai snapshot
// historis meski posisi berubah nama di kemudian hari.
type CreatePositionHistory struct {
	DepartmentID *uuid.UUID
	PositionID   *uuid.UUID
	StartDate    string
	EndDate      *string
}

type CreateCurrentSalary struct {
	Period    string
	BasePay   float64
	Allowance float64
	Deduction float64
}
