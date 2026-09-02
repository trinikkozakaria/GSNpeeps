package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type officeLocationServiceStub struct {
	createInput domain.OfficeLocationInput
	createCalls int
	createErr   error
	updateID    uuid.UUID
	updateCalls int
	updateErr   error
	deleteID    uuid.UUID
	deleteCalls int
	deleteErr   error
}

func (s *officeLocationServiceStub) ListOfficeLocations(context.Context) ([]domain.OfficeLocation, error) {
	return nil, nil
}

func (s *officeLocationServiceStub) CreateOfficeLocation(_ context.Context, _ domain.Identity, input domain.OfficeLocationInput, _ service.RequestMeta) (domain.OfficeLocation, error) {
	s.createCalls++
	s.createInput = input
	if s.createErr != nil {
		return domain.OfficeLocation{}, s.createErr
	}
	return domain.OfficeLocation{ID: uuid.New(), Code: input.Code, Name: input.Name, Latitude: input.Latitude, Longitude: input.Longitude, IsActive: input.IsActive}, nil
}

func (s *officeLocationServiceStub) UpdateOfficeLocation(_ context.Context, _ domain.Identity, id uuid.UUID, input domain.OfficeLocationInput, _ service.RequestMeta) (domain.OfficeLocation, error) {
	s.updateCalls++
	s.updateID = id
	if s.updateErr != nil {
		return domain.OfficeLocation{}, s.updateErr
	}
	return domain.OfficeLocation{ID: id, Code: input.Code, Name: input.Name, Latitude: input.Latitude, Longitude: input.Longitude, IsActive: input.IsActive}, nil
}

func (s *officeLocationServiceStub) DeactivateOfficeLocation(_ context.Context, _ domain.Identity, id uuid.UUID, _ service.RequestMeta) error {
	s.deleteCalls++
	s.deleteID = id
	return s.deleteErr
}

func (s *officeLocationServiceStub) Record(context.Context, domain.Identity, domain.RecordAttendance, service.RequestMeta) (domain.Attendance, error) {
	return domain.Attendance{}, nil
}

func (s *officeLocationServiceStub) LiveFeed(context.Context, domain.Identity, string) ([]domain.AttendanceLiveFeedItem, error) {
	return nil, nil
}

func (s *officeLocationServiceStub) ExportLiveFeed(context.Context, domain.Identity, string, service.RequestMeta) (domain.ExportFile, error) {
	return domain.ExportFile{}, nil
}

func (s *officeLocationServiceStub) Report(context.Context, domain.Identity, service.ReportQuery) (domain.AttendanceReportPage, error) {
	return domain.AttendanceReportPage{}, nil
}

func (s *officeLocationServiceStub) ExportReport(context.Context, domain.Identity, service.ReportQuery, domain.ExportFormat, service.RequestMeta) (domain.ExportFile, error) {
	return domain.ExportFile{}, nil
}

func newOfficeLocationRouter(stub *officeLocationServiceStub) http.Handler {
	handler := NewAttendanceHandler(stub, false)
	router := mux.NewRouter()
	router.HandleFunc("/master/lokasi-kantor", handler.CreateOfficeLocation).Methods(http.MethodPost)
	router.HandleFunc("/master/lokasi-kantor/{id}", handler.UpdateOfficeLocation).Methods(http.MethodPut)
	router.HandleFunc("/master/lokasi-kantor/{id}", handler.DeactivateOfficeLocation).Methods(http.MethodDelete)
	return router
}

func serveOffice(stub *officeLocationServiceStub, method, target, body string) *httptest.ResponseRecorder {
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, target, nil)
	} else {
		request = httptest.NewRequest(method, target, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}
	request = request.WithContext(middleware.WithIdentity(request.Context(), domain.Identity{
		UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR,
	}))
	recorder := httptest.NewRecorder()
	newOfficeLocationRouter(stub).ServeHTTP(recorder, request)
	return recorder
}

func TestCreateOfficeLocationHandlerAcceptsValidBody(t *testing.T) {
	stub := &officeLocationServiceStub{}

	recorder := serveOffice(stub, http.MethodPost, "/master/lokasi-kantor",
		`{"kode":"HQ","nama":"Kantor Pusat","alamat":"Jl. Contoh 1","latitude":-6.2,"longitude":106.8,"is_active":true}`)

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Equal(t, 1, stub.createCalls)
	assert.Equal(t, "HQ", stub.createInput.Code)
	assert.InDelta(t, 106.8, stub.createInput.Longitude, 1e-9)
}

func TestCreateOfficeLocationHandlerRejectsOutOfRangeCoordinates(t *testing.T) {
	stub := &officeLocationServiceStub{}

	recorder := serveOffice(stub, http.MethodPost, "/master/lokasi-kantor",
		`{"kode":"HQ","nama":"Kantor Pusat","latitude":200,"longitude":106.8}`)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Zero(t, stub.createCalls, "service tidak dipanggil untuk koordinat invalid")
}

func TestCreateOfficeLocationHandlerRejectsBlankCode(t *testing.T) {
	stub := &officeLocationServiceStub{}

	recorder := serveOffice(stub, http.MethodPost, "/master/lokasi-kantor",
		`{"kode":"   ","nama":"Kantor Pusat","latitude":-6.2,"longitude":106.8}`)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Zero(t, stub.createCalls)
}

func TestCreateOfficeLocationHandlerPropagatesForbidden(t *testing.T) {
	stub := &officeLocationServiceStub{createErr: domain.ErrForbidden}

	recorder := serveOffice(stub, http.MethodPost, "/master/lokasi-kantor",
		`{"kode":"HQ","nama":"Kantor Pusat","latitude":-6.2,"longitude":106.8}`)

	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestUpdateOfficeLocationHandlerRejectsInvalidUUID(t *testing.T) {
	stub := &officeLocationServiceStub{}

	recorder := serveOffice(stub, http.MethodPut, "/master/lokasi-kantor/not-a-uuid",
		`{"kode":"HQ","nama":"Kantor Pusat","latitude":-6.2,"longitude":106.8}`)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Zero(t, stub.updateCalls)
}

func TestUpdateOfficeLocationHandlerPropagatesNotFound(t *testing.T) {
	stub := &officeLocationServiceStub{updateErr: domain.ErrNotFound}
	id := uuid.NewString()

	recorder := serveOffice(stub, http.MethodPut, "/master/lokasi-kantor/"+id,
		`{"kode":"HQ","nama":"Kantor Pusat","latitude":-6.2,"longitude":106.8}`)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, id, stub.updateID.String())
}

func TestDeactivateOfficeLocationHandlerReturnsSuccessEnvelope(t *testing.T) {
	stub := &officeLocationServiceStub{}
	id := uuid.NewString()

	recorder := serveOffice(stub, http.MethodDelete, "/master/lokasi-kantor/"+id, "")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, 1, stub.deleteCalls)
	assert.Equal(t, id, stub.deleteID.String())

	var envelope struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	assert.True(t, envelope.Success)
}

func TestDeactivateOfficeLocationHandlerPropagatesNotFound(t *testing.T) {
	stub := &officeLocationServiceStub{deleteErr: domain.ErrNotFound}

	recorder := serveOffice(stub, http.MethodDelete, "/master/lokasi-kantor/"+uuid.NewString(), "")

	require.Equal(t, http.StatusNotFound, recorder.Code)
}
