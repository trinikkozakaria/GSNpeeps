package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Koordinat sintetis untuk kantor uji; bukan lokasi nyata.
const (
	testOfficeLat = -6.2000000
	testOfficeLon = 106.8000000
)

type attendanceStoreStub struct {
	office        domain.OfficeLocation
	officeErr     error
	existing      map[string]bool
	created       domain.AttendanceRow
	createErr     error
	feed          []domain.AttendanceLiveFeedItem
	report        domain.AttendanceReportPage
	reportFilter  domain.AttendanceReportFilter
	reportWorkday int
}

func (s *attendanceStoreStub) ListActiveOfficeLocations(
	context.Context,
) ([]domain.OfficeLocation, error) {
	if s.officeErr != nil {
		return nil, s.officeErr
	}
	if s.office.ID == uuid.Nil {
		return []domain.OfficeLocation{}, nil
	}
	return []domain.OfficeLocation{s.office}, nil
}

func (s *attendanceStoreStub) FindActiveOfficeLocation(
	context.Context, uuid.UUID,
) (domain.OfficeLocation, error) {
	return s.office, s.officeErr
}

func (s *attendanceStoreStub) ExistsForDate(
	_ context.Context, _ uuid.UUID, _ string, attendanceType domain.AttendanceType,
) (bool, error) {
	return s.existing[string(attendanceType)], nil
}

func (s *attendanceStoreStub) Create(
	_ context.Context, row domain.AttendanceRow,
) (domain.Attendance, error) {
	s.created = row
	if s.createErr != nil {
		return domain.Attendance{}, s.createErr
	}
	return domain.Attendance{
		ID: uuid.New(), Date: row.Date, Type: row.Type, WorkMode: row.WorkMode,
		Status: row.Status, DistanceMeters: row.DistanceMeters,
	}, nil
}

func (s *attendanceStoreStub) LiveFeed(
	context.Context, string,
) ([]domain.AttendanceLiveFeedItem, error) {
	return s.feed, nil
}

func (s *attendanceStoreStub) Report(
	_ context.Context, filter domain.AttendanceReportFilter, workingDays int,
) (domain.AttendanceReportPage, error) {
	s.reportFilter = filter
	s.reportWorkday = workingDays
	return s.report, nil
}

// Senin 3 Agustus 2026 pukul 08:30 WIB adalah hari kerja sebelum batas keterlambatan.
func workingMonday() time.Time {
	return time.Date(2026, time.August, 3, 8, 30, 0, 0, domain.Jakarta())
}

func newAttendanceServiceForTest(
	store AttendanceStore,
	photos DocumentStore,
	transactions EmployeeTransactionManager,
	now time.Time,
) *AttendanceService {
	service := NewAttendanceService(store, transactions, auditStub{}, photos)
	service.now = func() time.Time { return now }
	return service
}

func attendanceCommand(mode domain.WorkMode, lat, lon float64, office *uuid.UUID) domain.RecordAttendance {
	return domain.RecordAttendance{
		Type:             domain.AttendanceCheckIn,
		WorkMode:         mode,
		Latitude:         lat,
		Longitude:        lon,
		OfficeLocationID: office,
		PhotoExtension:   ".jpg",
		PhotoMediaType:   "image/jpeg",
		PhotoContent:     []byte{0xFF, 0xD8, 0xFF, 0xE0, 'x'},
	}
}

func employeeIdentity() domain.Identity {
	return domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee}
}

func activeOffice() domain.OfficeLocation {
	return domain.OfficeLocation{
		ID: uuid.New(), Latitude: testOfficeLat, Longitude: testOfficeLon, IsActive: true,
	}
}

