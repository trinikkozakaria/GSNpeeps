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

type dashboardReaderStub struct {
	receivedRange   domain.DashboardRange
	receivedPeriods []string
	snapshot        domain.DashboardSnapshot
	totals          map[string]float64
}

func (s *dashboardReaderStub) Snapshot(
	_ context.Context,
	period domain.DashboardRange,
) (domain.DashboardSnapshot, error) {
	s.receivedRange = period
	return s.snapshot, nil
}

func (s *dashboardReaderStub) SalaryTotals(
	_ context.Context,
	periods []string,
) (map[string]float64, error) {
	s.receivedPeriods = periods
	return s.totals, nil
}

func newDashboardServiceForTest(reader DashboardReader) *DashboardService {
	pending := repository.NewPendingMetricsRepository()
	service := NewDashboardService(reader, pending, pending)
	service.now = func() time.Time {
		return time.Date(2026, time.August, 12, 15, 0, 0, 0, domain.Jakarta())
	}
	return service
}

func TestDashboardRestrictsToHRAndTopManagement(t *testing.T) {
	for _, role := range []domain.RoleName{domain.RoleEmployee, domain.RoleSupervisor} {
		service := newDashboardServiceForTest(&dashboardReaderStub{})

		_, err := service.Metrics(context.Background(), domain.Identity{Role: role}, "", "")

		require.ErrorIsf(t, err, domain.ErrForbidden, "role %s harus ditolak", role)
	}

	for _, role := range []domain.RoleName{domain.RoleHR, domain.RoleTopManagement} {
		service := newDashboardServiceForTest(&dashboardReaderStub{})

		_, err := service.Metrics(context.Background(), domain.Identity{Role: role}, "", "")

		require.NoErrorf(t, err, "role %s harus dapat membaca dashboard", role)
	}
}

func TestDashboardPeriodBoundariesUseJakartaCalendar(t *testing.T) {
	cases := []struct {
		name     string
		period   domain.DashboardPeriodType
		anchor   string
		expected [2]string
	}{
		{"harian", domain.PeriodDaily, "2026-08-12", [2]string{"2026-08-12", "2026-08-12"}},
		// 12 Agustus 2026 jatuh pada hari Rabu; minggu berjalan Senin-Minggu.
		{"mingguan", domain.PeriodWeekly, "2026-08-12", [2]string{"2026-08-10", "2026-08-16"}},
		{"mingguan senin", domain.PeriodWeekly, "2026-08-10", [2]string{"2026-08-10", "2026-08-16"}},
		{"mingguan minggu", domain.PeriodWeekly, "2026-08-16", [2]string{"2026-08-10", "2026-08-16"}},
		{"bulanan", domain.PeriodMonthly, "2026-08-12", [2]string{"2026-08-01", "2026-08-31"}},
		{"bulanan februari kabisat", domain.PeriodMonthly, "2028-02-05", [2]string{"2028-02-01", "2028-02-29"}},
		{"tahunan", domain.PeriodYearly, "2026-08-12", [2]string{"2026-01-01", "2026-12-31"}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stub := &dashboardReaderStub{}
			service := newDashboardServiceForTest(stub)

			metrics, err := service.Metrics(
				context.Background(),
				domain.Identity{Role: domain.RoleHR},
				testCase.period,
				testCase.anchor,
			)

			require.NoError(t, err)
			assert.Equal(t, testCase.expected[0], stub.receivedRange.Start.Format(domain.DateLayout))
			assert.Equal(t, testCase.expected[1], stub.receivedRange.End.Format(domain.DateLayout))
			assert.Equal(t, domain.JakartaTimezone, metricsTimezone(t, metrics))
		})
	}
}

func TestDashboardDefaultsToMonthlyPeriod(t *testing.T) {
	stub := &dashboardReaderStub{}
	service := newDashboardServiceForTest(stub)

	metrics, err := service.Metrics(context.Background(), domain.Identity{Role: domain.RoleHR}, "", "")

	require.NoError(t, err)
	assert.Equal(t, domain.PeriodMonthly, metrics.Period.Type)
	assert.Equal(t, "2026-08-01", stub.receivedRange.Start.Format(domain.DateLayout))
	assert.Equal(t, "2026-08-31", stub.receivedRange.End.Format(domain.DateLayout))
}

func TestDashboardRejectsInvalidPeriodAndAnchor(t *testing.T) {
	service := newDashboardServiceForTest(&dashboardReaderStub{})

	_, err := service.Metrics(
		context.Background(),
		domain.Identity{Role: domain.RoleHR},
		domain.DashboardPeriodType("triwulan"),
		"",
	)
	require.ErrorIs(t, err, domain.ErrInvalidRequest)

	_, err = service.Metrics(
		context.Background(),
		domain.Identity{Role: domain.RoleHR},
		domain.PeriodMonthly,
		"12-08-2026",
	)
	require.ErrorIs(t, err, domain.ErrInvalidRequest)
}

func TestDashboardTurnoverRateFollowsDecisionFormula(t *testing.T) {
	stub := &dashboardReaderStub{snapshot: domain.DashboardSnapshot{
		HeadcountStart:  100,
		ActiveEmployees: 96,
		Resigned:        4,
	}}
	service := newDashboardServiceForTest(stub)

	metrics, err := service.Metrics(context.Background(), domain.Identity{Role: domain.RoleHR}, "", "")

	require.NoError(t, err)
	// 4 / ((100 + 96) / 2) * 100
	assert.InDelta(t, 4.0816, metrics.TurnoverRate, 0.0001)
}

