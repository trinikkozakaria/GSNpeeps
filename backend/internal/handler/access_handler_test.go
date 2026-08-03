package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/validation"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type accessServiceStub struct {
	change       domain.PermissionChange
	updateCalled bool
	updateErr    error
	filter       domain.AuditLogFilter
	auditPage    domain.AuditLogPage
}

func (s *accessServiceStub) ListRoles(
	context.Context, domain.Identity,
) ([]domain.RoleSummary, error) {
	return []domain.RoleSummary{}, nil
}

func (s *accessServiceStub) PermissionMatrix(
	context.Context, domain.Identity,
) ([]domain.Permission, error) {
	return []domain.Permission{}, nil
}

func (s *accessServiceStub) UpdatePermission(
	_ context.Context, _ domain.Identity, change domain.PermissionChange, _ service.RequestMeta,
) (uuid.UUID, error) {
	s.change, s.updateCalled = change, true
	if s.updateErr != nil {
		return uuid.Nil, s.updateErr
	}
	return uuid.New(), nil
}

func (s *accessServiceStub) ListAuditLogs(
	_ context.Context, _ domain.Identity, filter domain.AuditLogFilter,
) (domain.AuditLogPage, error) {
	s.filter = filter
	return s.auditPage, nil
}

func humanResourceIdentity() domain.Identity {
	return domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR}
}

func serveAccess(
	handlerFunc http.HandlerFunc, method, target, body string, identity domain.Identity,
) *httptest.ResponseRecorder {
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, target, nil)
	} else {
		request = httptest.NewRequest(method, target, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}
	request = request.WithContext(middleware.WithIdentity(request.Context(), identity))
	recorder := httptest.NewRecorder()
	handlerFunc(recorder, request)
	return recorder
}

func newAccessHandlerForTest(stub *accessServiceStub) *AccessHandler {
	return NewAccessHandler(stub, validation.New(), false)
}