func TestAttendanceRejectsWeekend(t *testing.T) {
	store := &attendanceStoreStub{existing: map[string]bool{}}
	photos := &documentStoreStub{}
	// Sabtu 8 Agustus 2026.
	saturday := time.Date(2026, time.August, 8, 9, 0, 0, 0, domain.Jakarta())
	service := newAttendanceServiceForTest(store, photos, transactionStub{}, saturday)

	_, err := service.Record(
		context.Background(), employeeIdentity(),
		attendanceCommand(domain.WorkModeWFH, testOfficeLat, testOfficeLon, nil),
		RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrNonWorkingDay)
	assert.Empty(t, photos.uploadedPath, "hari libur tidak boleh mengunggah foto")
}

// WFO ditolak hanya bila melebihi 100 meter dari koordinat tepercaya kantor.
func TestAttendanceWFORadiusBoundary(t *testing.T) {
	metersPerDegree := 6371008.8 * 3.141592653589793 / 180
	cases := []struct {
		name     string
		meters   float64
		accepted bool
	}{
		{"tepat di kantor", 0, true},
		{"99 meter", 99, true},
		{"tepat 100 meter", 100, true},
		{"120 meter", 120, false},
		{"500 meter", 500, false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			office := activeOffice()
			store := &attendanceStoreStub{office: office, existing: map[string]bool{}}
			photos := &documentStoreStub{}
			service := newAttendanceServiceForTest(store, photos, transactionStub{}, workingMonday())

			latitude := testOfficeLat + testCase.meters/metersPerDegree
			record, err := service.Record(
				context.Background(), employeeIdentity(),
				attendanceCommand(domain.WorkModeWFO, latitude, testOfficeLon, &office.ID),
				RequestMeta{},
			)

			if !testCase.accepted {
				require.ErrorIs(t, err, domain.ErrOutOfRadius)
				assert.Empty(t, photos.uploadedPath, "request di luar radius tidak mengunggah foto")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, record.DistanceMeters)
			assert.InDelta(t, testCase.meters, *record.DistanceMeters, 0.5)
			assert.Equal(t, office.ID, *store.created.OfficeLocationID)
		})
	}
}

// WFH dan WFA menyimpan koordinat tetapi tidak menjalankan validasi radius.
func TestAttendanceWFHAndWFAAreNotRadiusRejected(t *testing.T) {
	for _, mode := range []domain.WorkMode{domain.WorkModeWFH, domain.WorkModeWFA} {
		store := &attendanceStoreStub{existing: map[string]bool{}}
		service := newAttendanceServiceForTest(
			store, &documentStoreStub{}, transactionStub{}, workingMonday(),
		)

		// Koordinat jauh dari kantor mana pun.
		_, err := service.Record(
			context.Background(), employeeIdentity(),
			attendanceCommand(mode, -8.65, 115.21, nil),
			RequestMeta{},
		)

		require.NoErrorf(t, err, "mode %s tidak boleh ditolak radius", mode)
		assert.Nil(t, store.created.OfficeLocationID, "mode non-WFO tidak menyimpan lokasi kantor")
		assert.Nil(t, store.created.DistanceMeters)
	}
}

func TestAttendanceWFORequiresActiveOffice(t *testing.T) {
	office := activeOffice()
	store := &attendanceStoreStub{officeErr: repository.ErrNotFound, existing: map[string]bool{}}
	service := newAttendanceServiceForTest(
		store, &documentStoreStub{}, transactionStub{}, workingMonday(),
	)

	_, err := service.Record(
		context.Background(), employeeIdentity(),
		attendanceCommand(domain.WorkModeWFO, testOfficeLat, testOfficeLon, &office.ID),
		RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrInvalidOfficeLocation)
}

func TestAttendanceRejectsDuplicateCheckIn(t *testing.T) {
	store := &attendanceStoreStub{existing: map[string]bool{"check_in": true}}
	photos := &documentStoreStub{}
	service := newAttendanceServiceForTest(store, photos, transactionStub{}, workingMonday())

	_, err := service.Record(
		context.Background(), employeeIdentity(),
		attendanceCommand(domain.WorkModeWFH, testOfficeLat, testOfficeLon, nil),
		RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrDuplicateCheckIn)
	assert.Empty(t, photos.uploadedPath)
}

