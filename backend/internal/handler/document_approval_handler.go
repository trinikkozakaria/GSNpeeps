package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
)

type DocumentApprovalService interface {
	ListTemplates(context.Context, domain.Identity) ([]domain.WorkflowTemplate, error)
	CreateTemplate(
		context.Context, domain.Identity, domain.CreateWorkflowTemplate, service.RequestMeta,
	) (uuid.UUID, error)
	UpdateTemplate(
		context.Context, domain.Identity, uuid.UUID, domain.UpdateWorkflowTemplate, service.RequestMeta,
	) error

	Create(
		context.Context, domain.Identity, domain.CreateApprovalRequest, service.RequestMeta,
	) (domain.ApprovalRequest, bool, error)
	ListForApproval(context.Context, domain.Identity, int, int) (domain.ApprovalRequestPage, error)
	ListMine(
		context.Context, domain.Identity, *domain.DocumentApprovalStatus, int, int,
	) (domain.ApprovalRequestPage, error)
	ListMonitoring(
		context.Context, domain.Identity, *domain.DocumentApprovalStatus, *uuid.UUID, int, int,
	) (domain.ApprovalRequestPage, error)
	Detail(context.Context, domain.Identity, uuid.UUID) (domain.ApprovalRequestDetail, error)
	Decide(
		context.Context, domain.Identity, uuid.UUID, domain.DecisionInput, service.RequestMeta,
	) (domain.ApprovalRequest, error)
	Cancel(
		context.Context, domain.Identity, uuid.UUID, service.RequestMeta,
	) (domain.ApprovalRequest, error)
}

type DocumentApprovalHandler struct {
	service    DocumentApprovalService
	validator  Validator
	trustProxy bool
}

func NewDocumentApprovalHandler(
	service DocumentApprovalService, validator Validator, trustProxy bool,
) *DocumentApprovalHandler {
	return &DocumentApprovalHandler{service: service, validator: validator, trustProxy: trustProxy}
}

func (h *DocumentApprovalHandler) requestMeta(request *http.Request) service.RequestMeta {
	return service.RequestMeta{
		IPAddress: clientIP(request, h.trustProxy),
		RequestID: middleware.RequestIDFromContext(request.Context()),
	}
}

// --- Templates -----------------------------------------------------------------------

func (h *DocumentApprovalHandler) ListTemplates(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	items, err := h.service.ListTemplates(request.Context(), identity)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, items, "OK")
}

func (h *DocumentApprovalHandler) CreateTemplate(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	var input dto.CreateWorkflowTemplateRequest
	if decodeJSON(request, &input) != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	if fields := h.validator.Struct(input); len(fields) > 0 {
		response.ValidationError(writer, fields)
		return
	}

	stages, ok := parseStageRequests(writer, input.Stages)
	if !ok {
		return
	}

	id, err := h.service.CreateTemplate(request.Context(), identity, domain.CreateWorkflowTemplate{
		Name: input.Name, Active: input.Active, Stages: stages,
	}, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusCreated, struct {
		ID uuid.UUID `json:"id"`
	}{ID: id}, "Alur persetujuan berhasil dibuat")
}

func (h *DocumentApprovalHandler) UpdateTemplate(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	id, err := uuid.Parse(mux.Vars(request)["id"])
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "ID alur tidak valid")
		return
	}
	var input dto.UpdateWorkflowTemplateRequest
	if decodeJSON(request, &input) != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	if fields := h.validator.Struct(input); len(fields) > 0 {
		response.ValidationError(writer, fields)
		return
	}

	changes := domain.UpdateWorkflowTemplate{Name: input.Name, Active: input.Active}
	if input.Stages != nil {
		stages, ok := parseStageRequests(writer, *input.Stages)
		if !ok {
			return
		}
		changes.Stages = &stages
	}

	if err := h.service.UpdateTemplate(request.Context(), identity, id, changes, h.requestMeta(request)); err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, struct {
		ID uuid.UUID `json:"id"`
	}{ID: id}, "Alur persetujuan berhasil diperbarui")
}

// parseStageRequests converts the DTO's string PositionIDs to uuid.UUID, reporting a
// field-level validation error (rather than a generic 400) if any is malformed.
func parseStageRequests(
	writer http.ResponseWriter,
	input []dto.CreateWorkflowStageRequest,
) ([]domain.CreateWorkflowStage, bool) {
	stages := make([]domain.CreateWorkflowStage, 0, len(input))
	for i, stage := range input {
		positionID, err := uuid.Parse(stage.PositionID)
		if err != nil {
			response.ValidationError(writer, map[string]string{
				"tahap": "Position ID pada salah satu tahap tidak valid",
			})
			return nil, false
		}
		stages = append(stages, domain.CreateWorkflowStage{
			Order: i, StageName: strings.TrimSpace(stage.StageName), PositionID: positionID,
		})
	}
	return stages, true
}