func TestUpdatePermissionDecodesContractBody(t *testing.T) {
	stub := &accessServiceStub{}
	roleID := uuid.New()

	recorder := serveAccess(
		newAccessHandlerForTest(stub).UpdatePermission, http.MethodPut,
		"/api/v1/akses/permission",
		`{"role_id":"`+roleID.String()+`","modul":"lembur","aksi":"approve","is_allowed":true}`,
		humanResourceIdentity(),
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assert.Equal(t, domain.PermissionChange{
		RoleID:    roleID,
		Module:    "lembur",
		Action:    "approve",
		IsAllowed: true,
	}, stub.change)
}

// `is_allowed: false` harus tersimpan sebagai pencabutan, bukan dianggap field hilang.
func TestUpdatePermissionAcceptsExplicitFalse(t *testing.T) {
	stub := &accessServiceStub{}

	recorder := serveAccess(
		newAccessHandlerForTest(stub).UpdatePermission, http.MethodPut,
		"/api/v1/akses/permission",
		`{"role_id":"`+uuid.NewString()+`","modul":"audit","aksi":"read","is_allowed":false}`,
		humanResourceIdentity(),
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assert.False(t, stub.change.IsAllowed)
}

func TestUpdatePermissionRequiresIsAllowed(t *testing.T) {
	stub := &accessServiceStub{}

	recorder := serveAccess(
		newAccessHandlerForTest(stub).UpdatePermission, http.MethodPut,
		"/api/v1/akses/permission",
		`{"role_id":"`+uuid.NewString()+`","modul":"audit","aksi":"read"}`,
		humanResourceIdentity(),
	)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.False(t, stub.updateCalled)
}

func TestUpdatePermissionRejectsActionOutsideEnum(t *testing.T) {
	stub := &accessServiceStub{}

	recorder := serveAccess(
		newAccessHandlerForTest(stub).UpdatePermission, http.MethodPut,
		"/api/v1/akses/permission",
		`{"role_id":"`+uuid.NewString()+`","modul":"audit","aksi":"manage","is_allowed":true}`,
		humanResourceIdentity(),
	)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.False(t, stub.updateCalled)
}

// Field asing ditolak agar body tidak menyelundupkan atribut di luar kontrak.
func TestUpdatePermissionRejectsUnknownFields(t *testing.T) {
	stub := &accessServiceStub{}

	recorder := serveAccess(
		newAccessHandlerForTest(stub).UpdatePermission, http.MethodPut,
		"/api/v1/akses/permission",
		`{"role_id":"`+uuid.NewString()+`","modul":"audit","aksi":"read","is_allowed":true,"scope":"*"}`,
		humanResourceIdentity(),
	)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.False(t, stub.updateCalled)
}

func TestUpdatePermissionMapsInvariantViolationToValidationError(t *testing.T) {
	stub := &accessServiceStub{updateErr: domain.ErrPermissionInvariant}

	recorder := serveAccess(
		newAccessHandlerForTest(stub).UpdatePermission, http.MethodPut,
		"/api/v1/akses/permission",
		`{"role_id":"`+uuid.NewString()+`","modul":"akses","aksi":"update","is_allowed":true}`,
		humanResourceIdentity(),
	)

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "VALIDATION_ERROR")
}

func TestUpdatePermissionMapsForbidden(t *testing.T) {
	stub := &accessServiceStub{updateErr: domain.ErrForbidden}

	recorder := serveAccess(
		newAccessHandlerForTest(stub).UpdatePermission, http.MethodPut,
		"/api/v1/akses/permission",
		`{"role_id":"`+uuid.NewString()+`","modul":"audit","aksi":"read","is_allowed":true}`,
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleTopManagement},
	)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestAuditLogFilterParsesContractParameters(t *testing.T) {
	stub := &accessServiceStub{}
	userID := uuid.New()

	recorder := serveAccess(
		newAccessHandlerForTest(stub).ListAuditLogs, http.MethodGet,
		"/api/v1/akses/audit-log?page=3&limit=20&user_id="+userID.String()+
			"&aksi=APPROVE&modul=ketidakhadiran&tanggal_mulai=2026-08-01&tanggal_selesai=2026-08-31",
		"", humanResourceIdentity(),
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assert.Equal(t, 3, stub.filter.Page)
	assert.Equal(t, 20, stub.filter.Limit)
	require.NotNil(t, stub.filter.UserID)
	assert.Equal(t, userID, *stub.filter.UserID)
	require.NotNil(t, stub.filter.Action)
	assert.Equal(t, "APPROVE", *stub.filter.Action)
	require.NotNil(t, stub.filter.Module)
	assert.Equal(t, "ketidakhadiran", *stub.filter.Module)

	// Batas atas eksklusif hari berikutnya agar tanggal_selesai ikut terhitung penuh.
	require.NotNil(t, stub.filter.StartDate)
	require.NotNil(t, stub.filter.EndDate)
	assert.Equal(t, "2026-07-31", stub.filter.StartDate.Format(domain.DateLayout))
	assert.Equal(t, "2026-08-31", stub.filter.EndDate.Format(domain.DateLayout))
}

func TestAuditLogFilterRejectsMalformedParameters(t *testing.T) {
	cases := []string{
		"/api/v1/akses/audit-log?user_id=bukan-uuid",
		"/api/v1/akses/audit-log?tanggal_mulai=01-08-2026",
		"/api/v1/akses/audit-log?tanggal_selesai=2026-13-40",
		"/api/v1/akses/audit-log?page=nol",
	}

	for _, target := range cases {
		t.Run(target, func(t *testing.T) {
			stub := &accessServiceStub{}

			recorder := serveAccess(
				newAccessHandlerForTest(stub).ListAuditLogs, http.MethodGet, target, "",
				humanResourceIdentity(),
			)

			assert.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
		})
	}
}

func TestAuditLogListReturnsPaginationMeta(t *testing.T) {
	stub := &accessServiceStub{auditPage: domain.AuditLogPage{
		Page: 1, Limit: 10, Total: 21,
		Items: []domain.AuditLogEntry{{ID: uuid.New(), Action: "LOGIN", Module: "auth"}},
	}}

	recorder := serveAccess(
		newAccessHandlerForTest(stub).ListAuditLogs, http.MethodGet,
		"/api/v1/akses/audit-log", "", humanResourceIdentity(),
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	var body struct {
		Meta struct {
			TotalData int `json:"total_data"`
			TotalPage int `json:"total_page"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, 21, body.Meta.TotalData)
	assert.Equal(t, 3, body.Meta.TotalPage)
}

// Audit log dengan aktor sistem mengembalikan user_id null (D-029).
func TestAuditLogRendersSystemActorAsNull(t *testing.T) {
	stub := &accessServiceStub{auditPage: domain.AuditLogPage{
		Page: 1, Limit: 10, Total: 1,
		Items: []domain.AuditLogEntry{{
			ID: uuid.New(), Action: "AUTO_ESCALATE", Module: "ketidakhadiran",
		}},
	}}

	recorder := serveAccess(
		newAccessHandlerForTest(stub).ListAuditLogs, http.MethodGet,
		"/api/v1/akses/audit-log", "", humanResourceIdentity(),
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	assert.Nil(t, body.Data[0]["user_id"])
	assert.Nil(t, body.Data[0]["nama_user"])
}

func TestAccessHandlersRequireIdentity(t *testing.T) {
	stub := &accessServiceStub{}
	handler := newAccessHandlerForTest(stub)

	for _, handlerFunc := range []http.HandlerFunc{
		handler.ListRoles, handler.PermissionMatrix, handler.ListAuditLogs, handler.UpdatePermission,
	} {
		recorder := httptest.NewRecorder()
		handlerFunc(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/akses/role", nil))

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	}
	assert.False(t, stub.updateCalled)
}
