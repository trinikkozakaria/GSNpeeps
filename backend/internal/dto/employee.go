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

type CreateEmployeeRequest struct {
	NIP           string                  `json:"nip" validate:"required,min=1,max=20"`
	Name          string                  `json:"nama" validate:"required,min=1,max=150"`
	Email         string                  `json:"email" validate:"required,email,max=150"`
	Gender        string                  `json:"jenis_kelamin" validate:"required,oneof=L P"`
	BirthDate     string                  `json:"tanggal_lahir" validate:"required,datetime=2006-01-02"`
	JoinDate      string                  `json:"tanggal_join" validate:"required,datetime=2006-01-02"`
	DepartmentID  uuid.UUID               `json:"department_id" validate:"required"`
	PositionID    uuid.UUID               `json:"position_id" validate:"required"`
	SupervisorID  *uuid.UUID              `json:"atasan_id"`
	MaritalStatus *string                 `json:"status_pernikahan" validate:"omitempty,oneof=lajang menikah cerai"`
	Role          domain.RoleName         `json:"role" validate:"required,oneof=karyawan atasan hr top_management"`
	Address       EmployeeAddressRequest  `json:"alamat" validate:"required"`
	KTP           EmployeeKTPRequest      `json:"ktp" validate:"required"`
	Contract      EmployeeContractRequest `json:"kontrak" validate:"required"`
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
		request.Role == nil
}
