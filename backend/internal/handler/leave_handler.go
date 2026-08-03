package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/filetype"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
)

type LeaveService interface {
	ListLeaveTypes(context.Context, domain.Identity, *bool) ([]domain.LeaveType, error)
	CreateLeaveType(
		context.Context, domain.Identity, domain.CreateLeaveType, service.RequestMeta,
	) (uuid.UUID, error)
	UpdateLeaveType(
		context.Context, domain.Identity, uuid.UUID, domain.UpdateLeaveType, service.RequestMeta,
	) error
	Create(
		context.Context, domain.Identity, domain.CreateLeaveRequest, service.RequestMeta,
	) (domain.RequestStateData, error)
	ListForApproval(
		context.Context, domain.Identity, *domain.RequestStatus, int, int,
	) (domain.LeaveRequestPage, error)
	ListMine(context.Context, domain.Identity, int, int) (domain.LeaveRequestPage, error)
	Detail(context.Context, domain.Identity, uuid.UUID) (domain.LeaveRequestDetail, error)
	Decide(
		context.Context, domain.Identity, uuid.UUID, domain.DecisionInput, service.RequestMeta,
	) (domain.RequestStateData, error)
	Delegate(
		context.Context, domain.Identity, uuid.UUID, string, service.RequestMeta,
	) (domain.RequestStateData, error)
}

type LeaveHandler struct {
	service    LeaveService
	validator  Validator
	trustProxy bool
}

func NewLeaveHandler(service LeaveService, validator Validator, trustProxy bool) *LeaveHandler {
	return &LeaveHandler{service: service, validator: validator, trustProxy: trustProxy}
}

func (h *LeaveHandler) requestMeta(request *http.Request) service.RequestMeta {
	return service.RequestMeta{
		IPAddress: clientIP(request, h.trustProxy),
		RequestID: middleware.RequestIDFromContext(request.Context()),
	}
}

func (h *LeaveHandler) ListTypes(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	var activeOnly *bool
	if raw := strings.TrimSpace(request.URL.Query().Get("aktif")); raw != "" {
		value := raw == "true"
		if raw != "true" && raw != "false" {
			response.Error(writer, http.StatusBadRequest, "INVALID_PARAM",
				"Parameter aktif harus true atau false")
			return
		}
		activeOnly = &value
	}
	items, err := h.service.ListLeaveTypes(request.Context(), identity, activeOnly)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, items, "OK")
}

func (h *LeaveHandler) CreateType(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	var input dto.CreateLeaveTypeRequest
	if decodeJSON(request, &input) != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	if fields := h.validator.Struct(input); len(fields) > 0 {
		response.RequestValidationError(writer, fields)
		return
	}
	id, err := h.service.CreateLeaveType(request.Context(), identity, domain.CreateLeaveType{
		Code:             input.Code,
		Name:             input.Name,
		AnnualQuota:      input.AnnualQuota,
		RequiresDocument: input.RequiresDocument,
	}, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusCreated, struct {
		ID uuid.UUID `json:"id"`
	}{ID: id}, "Jenis izin berhasil dibuat")
}

func (h *LeaveHandler) UpdateType(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	id, err := uuid.Parse(mux.Vars(request)["id"])
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "ID jenis izin tidak valid")
		return
	}
	var input dto.UpdateLeaveTypeRequest
	if decodeJSON(request, &input) != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	if fields := h.validator.Struct(input); len(fields) > 0 {
		response.ValidationError(writer, fields)
		return
	}
	err = h.service.UpdateLeaveType(request.Context(), identity, id, domain.UpdateLeaveType{
		Name:             input.Name,
		AnnualQuota:      input.AnnualQuota,
		RequiresDocument: input.RequiresDocument,
		IsActive:         input.IsActive,
	}, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, struct {
		ID uuid.UUID `json:"id"`
	}{ID: id}, "Jenis izin berhasil diperbarui")
}

// Create menerima multipart pengajuan ketidakhadiran.
func (h *LeaveHandler) Create(writer http.ResponseWriter, request *http.Request) {
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
	leaveTypeID, err := uuid.Parse(strings.TrimSpace(request.FormValue("jenis_izin_id")))
	if err != nil {
		fields["jenis_izin_id"] = "Jenis izin wajib dipilih"
	}
	startDate := strings.TrimSpace(request.FormValue("tanggal_mulai"))
	endDate := strings.TrimSpace(request.FormValue("tanggal_selesai"))
	if startDate == "" {
		fields["tanggal_mulai"] = "Tanggal mulai wajib diisi"
	}
	if endDate == "" {
		fields["tanggal_selesai"] = "Tanggal selesai wajib diisi"
	}
	reason := strings.TrimSpace(request.FormValue("alasan"))
	if len(reason) < 10 || len(reason) > 2000 {
		fields["alasan"] = "Alasan wajib diisi antara 10 dan 2000 karakter"
	}
	if len(fields) > 0 {
		response.ValidationError(writer, fields)
		return
	}

	command := domain.CreateLeaveRequest{
		UserID:      identity.UserID,
		LeaveTypeID: leaveTypeID,
		StartDate:   startDate,
		EndDate:     endDate,
		Reason:      reason,
	}
	if value := strings.TrimSpace(request.FormValue("lokasi_tujuan")); value != "" {
		command.Destination = &value
	}
	if value := strings.TrimSpace(request.FormValue("keterangan_lokasi")); value != "" {
		command.DestinationNote = &value
	}

	document, ok := readSupportingDocument(writer, request, "dokumen_pendukung")
	if !ok {
		return
	}
	command.Document = document

	result, err := h.service.Create(request.Context(), identity, command, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusCreated, result, "Pengajuan berhasil dikirim")
}

