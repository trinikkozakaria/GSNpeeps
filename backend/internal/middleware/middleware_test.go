package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDPreservesValidValue(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(requestIDHeader, "request-12345678")
	recorder := httptest.NewRecorder()
	RequestID(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(recorder, request)
	if got := recorder.Header().Get(requestIDHeader); got != "request-12345678" {
		t.Fatalf("request id = %q", got)
	}
}

func TestRecoveryReturnsInternalErrorAndRequestID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := RequestID(Recovery(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("test")
	})))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", recorder.Code)
	}
	if recorder.Header().Get(requestIDHeader) == "" || !strings.Contains(recorder.Body.String(), "INTERNAL_ERROR") {
		t.Fatalf("unexpected response: headers=%v body=%s", recorder.Header(), recorder.Body.String())
	}
}
