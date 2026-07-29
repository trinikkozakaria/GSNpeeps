package domain

import "github.com/google/uuid"

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

type EmployeeDetail struct {
	EmployeeSummary
	Gender        string             `json:"jenis_kelamin"`
	BirthDate     string             `json:"tanggal_lahir"`
	JoinDate      string             `json:"tanggal_join"`
	DepartmentID  *uuid.UUID         `json:"department_id"`
	PositionID    *uuid.UUID         `json:"position_id"`
	SupervisorID  *uuid.UUID         `json:"atasan_id"`
	MaritalStatus *string            `json:"status_pernikahan"`
	Address       *EmployeeAddress   `json:"alamat"`
	KTP           *EmployeeKTP       `json:"ktp"`
	Contracts     []EmployeeContract `json:"kontrak"`
}

type EmployeeFilter struct {
	Search       string
	DepartmentID *uuid.UUID
	Status       string
	Page         int
	Limit        int
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
}

type EmployeeMutationResult struct {
	EmployeeID uuid.UUID
	UserID     *uuid.UUID
}

type CreateEmployee struct {
	NIP           string
	Name          string
	Email         string
	PasswordHash  string
	Gender        string
	BirthDate     string
	JoinDate      string
	DepartmentID  uuid.UUID
	PositionID    uuid.UUID
	SupervisorID  *uuid.UUID
	MaritalStatus *string
	Role          RoleName
	Address       EmployeeAddress
	KTPNumber     string
	Contract      CreateEmployeeContract
}

type CreateEmployeeContract struct {
	Number    string
	Type      string
	StartDate string
	EndDate   string
}