func TestDashboardTurnoverRateIsZeroWhenDenominatorIsZero(t *testing.T) {
	stub := &dashboardReaderStub{snapshot: domain.DashboardSnapshot{Resigned: 3}}
	service := newDashboardServiceForTest(stub)

	metrics, err := service.Metrics(context.Background(), domain.Identity{Role: domain.RoleHR}, "", "")

	require.NoError(t, err)
	assert.Zero(t, metrics.TurnoverRate)
}

// Payroll bulanan dialokasikan proporsional terhadap hari Senin-Jumat yang beririsan
// dengan periode (D-015).
func TestDashboardPayrollAllocatesByWeekdayOverlap(t *testing.T) {
	stub := &dashboardReaderStub{totals: map[string]float64{"2026-08": 21000000}}
	service := newDashboardServiceForTest(stub)

	metrics, err := service.Metrics(
		context.Background(),
		domain.Identity{Role: domain.RoleHR},
		domain.PeriodWeekly,
		"2026-08-12",
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"2026-08"}, stub.receivedPeriods)
	// Agustus 2026 memiliki 21 hari kerja; minggu 10-16 Agustus menyumbang 5 hari kerja.
	assert.InDelta(t, 21000000*5.0/21.0, metrics.EstimatedPayrollCost, 0.01)
}

func TestDashboardPayrollSpansEveryMonthInRange(t *testing.T) {
	stub := &dashboardReaderStub{totals: map[string]float64{}}
	service := newDashboardServiceForTest(stub)

	_, err := service.Metrics(
		context.Background(),
		domain.Identity{Role: domain.RoleHR},
		domain.PeriodYearly,
		"2026-08-12",
	)

	require.NoError(t, err)
	assert.Len(t, stub.receivedPeriods, 12)
	assert.Equal(t, "2026-01", stub.receivedPeriods[0])
	assert.Equal(t, "2026-12", stub.receivedPeriods[11])
}

// Metrik yang bersumber dari modul Attendance/Ketidakhadiran tetap nol sampai epic tersebut
// selesai; tidak ada nilai yang dikarang (D-020).
func TestDashboardAttendanceAndLeaveMetricsAreZeroWhilePending(t *testing.T) {
	service := newDashboardServiceForTest(&dashboardReaderStub{})

	metrics, err := service.Metrics(context.Background(), domain.Identity{Role: domain.RoleHR}, "", "")

	require.NoError(t, err)
	assert.Zero(t, metrics.ValidAttendance)
	assert.Zero(t, metrics.Late)
	assert.Zero(t, metrics.ApprovedLeaveDays)
	assert.Zero(t, metrics.PendingRequests)
}

func TestDashboardKeepsGenderAndDepartmentGroupsSeparate(t *testing.T) {
	stub := &dashboardReaderStub{snapshot: domain.DashboardSnapshot{
		ActiveEmployees:     3,
		InactiveEmployees:   1,
		ActiveDepartments:   []domain.NamedCount{{Name: "Teknologi", Count: 3}},
		InactiveDepartments: []domain.NamedCount{{Name: "Teknologi", Count: 1}},
		GenderRatio: []domain.GenderCount{
			{Category: domain.GenderMale, Count: 2},
			{Category: domain.GenderFemale, Count: 1},
			{Category: domain.GenderUnassigned, Count: 0},
		},
	}}
	service := newDashboardServiceForTest(stub)

	metrics, err := service.Metrics(context.Background(), domain.Identity{Role: domain.RoleHR}, "", "")

	require.NoError(t, err)
	assert.Equal(t, 4, metrics.TotalEmployees)
	assert.Equal(t, 3, metrics.ActiveDepartmentComposition[0].Count)
	assert.Equal(t, 1, metrics.InactiveDepartmentComposition[0].Count)
	assert.Len(t, metrics.GenderRatio, 3)
	assert.Equal(t, domain.GenderUnassigned, metrics.GenderRatio[2].Category)
}

func TestBuildOrganizationChartNestsDirectReports(t *testing.T) {
	director := uuid.New()
	manager := uuid.New()
	staff := uuid.New()
	orphanSupervisor := uuid.New()
	orphan := uuid.New()

	chart := buildOrganizationChart([]domain.OrganizationMember{
		{EmployeeID: director, Name: "Direktur", Department: "Eksekutif", Position: "Direktur"},
		{EmployeeID: manager, SupervisorID: &director, Name: "Manajer"},
		{EmployeeID: staff, SupervisorID: &manager, Name: "Staff"},
		{EmployeeID: orphan, SupervisorID: &orphanSupervisor, Name: "Tanpa Atasan Aktif"},
	})

	require.Len(t, chart, 2)
	assert.Equal(t, director, chart[0].EmployeeID)
	require.Len(t, chart[0].Subordinates, 1)
	assert.Equal(t, manager, chart[0].Subordinates[0].EmployeeID)
	require.Len(t, chart[0].Subordinates[0].Subordinates, 1)
	assert.Equal(t, staff, chart[0].Subordinates[0].Subordinates[0].EmployeeID)
	// Karyawan yang atasannya tidak termasuk populasi aktif tetap muncul sebagai root.
	assert.Equal(t, orphan, chart[1].EmployeeID)
}

func metricsTimezone(t *testing.T, metrics domain.DashboardMetrics) string {
	t.Helper()
	encoded, err := metrics.Period.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"timezone":"`+domain.JakartaTimezone+`"`)
	return domain.JakartaTimezone
}
