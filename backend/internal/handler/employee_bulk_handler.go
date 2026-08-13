package handler

import (
	"context"
	"encoding/csv"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
)

type employeeBulkService interface {
	BulkCreate(context.Context, domain.Identity, []dto.CreateEmployeeRequest, service.RequestMeta) (service.BulkEmployeeResult, error)
}

var bulkHeaders = []string{"nip", "nama", "email", "jenis_kelamin", "tanggal_lahir", "tanggal_join", "department_id", "position_id", "atasan_id", "status_pernikahan", "role", "jalan", "kota", "provinsi", "nomor_ktp", "nomor_kontrak", "jenis_kontrak", "tanggal_mulai_kontrak", "tanggal_berakhir_kontrak"}

func (h *EmployeeHandler) BulkCreate(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		return
	}
	if identity.Role != domain.RoleHR {
		response.Error(w, 403, "FORBIDDEN", "Anda tidak memiliki akses")
		return
	}
	bulk, ok := h.service.(employeeBulkService)
	if !ok {
		response.Error(w, 501, "NOT_IMPLEMENTED", "Bulk upload belum tersedia")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		response.Error(w, 400, "INVALID_BODY", "Berkas CSV tidak valid")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		response.ValidationError(w, map[string]string{"file": "Berkas CSV wajib diunggah"})
		return
	}
	defer file.Close()
	reader := csv.NewReader(io.LimitReader(file, 5<<20))
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 || len(records) > 1001 {
		response.Error(w, 400, "INVALID_PARAM", "CSV harus berisi header dan 1–1000 karyawan")
		return
	}
	for i, name := range bulkHeaders {
		if i >= len(records[0]) || strings.TrimSpace(records[0][i]) != name {
			response.Error(w, 400, "INVALID_PARAM", "Header CSV tidak sesuai template")
			return
		}
	}
	items := make([]dto.CreateEmployeeRequest, 0, len(records)-1)
	for _, row := range records[1:] {
		if len(row) < len(bulkHeaders) {
			response.Error(w, 400, "INVALID_PARAM", "Kolom CSV tidak lengkap")
			return
		}
		department, err := uuid.Parse(row[6])
		if err != nil {
			response.Error(w, 400, "INVALID_PARAM", "department_id tidak valid")
			return
		}
		position, err := uuid.Parse(row[7])
		if err != nil {
			response.Error(w, 400, "INVALID_PARAM", "position_id tidak valid")
			return
		}
		var supervisor *uuid.UUID
		if strings.TrimSpace(row[8]) != "" {
			id, e := uuid.Parse(row[8])
			if e != nil {
				response.Error(w, 400, "INVALID_PARAM", "atasan_id tidak valid")
				return
			}
			supervisor = &id
		}
		marital := strings.TrimSpace(row[9])
		items = append(items, dto.CreateEmployeeRequest{NIP: row[0], Name: row[1], Email: row[2], Gender: row[3], BirthDate: row[4], JoinDate: row[5], DepartmentID: department, PositionID: position, SupervisorID: supervisor, MaritalStatus: &marital, Role: domain.RoleName(row[10]), Address: dto.EmployeeAddressRequest{Street: row[11], City: row[12], Province: row[13]}, KTP: dto.EmployeeKTPRequest{Number: row[14]}, Contract: dto.EmployeeContractRequest{Number: row[15], Type: row[16], StartDate: row[17], EndDate: row[18]}})
	}
	result, err := bulk.BulkCreate(r.Context(), identity, items, h.requestMeta(r))
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.Success(w, 201, result, "Bulk upload selesai")
}
