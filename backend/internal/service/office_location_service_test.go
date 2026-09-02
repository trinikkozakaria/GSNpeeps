package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// officeStoreStub memakai attendanceStoreStub untuk method absensi lain dan merekam
// mutasi lokasi kantor agar HR-gate serta audit dapat diverifikasi tanpa PostgreSQL.
type officeStoreStub struct {
	*attendanceStoreStub
	createInput   domain.OfficeLocationInput
	createHits    int
	createErr     error
	updateInput   domain.OfficeLocationInput
	updateID      uuid.UUID
	updateHits    int
	updateErr     error
	deactivateID  uuid.UUID
	deactivateHit int
	deactivateErr error
}

func (s *officeStoreStub) CreateOfficeLocation(_ context.Context, input domain.OfficeLocationInput) (domain.OfficeLocation, error) {
	s.createHits++
	s.createInput = input
	if s.createErr != nil {
		return domain.OfficeLocation{}, s.createErr
	}
	return domain.OfficeLocation{ID: uuid.New(), Code: input.Code, Name: input.Name, Address: input.Address, Latitude: input.Latitude, Longitude: input.Longitude, IsActive: input.IsActive}, nil
}

func (s *officeStoreStub) UpdateOfficeLocation(_ context.Context, id uuid.UUID, input domain.OfficeLocationInput) (domain.OfficeLocation, error) {
	s.updateHits++
	s.updateID = id
	s.updateInput = input
	if s.updateErr != nil {
		return domain.OfficeLocation{}, s.updateErr
	}
	return domain.OfficeLocation{ID: id, Code: input.Code, Name: input.Name, Address: input.Address, Latitude: input.Latitude, Longitude: input.Longitude, IsActive: input.IsActive}, nil
}

func (s *officeStoreStub) DeactivateOfficeLocation(_ context.Context, id uuid.UUID) error {
	s.deactivateHit++
	s.deactivateID = id
	return s.deactivateErr
}

func newOfficeServiceFixture(t *testing.T) (*AttendanceService, *officeStoreStub, *recordingAudit) {
	t.Helper()
	store := &officeStoreStub{attendanceStoreStub: &attendanceStoreStub{existing: map[string]bool{}}}
	audit := &recordingAudit{}
	service := NewAttendanceService(store, transactionStub{}, audit, &documentStoreStub{})
	service.now = func() time.Time { return time.Date(2026, time.September, 2, 3, 0, 0, 0, time.UTC) }
	return service, store, audit
}

func sampleOfficeInput() domain.OfficeLocationInput {
	addr := "Jl. Contoh No. 1"
	return domain.OfficeLocationInput{Code: "HQ", Name: "Kantor Pusat", Address: &addr, Latitude: -6.2, Longitude: 106.8, IsActive: true}
}

func nonHRRoles() []domain.RoleName {
	return []domain.RoleName{domain.RoleEmployee, domain.RoleSupervisor, domain.RoleTopManagement}
}

func TestCreateOfficeLocationRequiresHRAndWritesAudit(t *testing.T) {
	service, store, audit := newOfficeServiceFixture(t)
	identity := domain.Identity{UserID: uuid.New(), Role: domain.RoleHR}

	created, err := service.CreateOfficeLocation(context.Background(), identity, sampleOfficeInput(), RequestMeta{RequestID: "req-a"})
	require.NoError(t, err)
	assert.Equal(t, "HQ", created.Code)
	assert.Equal(t, 1, store.createHits)
	assert.Equal(t, sampleOfficeInput().Latitude, store.createInput.Latitude)

	require.Len(t, audit.entries, 1)
	assert.Equal(t, "CREATE", audit.entries[0].Action)
	assert.Equal(t, "master_lokasi_kantor", audit.entries[0].Module)
	require.NotNil(t, audit.entries[0].DataID)
	assert.Equal(t, "req-a", audit.entries[0].Detail["request_id"])
}

