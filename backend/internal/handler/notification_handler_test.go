package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type notificationServiceStub struct {
	page        domain.NotificationPage
	unread      int
	isRead      *bool
	page_       int
	limit       int
	markCalled  bool
	markErr     error
	dismissed   bool
	dismissErr  error
	identitySaw domain.Identity
}

func (s *notificationServiceStub) List(
	_ context.Context, identity domain.Identity, isRead *bool, page, limit int,
) (domain.NotificationPage, error) {
	s.identitySaw, s.isRead, s.page_, s.limit = identity, isRead, page, limit
	return s.page, nil
}

func (s *notificationServiceStub) UnreadCount(
	_ context.Context, identity domain.Identity,
) (int, error) {
	s.identitySaw = identity
	return s.unread, nil
}

func (s *notificationServiceStub) MarkRead(
	_ context.Context, identity domain.Identity, _ uuid.UUID,
) error {
	s.identitySaw, s.markCalled = identity, true
	return s.markErr
}

func (s *notificationServiceStub) Dismiss(
	_ context.Context, identity domain.Identity, _ uuid.UUID,
) error {
	s.identitySaw, s.dismissed = identity, true
	return s.dismissErr
}

func notificationIdentity() domain.Identity {
	return domain.Identity{
		UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee,
	}
}

func serveNotification(
	handlerFunc http.HandlerFunc,
	method, target string,
	identity domain.Identity,
	vars map[string]string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, nil)
	request = request.WithContext(middleware.WithIdentity(request.Context(), identity))
	if vars != nil {
		request = mux.SetURLVars(request, vars)
	}
	recorder := httptest.NewRecorder()
	handlerFunc(recorder, request)
	return recorder
}