// readSupportingDocument membaca dokumen pendukung opsional dari multipart. Mengembalikan
// nil bila field tidak dikirim; kewajiban dokumen ditentukan service.
func readSupportingDocument(
	writer http.ResponseWriter,
	request *http.Request,
	field string,
) (*domain.UploadedFile, bool) {
	file, header, err := request.FormFile(field)
	if err != nil {
		return nil, true
	}
	defer file.Close()

	if header.Size > maxDocumentBytes {
		response.FromError(writer, domain.ErrFileTooLarge)
		return nil, false
	}
	content, err := io.ReadAll(io.LimitReader(file, maxDocumentBytes+1))
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Berkas tidak dapat dibaca")
		return nil, false
	}
	if len(content) > maxDocumentBytes {
		response.FromError(writer, domain.ErrFileTooLarge)
		return nil, false
	}
	if len(content) == 0 {
		response.ValidationError(writer, map[string]string{field: "Berkas dokumen kosong"})
		return nil, false
	}

	head := content
	if len(head) > signatureHeadBytes {
		head = head[:signatureHeadBytes]
	}
	descriptor, err := filetype.ValidateSupportingDocument(
		header.Filename, header.Header.Get("Content-Type"), head,
	)
	if err != nil {
		response.FromError(writer, domain.ErrUnsupportedFile)
		return nil, false
	}
	return &domain.UploadedFile{
		FileName:  header.Filename,
		Extension: descriptor.Extension,
		MediaType: descriptor.MediaType,
		Content:   content,
	}, true
}

func (h *LeaveHandler) ListForApproval(writer http.ResponseWriter, request *http.Request) {
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
	result, err := h.service.ListForApproval(request.Context(), identity, status, page, limit)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	writeLeavePage(writer, result)
}

func (h *LeaveHandler) ListMine(writer http.ResponseWriter, request *http.Request) {
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
	writeLeavePage(writer, result)
}

func writeLeavePage(writer http.ResponseWriter, result domain.LeaveRequestPage) {
	response.Paginated(writer, result.Items, response.PaginationMeta{
		Page:      result.Page,
		Limit:     result.Limit,
		TotalData: result.Total,
		TotalPage: totalPages(result.Total, result.Limit),
	}, "OK")
}

func optionalStatusQuery(
	writer http.ResponseWriter,
	request *http.Request,
) (*domain.RequestStatus, bool) {
	raw := strings.TrimSpace(request.URL.Query().Get("status"))
	if raw == "" {
		return nil, true
	}
	status := domain.RequestStatus(raw)
	if !status.Valid() {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "Status tidak valid")
		return nil, false
	}
	return &status, true
}

func (h *LeaveHandler) Detail(writer http.ResponseWriter, request *http.Request) {
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

func (h *LeaveHandler) Decide(writer http.ResponseWriter, request *http.Request) {
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

func (h *LeaveHandler) Delegate(writer http.ResponseWriter, request *http.Request) {
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
	var input dto.DelegateRequest
	if decodeJSON(request, &input) != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	if fields := h.validator.Struct(input); len(fields) > 0 {
		response.ValidationError(writer, fields)
		return
	}
	result, err := h.service.Delegate(
		request.Context(), identity, id, input.Note, h.requestMeta(request),
	)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, result, "Pengajuan didelegasikan ke HR")
}

// decodeDecision membaca body keputusan bersama untuk ketidakhadiran dan lembur.
func decodeDecision(
	writer http.ResponseWriter,
	request *http.Request,
	validator Validator,
) (uuid.UUID, domain.DecisionInput, bool) {
	id, err := uuid.Parse(mux.Vars(request)["id"])
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "ID pengajuan tidak valid")
		return uuid.Nil, domain.DecisionInput{}, false
	}
	var input dto.DecisionRequest
	if decodeJSON(request, &input) != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return uuid.Nil, domain.DecisionInput{}, false
	}
	if fields := validator.Struct(input); len(fields) > 0 {
		response.ValidationError(writer, fields)
		return uuid.Nil, domain.DecisionInput{}, false
	}
	// Penolakan wajib disertai catatan minimal lima karakter sesuai schema DecisionRequest.
	if input.Decision == "tolak" && len(strings.TrimSpace(input.Note)) < 5 {
		response.ValidationError(writer, map[string]string{
			"catatan": "Catatan wajib diisi minimal 5 karakter saat menolak",
		})
		return uuid.Nil, domain.DecisionInput{}, false
	}
	decision := domain.DecisionInput{Approve: input.Decision == "setujui"}
	if note := strings.TrimSpace(input.Note); note != "" {
		decision.Note = &note
	}
	return id, decision, true
}