func TestCreateOfficeLocationRejectsNonHR(t *testing.T) {
	for _, role := range nonHRRoles() {
		service, store, audit := newOfficeServiceFixture(t)

		_, err := service.CreateOfficeLocation(context.Background(), domain.Identity{Role: role}, sampleOfficeInput(), RequestMeta{})
		require.ErrorIsf(t, err, domain.ErrForbidden, "role %s ditolak", role)
		assert.Zero(t, store.createHits, "tidak ada mutasi untuk role %s", role)
		assert.Empty(t, audit.entries, "tidak ada audit untuk role %s", role)
	}
}

func TestUpdateOfficeLocationRequiresHRAndWritesAudit(t *testing.T) {
	service, store, audit := newOfficeServiceFixture(t)
	id := uuid.New()

	updated, err := service.UpdateOfficeLocation(context.Background(), domain.Identity{UserID: uuid.New(), Role: domain.RoleHR}, id, sampleOfficeInput(), RequestMeta{RequestID: "req-b"})
	require.NoError(t, err)
	assert.Equal(t, id, updated.ID)
	assert.Equal(t, id, store.updateID)
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "UPDATE", audit.entries[0].Action)
	assert.Equal(t, "master_lokasi_kantor", audit.entries[0].Module)
}

func TestUpdateOfficeLocationPropagatesNotFound(t *testing.T) {
	service, store, audit := newOfficeServiceFixture(t)
	store.updateErr = repository.ErrNotFound

	_, err := service.UpdateOfficeLocation(context.Background(), domain.Identity{UserID: uuid.New(), Role: domain.RoleHR}, uuid.New(), sampleOfficeInput(), RequestMeta{})
	require.ErrorIs(t, err, repository.ErrNotFound)
	assert.Empty(t, audit.entries, "audit tidak ditulis bila mutasi gagal")
}

func TestUpdateOfficeLocationRejectsNonHR(t *testing.T) {
	for _, role := range nonHRRoles() {
		service, store, _ := newOfficeServiceFixture(t)
		_, err := service.UpdateOfficeLocation(context.Background(), domain.Identity{Role: role}, uuid.New(), sampleOfficeInput(), RequestMeta{})
		require.ErrorIsf(t, err, domain.ErrForbidden, "role %s ditolak", role)
		assert.Zero(t, store.updateHits)
	}
}

func TestDeactivateOfficeLocationRequiresHRAndWritesAudit(t *testing.T) {
	service, store, audit := newOfficeServiceFixture(t)
	id := uuid.New()

	err := service.DeactivateOfficeLocation(context.Background(), domain.Identity{UserID: uuid.New(), Role: domain.RoleHR}, id, RequestMeta{RequestID: "req-c"})
	require.NoError(t, err)
	assert.Equal(t, 1, store.deactivateHit)
	assert.Equal(t, id, store.deactivateID)
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "DELETE", audit.entries[0].Action)
	assert.Equal(t, "master_lokasi_kantor", audit.entries[0].Module)
}

func TestDeactivateOfficeLocationRejectsNonHR(t *testing.T) {
	for _, role := range nonHRRoles() {
		service, store, audit := newOfficeServiceFixture(t)
		err := service.DeactivateOfficeLocation(context.Background(), domain.Identity{Role: role}, uuid.New(), RequestMeta{})
		require.ErrorIsf(t, err, domain.ErrForbidden, "role %s ditolak", role)
		assert.Zero(t, store.deactivateHit)
		assert.Empty(t, audit.entries)
	}
}

// ListOfficeLocations tidak menerima identity: kontrak membukanya untuk seluruh role
// terautentikasi karena menyuplai dropdown WFO. HR-gate hanya pada mutasi.
func TestListOfficeLocationsHasNoHRGate(t *testing.T) {
	service, store, _ := newOfficeServiceFixture(t)
	store.attendanceStoreStub.office = domain.OfficeLocation{ID: uuid.New(), Code: "HQ", IsActive: true}

	locations, err := service.ListOfficeLocations(context.Background())
	require.NoError(t, err)
	assert.Len(t, locations, 1)
}
