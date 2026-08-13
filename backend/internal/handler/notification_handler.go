package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
)

type NotificationService interface {
	List(context.Context, domain.Identity, *bool, int, int) (domain.NotificationPage, error)
	UnreadCount(context.Context, domain.Identity) (int, error)
	MarkRead(context.Context, domain.Identity, uuid.UUID) error
	Dismiss(context.Context, domain.Identity, uuid.UUID) error
}

type NotificationHandler struct {
	service NotificationService
}

func NewNotificationHandler(service NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

func (h *NotificationHandler) List(writer http.ResponseWriter, request *http.Request) {
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
	isRead, ok := optionalBoolQuery(writer, request, "is_read")
	if !ok {
		return
	}

	result, err := h.service.List(request.Context(), identity, isRead, page, limit)
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

func (h *NotificationHandler) UnreadCount(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	total, err := h.service.UnreadCount(request.Context(), identity)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, struct {
		UnreadCount int `json:"unread_count"`
	}{UnreadCount: total}, "OK")
}

func (h *NotificationHandler) MarkRead(writer http.ResponseWriter, request *http.Request) {
	identity, id, ok := h.identityAndID(writer, request)
	if !ok {
		return
	}
	if err := h.service.MarkRead(request.Context(), identity, id); err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, struct {
		ID uuid.UUID `json:"id"`
	}{ID: id}, "Notifikasi ditandai telah dibaca")
}

func (h *NotificationHandler) Dismiss(writer http.ResponseWriter, request *http.Request) {
	identity, id, ok := h.identityAndID(writer, request)
	if !ok {
		return
	}
	if err := h.service.Dismiss(request.Context(), identity, id); err != nil {
		response.FromError(writer, err)
		return
	}
	response.EmptySuccess(writer, "Notifikasi berhasil dihapus")
}

func (h *NotificationHandler) identityAndID(
	writer http.ResponseWriter,
	request *http.Request,
) (domain.Identity, uuid.UUID, bool) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return domain.Identity{}, uuid.Nil, false
	}
	id, err := uuid.Parse(mux.Vars(request)["id"])
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "ID notifikasi tidak valid")
		return domain.Identity{}, uuid.Nil, false
	}
	return identity, id, true
}

// optionalBoolQuery membaca parameter boolean opsional. Nilai selain true/false ditolak agar
// filter tidak diam-diam berubah menjadi "tanpa filter".
func optionalBoolQuery(
	writer http.ResponseWriter,
	request *http.Request,
	name string,
) (*bool, bool) {
	raw := strings.TrimSpace(request.URL.Query().Get(name))
	if raw == "" {
		return nil, true
	}
	if raw != "true" && raw != "false" {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM",
			"Parameter "+name+" harus true atau false")
		return nil, false
	}
	value := raw == "true"
	return &value, true
}
