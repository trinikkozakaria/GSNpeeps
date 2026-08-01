package handler

import (
	"context"
	"net/http"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
)

type ProfileService interface {
	Me(context.Context, domain.Identity) (domain.EmployeeDetail, error)
	Metrics(context.Context, domain.Identity) (domain.PersonalMetrics, error)
}

type ProfileHandler struct {
	service ProfileService
}

func NewProfileHandler(service ProfileService) *ProfileHandler {
	return &ProfileHandler{service: service}
}

// Me mengembalikan profil milik user yang login. Identitas selalu berasal dari token, tidak
// pernah dari parameter request.
func (h *ProfileHandler) Me(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	profile, err := h.service.Me(request.Context(), identity)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, profile, "OK")
}

// Metrics mengembalikan metrik personal bulan berjalan; Top Management menerima 403.
func (h *ProfileHandler) Metrics(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	metrics, err := h.service.Metrics(request.Context(), identity)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, metrics, "OK")
}