// --- Submission and reading ------------------------------------------------------------

func (h *DocumentApprovalHandler) Create(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	var input dto.CreateApprovalRequestRequest
	if decodeJSON(request, &input) != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	if fields := h.validator.Struct(input); len(fields) > 0 {
		response.ValidationError(writer, fields)
		return
	}
	templateID, err := uuid.Parse(input.TemplateID)
	if err != nil {
		response.ValidationError(writer, map[string]string{
			"template_id": "Template ID tidak valid",
		})
		return
	}

	result, idempotentHit, err := h.service.Create(request.Context(), identity, domain.CreateApprovalRequest{
		TemplateID: templateID, DocumentURL: input.DocumentURL, Title: input.Title,
	}, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	// Idempotent hit is a 200 with the already-running request, not a 201/409 — PRD §5.2
	// is explicit this is the expected outcome, not an error.
	if idempotentHit {
		response.Success(writer, http.StatusOK, result, "Dokumen ini sudah dalam proses persetujuan")
		return
	}
	response.Success(writer, http.StatusCreated, result, "Pengajuan berhasil dikirim")
}

func (h *DocumentApprovalHandler) ListForApproval(writer http.ResponseWriter, request *http.Request) {
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
	result, err := h.service.ListForApproval(request.Context(), identity, page, limit)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	writeApprovalPage(writer, result)
}

func (h *DocumentApprovalHandler) ListMine(writer http.ResponseWriter, request *http.Request) {
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
	status, ok := optionalDocApprovalStatusQuery(writer, request)
	if !ok {
		return
	}
	result, err := h.service.ListMine(request.Context(), identity, status, page, limit)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	writeApprovalPage(writer, result)
}

func (h *DocumentApprovalHandler) ListMonitoring(writer http.ResponseWriter, request *http.Request) {
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
	status, ok := optionalDocApprovalStatusQuery(writer, request)
	if !ok {
		return
	}
	var templateID *uuid.UUID
	if raw := strings.TrimSpace(request.URL.Query().Get("template_id")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "template_id tidak valid")
			return
		}
		templateID = &parsed
	}
	result, err := h.service.ListMonitoring(request.Context(), identity, status, templateID, page, limit)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	writeApprovalPage(writer, result)
}

func writeApprovalPage(writer http.ResponseWriter, result domain.ApprovalRequestPage) {
	response.Paginated(writer, result.Items, response.PaginationMeta{
		Page:      result.Page,
		Limit:     result.Limit,
		TotalData: result.Total,
		TotalPage: totalPages(result.Total, result.Limit),
	}, "OK")
}

func optionalDocApprovalStatusQuery(
	writer http.ResponseWriter,
	request *http.Request,
) (*domain.DocumentApprovalStatus, bool) {
	raw := strings.TrimSpace(request.URL.Query().Get("status"))
	if raw == "" {
		return nil, true
	}
	switch domain.DocumentApprovalStatus(raw) {
	case domain.DocApprovalRunning, domain.DocApprovalApproved, domain.DocApprovalRejected,
		domain.DocApprovalCancelled, domain.DocApprovalBlocked:
		status := domain.DocumentApprovalStatus(raw)
		return &status, true
	default:
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "Status tidak valid")
		return nil, false
	}
}

func (h *DocumentApprovalHandler) Detail(writer http.ResponseWriter, request *http.Request) {
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

// --- Decision and cancellation ---------------------------------------------------------

func (h *DocumentApprovalHandler) Decide(writer http.ResponseWriter, request *http.Request) {
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
	var input dto.DocumentDecisionRequest
	if decodeJSON(request, &input) != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	if fields := h.validator.Struct(input); len(fields) > 0 {
		response.ValidationError(writer, fields)
		return
	}
	// Reject requires a note — PRD §5.3. Duplicated here (service also enforces it) purely
	// for a friendlier field-level error instead of a generic one; the service's own check
	// remains the real guard.
	if input.Keputusan == "reject" && len(strings.TrimSpace(input.Catatan)) < 5 {
		response.ValidationError(writer, map[string]string{
			"catatan": "Catatan wajib diisi minimal 5 karakter saat menolak",
		})
		return
	}

	decision := domain.DecisionInput{Approve: input.Keputusan == "approve"}
	if note := strings.TrimSpace(input.Catatan); note != "" {
		decision.Note = &note
	}

	result, err := h.service.Decide(request.Context(), identity, id, decision, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, result, "Keputusan tersimpan")
}

func (h *DocumentApprovalHandler) Cancel(writer http.ResponseWriter, request *http.Request) {
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
	result, err := h.service.Cancel(request.Context(), identity, id, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, result, "Pengajuan dibatalkan")
}