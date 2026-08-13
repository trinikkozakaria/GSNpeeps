package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
)

type fakeHealthChecker struct{ healthy bool }

func (f fakeHealthChecker) Check(context.Context) (service.HealthResult, bool) {
	return service.HealthResult{Status: "ok", DB: "ok", Redis: "ok"}, f.healthy
}

func TestHealthHandler(t *testing.T) {
	for _, test := range []struct {
		name    string
		healthy bool
		status  int
	}{
		{name: "healthy", healthy: true, status: http.StatusOK},
		{name: "dependency unavailable", healthy: false, status: http.StatusServiceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			NewHealthHandler(fakeHealthChecker{healthy: test.healthy}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d", recorder.Code, test.status)
			}
		})
	}
}
