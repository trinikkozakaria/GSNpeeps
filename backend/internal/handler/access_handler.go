package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
)

type AccessService interface {
	ListRoles(context.Context, domain.Identity) ([]domain.RoleSummary, error)
	PermissionMatrix(context.Context, domain.Identity) ([]domain.Permission, error)
	UpdatePermission(
		context.Context, domain.Identity, domain.PermissionChange, service.RequestMeta,
	) (uuid.UUID, error)
	ListAuditLogs(
		context.Context, domain.Identity, domain.AuditLogFilter,
	) (domain.AuditLogPage, error)
}

type AccessHandler struct {
	service    AccessService
	validator  Validator
	trustProxy bool
}

func NewAccessHandler(service AccessService, validator Validator, trustProxy bool) *AccessHandler {
	return &AccessHandler{service: service, validator: validator, trustProxy: trustProxy}
}

func (h *AccessHandler) ListRoles(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	roles, err := h.service.ListRoles(request.Context(), identity)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, roles, "OK")
}

func (h *AccessHandler) PermissionMatrix(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	items, err := h.service.PermissionMatrix(request.Context(), identity)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, items, "OK")
}

func (h *AccessHandler) UpdatePermission(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	var input dto.UpdatePermissionRequest
	if decodeJSON(request, &input) != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	if fields := h.validator.Struct(input); len(fields) > 0 {
		response.ValidationError(writer, fields)
		return
	}

	id, err := h.service.UpdatePermission(request.Context(), identity, domain.PermissionChange{
		RoleID:    input.RoleID,
		Module:    strings.TrimSpace(input.Module),
		Action:    input.Action,
		IsAllowed: *input.IsAllowed,
	}, service.RequestMeta{
		IPAddress: clientIP(request, h.trustProxy),
		RequestID: middleware.RequestIDFromContext(request.Context()),
	})
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, struct {
		ID uuid.UUID `json:"id"`
	}{ID: id}, "Permission berhasil diperbarui")
}

func (h *AccessHandler) ListAuditLogs(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	filter, ok := auditLogFilter(writer, request)
	if !ok {
		return
	}

	page, err := h.service.ListAuditLogs(request.Context(), identity, filter)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Paginated(writer, page.Items, response.PaginationMeta{
		Page:      page.Page,
		Limit:     page.Limit,
		TotalData: page.Total,
		TotalPage: totalPages(page.Total, page.Limit),
	}, "OK")
}

// auditLogFilter membaca filter kontrak. Rentang tanggal memakai batas atas eksklusif hari
// berikutnya sehingga `tanggal_selesai` ikut terhitung penuh.
func auditLogFilter(
	writer http.ResponseWriter,
	request *http.Request,
) (domain.AuditLogFilter, bool) {
	page, ok := positiveIntQuery(writer, request, "page", 1)
	if !ok {
		return domain.AuditLogFilter{}, false
	}
	limit, ok := positiveIntQuery(writer, request, "limit", 10)
	if !ok {
		return domain.AuditLogFilter{}, false
	}
	filter := domain.AuditLogFilter{Page: page, Limit: limit}

	if raw := strings.TrimSpace(request.URL.Query().Get("user_id")); raw != "" {
		userID, err := uuid.Parse(raw)
		if err != nil {
			response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "user_id tidak valid")
			return domain.AuditLogFilter{}, false
		}
		filter.UserID = &userID
	}
	if raw := strings.TrimSpace(request.URL.Query().Get("aksi")); raw != "" {
		filter.Action = &raw
	}
	if raw := strings.TrimSpace(request.URL.Query().Get("modul")); raw != "" {
		filter.Module = &raw
	}

	start, ok := optionalDateQuery(writer, request, "tanggal_mulai")
	if !ok {
		return domain.AuditLogFilter{}, false
	}
	filter.StartDate = start

	end, ok := optionalDateQuery(writer, request, "tanggal_selesai")
	if !ok {
		return domain.AuditLogFilter{}, false
	}
	if end != nil {
		exclusive := end.AddDate(0, 0, 1)
		filter.EndDate = &exclusive
	}
	return filter, true
}

func optionalDateQuery(
	writer http.ResponseWriter,
	request *http.Request,
	name string,
) (*time.Time, bool) {
	raw := strings.TrimSpace(request.URL.Query().Get(name))
	if raw == "" {
		return nil, true
	}
	// Batas dibaca pada zona Asia/Jakarta lalu dibandingkan dalam UTC seperti kolom
	// created_at yang disimpan modul lain.
	parsed, err := time.ParseInLocation(domain.DateLayout, raw, domain.Jakarta())
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM",
			"Parameter "+name+" harus berformat YYYY-MM-DD")
		return nil, false
	}
	value := parsed.UTC()
	return &value, true
}
