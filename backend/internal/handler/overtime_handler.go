package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
)

type OvertimeService interface {
	Create(
		context.Context, domain.Identity, domain.CreateOvertimeRequest, service.RequestMeta,
	) (domain.RequestStateData, error)
	List(
		context.Context, domain.Identity, domain.OvertimeRequestFilter,
	) (domain.OvertimeRequestPage, error)
	ListMine(
		context.Context, domain.Identity, int, int,
	) (domain.OvertimeRequestPage, error)
	Detail(context.Context, domain.Identity, uuid.UUID) (domain.OvertimeRequestDetail, error)
	Decide(
		context.Context, domain.Identity, uuid.UUID, domain.DecisionInput, service.RequestMeta,
	) (domain.RequestStateData, error)
	Recap(
		context.Context, domain.Identity, domain.OvertimeRecapFilter,
	) ([]domain.OvertimeRecapItem, error)
}

type OvertimeHandler struct {
	service    OvertimeService
	validator  Validator
	trustProxy bool
}

func NewOvertimeHandler(
	service OvertimeService,
	validator Validator,
	trustProxy bool,
) *OvertimeHandler {
	return &OvertimeHandler{service: service, validator: validator, trustProxy: trustProxy}
}

func (h *OvertimeHandler) requestMeta(request *http.Request) service.RequestMeta {
	return service.RequestMeta{
		IPAddress: clientIP(request, h.trustProxy),
		RequestID: middleware.RequestIDFromContext(request.Context()),
	}
}

// Create menerima multipart pengajuan lembur; dokumen pendukung bersifat opsional.
func (h *OvertimeHandler) Create(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}

	request.Body = http.MaxBytesReader(writer, request.Body, maxDocumentBytes+multipartMemoryBytes)
	if err := request.ParseMultipartForm(multipartMemoryBytes); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			response.FromError(writer, domain.ErrFileTooLarge)
			return
		}
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	defer func() {
		if request.MultipartForm != nil {
			_ = request.MultipartForm.RemoveAll()
		}
	}()

	fields := map[string]string{}
	date := strings.TrimSpace(request.FormValue("tanggal"))
	if date == "" {
		fields["tanggal"] = "Tanggal lembur wajib diisi"
	}
	startTime := strings.TrimSpace(request.FormValue("waktu_mulai"))
	endTime := strings.TrimSpace(request.FormValue("waktu_selesai"))
	if startTime == "" {
		fields["waktu_mulai"] = "Waktu mulai wajib diisi"
	}
	if endTime == "" {
		fields["waktu_selesai"] = "Waktu selesai wajib diisi"
	}
	reason := strings.TrimSpace(request.FormValue("alasan"))
	if len(reason) < 10 || len(reason) > 2000 {
		fields["alasan"] = "Alasan wajib diisi antara 10 dan 2000 karakter"
	}
	if len(fields) > 0 {
		response.ValidationError(writer, fields)
		return
	}

	document, ok := readSupportingDocument(writer, request, "dokumen_pendukung")
	if !ok {
		return
	}

	result, err := h.service.Create(request.Context(), identity, domain.CreateOvertimeRequest{
		UserID:    identity.UserID,
		Date:      date,
		StartTime: startTime,
		EndTime:   endTime,
		Reason:    reason,
		Document:  document,
	}, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusCreated, result, "Pengajuan lembur berhasil dikirim")
}

func (h *OvertimeHandler) List(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
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
	status, ok := optionalStatusQuery(writer, request)
	if !ok {
		return
	}
	filter := domain.OvertimeRequestFilter{Status: status, Page: page, Limit: limit}
	if value := strings.TrimSpace(request.URL.Query().Get("tanggal_mulai")); value != "" {
		filter.Start = &value
	}
	if value := strings.TrimSpace(request.URL.Query().Get("tanggal_selesai")); value != "" {
		filter.End = &value
	}

	result, err := h.service.List(request.Context(), identity, filter)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Paginated(writer, result.Items, response.PaginationMeta{
		Page:      result.Page,
		Limit:     result.Limit,
		TotalData: result.Total,
		TotalPage: totalPages(result.Total, result.Limit),
	}, "OK")
}

func (h *OvertimeHandler) ListMine(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
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
	result, err := h.service.ListMine(request.Context(), identity, page, limit)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Paginated(writer, result.Items, response.PaginationMeta{
		Page:      result.Page,
		Limit:     result.Limit,
		TotalData: result.Total,
		TotalPage: totalPages(result.Total, result.Limit),
	}, "OK")
}

func (h *OvertimeHandler) Detail(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	id, err := uuid.Parse(mux.Vars(request)["id"])
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "ID pengajuan tidak valid")
		return
	}
	detail, err := h.service.Detail(request.Context(), identity, id)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, detail, "OK")
}

func (h *OvertimeHandler) Decide(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	id, input, ok := decodeDecision(writer, request, h.validator)
	if !ok {
		return
	}
	result, err := h.service.Decide(request.Context(), identity, id, input, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, result, "Keputusan tersimpan")
}

func (h *OvertimeHandler) Recap(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	departmentID, ok := optionalUUIDQuery(writer, request, "department_id")
	if !ok {
		return
	}
	filter := domain.OvertimeRecapFilter{DepartmentID: departmentID}
	if value := strings.TrimSpace(request.URL.Query().Get("tanggal_mulai")); value != "" {
		filter.Start = &value
	}
	if value := strings.TrimSpace(request.URL.Query().Get("tanggal_selesai")); value != "" {
		filter.End = &value
	}

	items, err := h.service.Recap(request.Context(), identity, filter)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, items, "OK")
}
