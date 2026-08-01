package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
)

type DashboardService interface {
	Metrics(
		context.Context,
		domain.Identity,
		domain.DashboardPeriodType,
		string,
	) (domain.DashboardMetrics, error)
}

type DashboardHandler struct {
	service DashboardService
}

func NewDashboardHandler(service DashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

// Metrics mengembalikan agregasi dashboard untuk periode yang diminta. Default periode
// adalah bulanan dengan tanggal acuan hari ini di Asia/Jakarta.
func (h *DashboardHandler) Metrics(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	period := domain.DashboardPeriodType(
		strings.ToLower(strings.TrimSpace(request.URL.Query().Get("periode"))),
	)
	anchor := strings.TrimSpace(request.URL.Query().Get("tanggal_acuan"))

	metrics, err := h.service.Metrics(request.Context(), identity, period, anchor)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, metrics, "OK")
}
