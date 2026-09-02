package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newResetPasswordRouter(stub *employeeServiceStub) http.Handler {
	handler := NewEmployeeHandler(stub, validation.New(), false)
	router := mux.NewRouter()
	router.HandleFunc("/karyawan/{id}/reset-password", handler.ResetEmployeePassword).Methods(http.MethodPost)
	return router
}

func postReset(t *testing.T, stub *employeeServiceStub, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/karyawan/"+id+"/reset-password", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(middleware.WithIdentity(request.Context(), domain.Identity{
		UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR,
	}))
	recorder := httptest.NewRecorder()
	newResetPasswordRouter(stub).ServeHTTP(recorder, request)
	return recorder
}

func TestResetEmployeePasswordHandlerAcceptsValidBody(t *testing.T) {
	stub := &employeeServiceStub{}
	id := uuid.NewString()

	recorder := postReset(t, stub, id, `{"new_password":"GantiSandi2026!","new_password_confirmation":"GantiSandi2026!"}`)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, stub.resetCalled)
	assert.Equal(t, id, stub.resetID.String())

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			PasswordReset   bool `json:"password_reset"`
			AccountUnlocked bool `json:"account_unlocked"`
			SessionsRevoked bool `json:"sessions_revoked"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	assert.True(t, envelope.Success)
	assert.True(t, envelope.Data.PasswordReset)
	assert.True(t, envelope.Data.AccountUnlocked)
	assert.True(t, envelope.Data.SessionsRevoked)
	assert.NotContains(t, recorder.Body.String(), "GantiSandi2026!", "password tidak boleh dikembalikan")
}

func TestResetEmployeePasswordHandlerRejectsShortPasswordWith422(t *testing.T) {
	stub := &employeeServiceStub{}

	recorder := postReset(t, stub, uuid.NewString(), `{"new_password":"pendek","new_password_confirmation":"pendek"}`)

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.False(t, stub.resetCalled, "service tidak dipanggil saat body invalid")

	var envelope struct {
		Success bool `json:"success"`
		Error   struct {
			Code   string            `json:"code"`
			Fields map[string]string `json:"fields"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	assert.False(t, envelope.Success)
	assert.Equal(t, "VALIDATION_ERROR", envelope.Error.Code)
	assert.Contains(t, envelope.Error.Fields, "new_password")
}

func TestResetEmployeePasswordHandlerRejectsMissingConfirmationWith422(t *testing.T) {
	stub := &employeeServiceStub{}

	recorder := postReset(t, stub, uuid.NewString(), `{"new_password":"GantiSandi2026!"}`)

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.False(t, stub.resetCalled)
}

func TestResetEmployeePasswordHandlerRejectsMalformedBodyWith400(t *testing.T) {
	stub := &employeeServiceStub{}

	recorder := postReset(t, stub, uuid.NewString(), `{"new_password":`)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.False(t, stub.resetCalled)
}

func TestResetEmployeePasswordHandlerRejectsInvalidUUIDWith400(t *testing.T) {
	stub := &employeeServiceStub{}

	recorder := postReset(t, stub, "not-a-uuid", `{"new_password":"GantiSandi2026!","new_password_confirmation":"GantiSandi2026!"}`)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.False(t, stub.resetCalled)
}

func TestResetEmployeePasswordHandlerPropagatesForbidden(t *testing.T) {
	stub := &employeeServiceStub{resetErr: domain.ErrForbidden}

	recorder := postReset(t, stub, uuid.NewString(), `{"new_password":"GantiSandi2026!","new_password_confirmation":"GantiSandi2026!"}`)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	assert.True(t, stub.resetCalled)
}

func TestResetEmployeePasswordHandlerPropagatesNotFound(t *testing.T) {
	stub := &employeeServiceStub{resetErr: domain.ErrNotFound}

	recorder := postReset(t, stub, uuid.NewString(), `{"new_password":"GantiSandi2026!","new_password_confirmation":"GantiSandi2026!"}`)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}