func TestAttendanceRejectsCheckoutWithoutCheckIn(t *testing.T) {
	store := &attendanceStoreStub{existing: map[string]bool{}}
	service := newAttendanceServiceForTest(
		store, &documentStoreStub{}, transactionStub{}, workingMonday(),
	)
	command := attendanceCommand(domain.WorkModeWFH, testOfficeLat, testOfficeLon, nil)
	command.Type = domain.AttendanceCheckOut

	_, err := service.Record(context.Background(), employeeIdentity(), command, RequestMeta{})

	require.ErrorIs(t, err, domain.ErrCheckoutWithoutCheckIn)
}

// Status dihitung dari waktu server, bukan dari waktu perangkat maupun metadata foto.
func TestAttendanceStatusUsesServerTime(t *testing.T) {
	late := time.Date(2026, time.August, 3, 9, 0, 1, 0, domain.Jakarta())
	store := &attendanceStoreStub{existing: map[string]bool{}}
	service := newAttendanceServiceForTest(store, &documentStoreStub{}, transactionStub{}, late)

	_, err := service.Record(
		context.Background(), employeeIdentity(),
		attendanceCommand(domain.WorkModeWFH, testOfficeLat, testOfficeLon, nil),
		RequestMeta{},
	)

	require.NoError(t, err)
	assert.Equal(t, domain.AttendanceStatusLate, store.created.Status)
	assert.Equal(t, "2026-08-03", store.created.Date)
	// waktu_local diisi waktu server dalam WIB, bukan dari client (D-022).
	assert.Equal(t, late.UTC(), store.created.NetworkTime)
}

// Kegagalan transaction setelah upload harus membersihkan foto agar tidak menjadi orphan.
func TestAttendanceRemovesOrphanPhotoWhenTransactionFails(t *testing.T) {
	store := &attendanceStoreStub{existing: map[string]bool{}}
	photos := &documentStoreStub{}
	service := newAttendanceServiceForTest(
		store, photos, failingTransactionStub{err: errors.New("database unavailable")}, workingMonday(),
	)

	_, err := service.Record(
		context.Background(), employeeIdentity(),
		attendanceCommand(domain.WorkModeWFH, testOfficeLat, testOfficeLon, nil),
		RequestMeta{},
	)

	require.Error(t, err)
	assert.NotEmpty(t, photos.uploadedPath)
	assert.Equal(t, photos.uploadedPath, photos.deletedPath)
}

func TestAttendanceForbidsTopManagement(t *testing.T) {
	photos := &documentStoreStub{}
	service := newAttendanceServiceForTest(
		&attendanceStoreStub{existing: map[string]bool{}}, photos, transactionStub{}, workingMonday(),
	)

	_, err := service.Record(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleTopManagement},
		attendanceCommand(domain.WorkModeWFH, testOfficeLat, testOfficeLon, nil),
		RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrForbidden)
	assert.Empty(t, photos.uploadedPath, "role tertolak tidak boleh menyentuh storage")
}

func TestLiveFeedRestrictsToMonitoringRoles(t *testing.T) {
	service := newAttendanceServiceForTest(
		&attendanceStoreStub{}, &documentStoreStub{}, transactionStub{}, workingMonday(),
	)

	for _, role := range []domain.RoleName{domain.RoleEmployee, domain.RoleSupervisor, domain.RoleTopManagement} {
		_, err := service.LiveFeed(context.Background(), domain.Identity{Role: role}, "")
		require.ErrorIsf(t, err, domain.ErrForbidden, "role %s harus ditolak", role)
	}
	for _, role := range []domain.RoleName{domain.RoleHR} {
		_, err := service.LiveFeed(context.Background(), domain.Identity{Role: role}, "")
		require.NoErrorf(t, err, "role %s harus dapat membaca live feed", role)
	}
}

