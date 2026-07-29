package handler

import (
	"context"
	"net/http"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
)

type HealthChecker interface {
	Check(context.Context) (service.HealthResult, bool)
}

type HealthHandler struct {
	checker HealthChecker
}

func NewHealthHandler(checker HealthChecker) *HealthHandler {
	return &HealthHandler{checker: checker}
}

func (h *HealthHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	result, healthy := h.checker.Check(request.Context())
	if !healthy {
		response.Error(writer, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Layanan sementara tidak tersedia")
		return
	}
	response.Success(writer, http.StatusOK, result, "Service healthy")
}
