package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

// DashboardReader membaca agregasi employee-side untuk dashboard.
type DashboardReader interface {
	Snapshot(context.Context, domain.DashboardRange) (domain.DashboardSnapshot, error)
	SalaryTotals(context.Context, []string) (map[string]float64, error)
}

// AttendanceMetricsReader dan LeaveMetricsReader menjembatani metrik yang berasal dari modul
// Attendance dan Ketidakhadiran/Lembur. Sampai epic tersebut selesai, adapter sementara
// mengembalikan nol tanpa membuat data palsu (keputusan D-020).
type AttendanceMetricsReader interface {
	AttendanceMetrics(context.Context, domain.DashboardRange) (domain.AttendanceMetrics, error)
}

type LeaveMetricsReader interface {
	LeaveMetrics(context.Context, domain.DashboardRange) (domain.LeaveMetrics, error)
}

type DashboardService struct {
	dashboard  DashboardReader
	attendance AttendanceMetricsReader
	leave      LeaveMetricsReader
	now        func() time.Time
}

func NewDashboardService(
	dashboard DashboardReader,
	attendance AttendanceMetricsReader,
	leave LeaveMetricsReader,
) *DashboardService {
	return &DashboardService{
		dashboard:  dashboard,
		attendance: attendance,
		leave:      leave,
		now:        time.Now,
	}
}

// Metrics menghitung agregasi dashboard untuk satu periode kalender. Hanya HR yang dapat
// membaca Dashboard HR; role lain ditolak sesuai aturan produk terbaru.
func (s *DashboardService) Metrics(
	ctx context.Context,
	identity domain.Identity,
	periodType domain.DashboardPeriodType,
	anchor string,
) (domain.DashboardMetrics, error) {
	if identity.Role != domain.RoleHR {
		return domain.DashboardMetrics{}, domain.ErrForbidden
	}
	if periodType == "" {
		periodType = domain.PeriodMonthly
	}
	anchorDate := s.now().In(domain.Jakarta())
	if anchor != "" {
		parsed, err := time.ParseInLocation(domain.DateLayout, anchor, domain.Jakarta())
		if err != nil {
			return domain.DashboardMetrics{}, domain.ErrInvalidRequest
		}
		anchorDate = parsed
	}
	period, err := domain.ResolveDashboardRange(periodType, anchorDate)
	if err != nil {
		return domain.DashboardMetrics{}, domain.ErrInvalidRequest
	}

	snapshot, err := s.dashboard.Snapshot(ctx, period)
	if err != nil {
		return domain.DashboardMetrics{}, fmt.Errorf("read dashboard snapshot: %w", err)
	}
	payroll, err := s.payrollEstimate(ctx, period)
	if err != nil {
		return domain.DashboardMetrics{}, err
	}
	attendance, err := s.attendance.AttendanceMetrics(ctx, period)
	if err != nil {
		return domain.DashboardMetrics{}, fmt.Errorf("read attendance metrics: %w", err)
	}
	leave, err := s.leave.LeaveMetrics(ctx, period)
	if err != nil {
		return domain.DashboardMetrics{}, fmt.Errorf("read leave metrics: %w", err)
	}

	return domain.DashboardMetrics{
		Period:                        period,
		TotalEmployees:                snapshot.ActiveEmployees + snapshot.InactiveEmployees,
		ActiveEmployees:               snapshot.ActiveEmployees,
		InactiveEmployees:             snapshot.InactiveEmployees,
		NewEmployees:                  snapshot.NewEmployees,
		Resigned:                      snapshot.Resigned,
		TurnoverRate:                  turnoverRate(snapshot),
		ValidAttendance:               attendance.ValidAttendance,
		Late:                          attendance.Late,
		ApprovedLeaveDays:             leave.ApprovedLeaveDays,
		EstimatedPayrollCost:          payroll,
		PendingRequests:               leave.PendingRequests,
		ActiveDepartmentComposition:   defaultNamedCounts(snapshot.ActiveDepartments),
		InactiveDepartmentComposition: defaultNamedCounts(snapshot.InactiveDepartments),
		GenderRatio:                   snapshot.GenderRatio,
		OrganizationChart:             buildOrganizationChart(snapshot.OrganizationMembers),
	}, nil
}

// turnoverRate mengikuti D-015: resign dibagi rata-rata headcount awal dan akhir periode,
// dikali 100. Denominator nol menghasilkan nol, bukan pembagian tak hingga.
func turnoverRate(snapshot domain.DashboardSnapshot) float64 {
	average := float64(snapshot.HeadcountStart+snapshot.ActiveEmployees) / 2
	if average <= 0 {
		return 0
	}
	return float64(snapshot.Resigned) / average * 100
}