// Rentang laporan mengikuti keputusan D-026.
func TestAttendanceReportResolvesPeriodRange(t *testing.T) {
	cases := []struct {
		name          string
		query         ReportQuery
		expectedStart string
		expectedEnd   string
		expectError   bool
	}{
		{"default bulan berjalan", ReportQuery{}, "2026-08-01", "2026-08-31", false},
		{"periode bulan tertentu", ReportQuery{Period: "2026-07"}, "2026-07-01", "2026-07-31", false},
		{
			"rentang custom",
			ReportQuery{Start: "2026-08-03", End: "2026-08-07"},
			"2026-08-03", "2026-08-07", false,
		},
		{"rentang setengah", ReportQuery{Start: "2026-08-03"}, "", "", true},
		{"rentang terbalik", ReportQuery{Start: "2026-08-07", End: "2026-08-03"}, "", "", true},
		{"periode tidak valid", ReportQuery{Period: "Agustus"}, "", "", true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			store := &attendanceStoreStub{}
			service := newAttendanceServiceForTest(
				store, &documentStoreStub{}, transactionStub{}, workingMonday(),
			)

			_, err := service.Report(
				context.Background(), domain.Identity{Role: domain.RoleHR}, testCase.query,
			)

			if testCase.expectError {
				require.ErrorIs(t, err, domain.ErrInvalidRequest)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedStart, store.reportFilter.Start.Format(domain.DateLayout))
			assert.Equal(t, testCase.expectedEnd, store.reportFilter.End.Format(domain.DateLayout))
		})
	}
}

// Alpha dihitung dari hari kerja Senin-Jumat dalam rentang laporan.
func TestAttendanceReportPassesWorkingDayCount(t *testing.T) {
	store := &attendanceStoreStub{}
	service := newAttendanceServiceForTest(
		store, &documentStoreStub{}, transactionStub{}, workingMonday(),
	)

	_, err := service.Report(context.Background(), domain.Identity{Role: domain.RoleHR}, ReportQuery{
		Start: "2026-08-03", End: "2026-08-16",
	})

	require.NoError(t, err)
	// Dua minggu penuh berisi sepuluh hari kerja.
	assert.Equal(t, 10, store.reportWorkday)
}

func TestAttendanceExportRestrictsToHR(t *testing.T) {
	store := &attendanceStoreStub{report: domain.AttendanceReportPage{
		Items: []domain.AttendanceReportItem{{EmployeeName: "Karyawan Uji", Present: 1}},
	}}
	service := newAttendanceServiceForTest(
		store, &documentStoreStub{}, transactionStub{}, workingMonday(),
	)

	for _, role := range []domain.RoleName{
		domain.RoleEmployee, domain.RoleSupervisor, domain.RoleTopManagement,
	} {
		_, err := service.ExportReport(
			context.Background(), domain.Identity{Role: role},
			ReportQuery{}, domain.ExportFormatXLSX, RequestMeta{},
		)
		require.ErrorIsf(t, err, domain.ErrForbidden, "role %s harus ditolak", role)
	}

	file, err := service.ExportReport(
		context.Background(), domain.Identity{UserID: uuid.New(), Role: domain.RoleHR},
		ReportQuery{}, domain.ExportFormatXLSX, RequestMeta{},
	)
	require.NoError(t, err)
	assert.NotEmpty(t, file.Content)
	assert.Contains(t, file.FileName, "laporan-kehadiran")
}

func TestAttendanceExportReturnsNotFoundForEmptyDataset(t *testing.T) {
	service := newAttendanceServiceForTest(
		&attendanceStoreStub{}, &documentStoreStub{}, transactionStub{}, workingMonday(),
	)

	_, err := service.ExportReport(
		context.Background(), domain.Identity{UserID: uuid.New(), Role: domain.RoleHR},
		ReportQuery{}, domain.ExportFormatXLSX, RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrNotFound)
}
