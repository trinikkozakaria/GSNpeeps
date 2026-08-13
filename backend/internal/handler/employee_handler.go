package handler

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
)

type EmployeeService interface {
	ListDepartments(context.Context) ([]domain.Department, error)
	ListPositions(context.Context, *uuid.UUID) ([]domain.Position, error)
	List(context.Context, domain.Identity, domain.EmployeeFilter) (domain.EmployeePage, error)
	Detail(context.Context, domain.Identity, uuid.UUID) (domain.EmployeeDetail, error)
	Create(
		context.Context,
		domain.Identity,
		dto.CreateEmployeeRequest,
		service.RequestMeta,
	) (domain.EmployeeMutationResult, error)
	Update(
		context.Context,
		domain.Identity,
		uuid.UUID,
		dto.UpdateEmployeeRequest,
		service.RequestMeta,
	) (domain.EmployeeMutationResult, error)
	Deactivate(
		context.Context,
		domain.Identity,
		uuid.UUID,
		service.RequestMeta,
	) (domain.EmployeeMutationResult, error)
	ListDocuments(context.Context, domain.Identity, uuid.UUID) ([]domain.EmployeeDocument, error)
	UploadDocument(
		context.Context,
		domain.Identity,
		uuid.UUID,
		service.DocumentUpload,
		service.RequestMeta,
	) (domain.EmployeeDocument, error)
	Export(
		context.Context,
		domain.Identity,
		domain.EmployeeExportQuery,
		service.RequestMeta,
	) (domain.ExportFile, error)
	UpdatePhoto(
		context.Context,
		domain.Identity,
		uuid.UUID,
		service.PhotoUpload,
		service.RequestMeta,
	) (string, error)
}

func (h *EmployeeHandler) Create(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	var input dto.CreateEmployeeRequest
	if decodeJSON(request, &input) != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	if fields := h.validator.Struct(input); len(fields) > 0 {
		response.RequestValidationError(writer, fields)
		return
	}
	result, err := h.service.Create(request.Context(), identity, input, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusCreated, struct {
		ID uuid.UUID `json:"id"`
	}{ID: result.EmployeeID}, "Karyawan berhasil ditambahkan")
}

type EmployeeHandler struct {
	service    EmployeeService
	validator  Validator
	trustProxy bool
}

func NewEmployeeHandler(
	service EmployeeService,
	validator Validator,
	trustProxy bool,
) *EmployeeHandler {
	return &EmployeeHandler{service: service, validator: validator, trustProxy: trustProxy}
}

func (h *EmployeeHandler) Update(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	id, err := uuid.Parse(mux.Vars(request)["id"])
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "ID karyawan tidak valid")
		return
	}
	var input dto.UpdateEmployeeRequest
	if decodeJSON(request, &input) != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	if fields := h.validator.Struct(input); len(fields) > 0 {
		response.ValidationError(writer, fields)
		return
	}
	result, err := h.service.Update(request.Context(), identity, id, input, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, struct {
		ID uuid.UUID `json:"id"`
	}{ID: result.EmployeeID}, "Karyawan berhasil diperbarui")
}

func (h *EmployeeHandler) Deactivate(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	id, err := uuid.Parse(mux.Vars(request)["id"])
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "ID karyawan tidak valid")
		return
	}
	result, err := h.service.Deactivate(request.Context(), identity, id, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, struct {
		ID     uuid.UUID `json:"id"`
		Status string    `json:"status"`
	}{ID: result.EmployeeID, Status: "nonaktif"}, "Karyawan berhasil dinonaktifkan")
}

func (h *EmployeeHandler) requestMeta(request *http.Request) service.RequestMeta {
	return service.RequestMeta{
		IPAddress: clientIP(request, h.trustProxy),
		RequestID: middleware.RequestIDFromContext(request.Context()),
	}
}

func (h *EmployeeHandler) ListDepartments(writer http.ResponseWriter, request *http.Request) {
	items, err := h.service.ListDepartments(request.Context())
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, items, "OK")
}

func (h *EmployeeHandler) ListPositions(writer http.ResponseWriter, request *http.Request) {
	departmentID, ok := optionalUUIDQuery(writer, request, "department_id")
	if !ok {
		return
	}
	items, err := h.service.ListPositions(request.Context(), departmentID)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, items, "OK")
}

func (h *EmployeeHandler) List(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	departmentID, ok := optionalUUIDQuery(writer, request, "department_id")
	if !ok {
		return
	}
	page, ok := positiveIntQuery(writer, request, "page", 1)
	if !ok {
		return
	}
	limit, ok := positiveIntQuery(writer, request, "limit", 10)
	if !ok {
		return
	}
	result, err := h.service.List(request.Context(), identity, domain.EmployeeFilter{
		Search:       request.URL.Query().Get("search"),
		DepartmentID: departmentID,
		Status:       strings.TrimSpace(request.URL.Query().Get("status")),
		Page:         page,
		Limit:        limit,
	})
	if err != nil {
		response.FromError(writer, err)
		return
	}
	totalPage := 0
	if result.Total > 0 {
		totalPage = int(math.Ceil(float64(result.Total) / float64(result.Limit)))
	}
	response.Paginated(writer, result.Items, response.PaginationMeta{
		Page:      result.Page,
		Limit:     result.Limit,
		TotalData: result.Total,
		TotalPage: totalPage,
	}, "OK")
}

func (h *EmployeeHandler) Detail(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	id, err := uuid.Parse(mux.Vars(request)["id"])
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "ID karyawan tidak valid")
		return
	}
	item, err := h.service.Detail(request.Context(), identity, id)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, item, "OK")
}

func optionalUUIDQuery(
	writer http.ResponseWriter,
	request *http.Request,
	key string,
) (*uuid.UUID, bool) {
	raw := strings.TrimSpace(request.URL.Query().Get(key))
	if raw == "" {
		return nil, true
	}
	value, err := uuid.Parse(raw)
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "Parameter "+key+" tidak valid")
		return nil, false
	}
	return &value, true
}

func positiveIntQuery(
	writer http.ResponseWriter,
	request *http.Request,
	key string,
	fallback int,
) (int, bool) {
	raw := strings.TrimSpace(request.URL.Query().Get(key))
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "Parameter "+key+" tidak valid")
		return 0, false
	}
	return value, true
}