func TestNotificationListReturnsPaginationMeta(t *testing.T) {
	stub := &notificationServiceStub{page: domain.NotificationPage{
		Page: 2, Limit: 5, Total: 7,
		Items: []domain.Notification{{ID: uuid.New(), Title: "Pengajuan baru"}},
	}}
	handler := NewNotificationHandler(stub)

	recorder := serveNotification(
		handler.List, http.MethodGet, "/api/v1/notifikasi?page=2&limit=5",
		notificationIdentity(), nil,
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var body struct {
		Success bool `json:"success"`
		Meta    struct {
			Page      int `json:"page"`
			Limit     int `json:"limit"`
			TotalData int `json:"total_data"`
			TotalPage int `json:"total_page"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.True(t, body.Success)
	assert.Equal(t, 2, body.Meta.Page)
	assert.Equal(t, 5, body.Meta.Limit)
	assert.Equal(t, 7, body.Meta.TotalData)
	assert.Equal(t, 2, body.Meta.TotalPage)
	assert.Equal(t, 2, stub.page_)
	assert.Equal(t, 5, stub.limit)
}

func TestNotificationListParsesIsReadFilter(t *testing.T) {
	stub := &notificationServiceStub{}
	handler := NewNotificationHandler(stub)

	recorder := serveNotification(
		handler.List, http.MethodGet, "/api/v1/notifikasi?is_read=true",
		notificationIdentity(), nil,
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, stub.isRead)
	assert.True(t, *stub.isRead)
}

// Nilai boolean yang tidak valid ditolak, bukan diam-diam dianggap tanpa filter.
func TestNotificationListRejectsInvalidIsReadFilter(t *testing.T) {
	handler := NewNotificationHandler(&notificationServiceStub{})

	recorder := serveNotification(
		handler.List, http.MethodGet, "/api/v1/notifikasi?is_read=maybe",
		notificationIdentity(), nil,
	)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "INVALID_PARAM")
}

func TestUnreadCountReturnsContractShape(t *testing.T) {
	handler := NewNotificationHandler(&notificationServiceStub{unread: 12})

	recorder := serveNotification(
		handler.UnreadCount, http.MethodGet, "/api/v1/notifikasi/unread-count",
		notificationIdentity(), nil,
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	var body struct {
		Data struct {
			UnreadCount int `json:"unread_count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, 12, body.Data.UnreadCount)
}

// Notifikasi milik pengguna lain menghasilkan 403 seragam (D-032).
func TestMarkReadMapsForbiddenWithoutLeakingExistence(t *testing.T) {
	handler := NewNotificationHandler(&notificationServiceStub{markErr: domain.ErrForbidden})

	recorder := serveNotification(
		handler.MarkRead, http.MethodPut, "/api/v1/notifikasi/x/read",
		notificationIdentity(), map[string]string{"id": uuid.NewString()},
	)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "FORBIDDEN")
	assert.NotContains(t, recorder.Body.String(), "NOT_FOUND")
}

func TestDismissMapsForbiddenWithoutLeakingExistence(t *testing.T) {
	handler := NewNotificationHandler(&notificationServiceStub{dismissErr: domain.ErrForbidden})

	recorder := serveNotification(
		handler.Dismiss, http.MethodDelete, "/api/v1/notifikasi/x",
		notificationIdentity(), map[string]string{"id": uuid.NewString()},
	)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestNotificationIDMustBeUUID(t *testing.T) {
	stub := &notificationServiceStub{}
	handler := NewNotificationHandler(stub)

	recorder := serveNotification(
		handler.MarkRead, http.MethodPut, "/api/v1/notifikasi/bukan-uuid/read",
		notificationIdentity(), map[string]string{"id": "bukan-uuid"},
	)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.False(t, stub.markCalled)
}

// Tanpa identity pada context, handler tidak boleh memanggil service.
func TestNotificationHandlersRequireIdentity(t *testing.T) {
	stub := &notificationServiceStub{}
	handler := NewNotificationHandler(stub)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/notifikasi", nil)
	recorder := httptest.NewRecorder()
	handler.List(recorder, request)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Zero(t, stub.limit)
}

func TestDismissReturnsEmptySuccessEnvelope(t *testing.T) {
	stub := &notificationServiceStub{}
	handler := NewNotificationHandler(stub)

	recorder := serveNotification(
		handler.Dismiss, http.MethodDelete, "/api/v1/notifikasi/x",
		notificationIdentity(), map[string]string{"id": uuid.NewString()},
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, stub.dismissed)
	var body struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Message string          `json:"message"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.True(t, body.Success)
	assert.JSONEq(t, "null", string(body.Data))
	assert.NotEmpty(t, body.Message)
}

// Notifikasi yang dikembalikan tidak boleh membawa timestamp internal selain kontrak.
func TestNotificationResponseFollowsContractFields(t *testing.T) {
	readAt := time.Date(2026, 8, 3, 2, 0, 0, 0, time.UTC)
	referenceID := uuid.New()
	referenceType := "ketidakhadiran"
	stub := &notificationServiceStub{page: domain.NotificationPage{
		Page: 1, Limit: 10, Total: 1,
		Items: []domain.Notification{{
			ID:            uuid.New(),
			UserID:        uuid.New(),
			Type:          "ketidakhadiran_baru",
			Title:         "Pengajuan ketidakhadiran baru",
			Message:       "Ada pengajuan ketidakhadiran yang menunggu keputusan Anda.",
			ReferenceID:   &referenceID,
			ReferenceType: &referenceType,
			IsRead:        true,
			ReadAt:        &readAt,
			CreatedAt:     readAt,
		}},
	}}
	handler := NewNotificationHandler(stub)

	recorder := serveNotification(
		handler.List, http.MethodGet, "/api/v1/notifikasi", notificationIdentity(), nil,
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)

	item := body.Data[0]
	for _, field := range []string{
		"id", "user_id", "tipe", "judul", "pesan",
		"reference_id", "reference_type", "is_read", "read_at", "created_at",
	} {
		assert.Containsf(t, item, field, "field %s wajib ada pada response", field)
	}
	assert.NotContains(t, item, "event_key")
	assert.NotContains(t, item, "dismissed_at")
}