// payrollEstimate mengalokasikan take_home_pay bulanan secara proporsional menurut hari
// Senin-Jumat yang beririsan dengan periode, tanpa kalender libur (D-015 dan G-010).
func (s *DashboardService) payrollEstimate(
	ctx context.Context,
	period domain.DashboardRange,
) (float64, error) {
	months := monthsInRange(period)
	keys := make([]string, 0, len(months))
	for _, month := range months {
		keys = append(keys, month.period)
	}
	totals, err := s.dashboard.SalaryTotals(ctx, keys)
	if err != nil {
		return 0, fmt.Errorf("read payroll totals: %w", err)
	}
	estimate := 0.0
	for _, month := range months {
		total, ok := totals[month.period]
		if !ok || month.monthWeekdays == 0 {
			continue
		}
		estimate += total * float64(month.overlapWeekdays) / float64(month.monthWeekdays)
	}
	return estimate, nil
}

type monthAllocation struct {
	period          string
	monthWeekdays   int
	overlapWeekdays int
}

func monthsInRange(period domain.DashboardRange) []monthAllocation {
	allocations := make([]monthAllocation, 0, 12)
	cursor := time.Date(
		period.Start.Year(), period.Start.Month(), 1, 0, 0, 0, 0, domain.Jakarta(),
	)
	for !cursor.After(period.End) {
		monthRange := domain.DashboardRange{
			Type:  domain.PeriodMonthly,
			Start: cursor,
			End:   cursor.AddDate(0, 1, -1),
		}
		overlap := domain.DashboardRange{
			Type:  domain.PeriodMonthly,
			Start: laterDate(monthRange.Start, period.Start),
			End:   earlierDate(monthRange.End, period.End),
		}
		allocations = append(allocations, monthAllocation{
			period:          cursor.Format(domain.PeriodLayout),
			monthWeekdays:   monthRange.WeekdayCount(),
			overlapWeekdays: overlap.WeekdayCount(),
		})
		cursor = cursor.AddDate(0, 1, 0)
	}
	return allocations
}

func laterDate(left, right time.Time) time.Time {
	if left.After(right) {
		return left
	}
	return right
}

func earlierDate(left, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}

func defaultNamedCounts(items []domain.NamedCount) []domain.NamedCount {
	if items == nil {
		return make([]domain.NamedCount, 0)
	}
	return items
}

// buildOrganizationChart menyusun pohon dari relasi atasan_id. Karyawan yang atasannya tidak
// termasuk populasi aktif diperlakukan sebagai root agar tidak ada simpul yang hilang.
func buildOrganizationChart(members []domain.OrganizationMember) []domain.OrganizationNode {
	nodes := make(map[uuid.UUID]*domain.OrganizationNode, len(members))
	order := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		nodes[member.EmployeeID] = &domain.OrganizationNode{
			EmployeeID:   member.EmployeeID,
			Name:         member.Name,
			Department:   member.Department,
			Position:     member.Position,
			Subordinates: make([]domain.OrganizationNode, 0),
		}
		order = append(order, member.EmployeeID)
	}

	children := make(map[uuid.UUID][]uuid.UUID, len(members))
	roots := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		if member.SupervisorID != nil && *member.SupervisorID != member.EmployeeID {
			if _, ok := nodes[*member.SupervisorID]; ok {
				children[*member.SupervisorID] = append(children[*member.SupervisorID], member.EmployeeID)
				continue
			}
		}
		roots = append(roots, member.EmployeeID)
	}

	position := make(map[uuid.UUID]int, len(order))
	for index, id := range order {
		position[id] = index
	}
	sortByOrder := func(ids []uuid.UUID) {
		sort.SliceStable(ids, func(i, j int) bool { return position[ids[i]] < position[ids[j]] })
	}

	visited := make(map[uuid.UUID]bool, len(members))
	var build func(id uuid.UUID) domain.OrganizationNode
	build = func(id uuid.UUID) domain.OrganizationNode {
		node := *nodes[id]
		if visited[id] {
			return node
		}
		visited[id] = true
		descendants := children[id]
		sortByOrder(descendants)
		for _, child := range descendants {
			node.Subordinates = append(node.Subordinates, build(child))
		}
		return node
	}

	sortByOrder(roots)
	chart := make([]domain.OrganizationNode, 0, len(roots))
	for _, root := range roots {
		chart = append(chart, build(root))
	}
	return chart
}
