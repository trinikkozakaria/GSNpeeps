package domain

import (
	"encoding/json"
	"errors"
	"time"
)

// DateLayout adalah format tanggal kontrak (JSON Schema `format: date`).
const DateLayout = "2006-01-02"

// PeriodLayout adalah format periode YYYY-MM pada schema YearMonth.
const PeriodLayout = "2006-01"

// JakartaTimezone adalah nama timezone yang dikirim pada DashboardPeriodRange.
const JakartaTimezone = "Asia/Jakarta"

// ErrInvalidPeriod dikembalikan ketika query periode dashboard tidak dikenali.
var ErrInvalidPeriod = errors.New("invalid dashboard period")

// jakarta memakai offset tetap +07:00 karena WIB tidak mengenal daylight saving.
// Offset tetap membuat perhitungan boundary deterministik dan tidak bergantung pada
// ketersediaan database tzdata di dalam container runtime.
var jakarta = time.FixedZone("WIB", 7*60*60)

// Jakarta mengembalikan lokasi waktu kerja resmi sistem.
func Jakarta() *time.Location { return jakarta }

// DashboardPeriodType adalah granularitas kalender dashboard sesuai D-015.
type DashboardPeriodType string

const (
	PeriodDaily   DashboardPeriodType = "harian"
	PeriodWeekly  DashboardPeriodType = "mingguan"
	PeriodMonthly DashboardPeriodType = "bulanan"
	PeriodYearly  DashboardPeriodType = "tahunan"
)

// DashboardRange adalah rentang tanggal inklusif dalam timezone Asia/Jakarta.
type DashboardRange struct {
	Type  DashboardPeriodType `json:"tipe"`
	Start time.Time           `json:"-"`
	End   time.Time           `json:"-"`
}

// MarshalJSON menghasilkan schema DashboardPeriodRange.
func (r DashboardRange) MarshalJSON() ([]byte, error) {
	type payload struct {
		Type     DashboardPeriodType `json:"tipe"`
		Start    string              `json:"tanggal_mulai"`
		End      string              `json:"tanggal_selesai"`
		Timezone string              `json:"timezone"`
	}
	return json.Marshal(payload{
		Type:     r.Type,
		Start:    r.Start.Format(DateLayout),
		End:      r.End.Format(DateLayout),
		Timezone: JakartaTimezone,
	})
}

// ExclusiveEnd mengembalikan batas atas eksklusif untuk perbandingan timestamp.
func (r DashboardRange) ExclusiveEnd() time.Time { return r.End.AddDate(0, 0, 1) }

// WeekdayCount menghitung hari Senin-Jumat di dalam rentang.
func (r DashboardRange) WeekdayCount() int {
	count := 0
	for day := r.Start; !day.After(r.End); day = day.AddDate(0, 0, 1) {
		if day.Weekday() != time.Saturday && day.Weekday() != time.Sunday {
			count++
		}
	}
	return count
}

// ResolveDashboardRange menghitung boundary periode sesuai keputusan D-015. Minggu
// berjalan Senin-Minggu, bulan memakai bulan kalender, dan tahun memakai 1 Januari
// sampai 31 Desember dalam timezone Asia/Jakarta.
func ResolveDashboardRange(periodType DashboardPeriodType, anchor time.Time) (DashboardRange, error) {
	anchor = anchor.In(jakarta)
	day := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, jakarta)
	switch periodType {
	case PeriodDaily:
		return DashboardRange{Type: periodType, Start: day, End: day}, nil
	case PeriodWeekly:
		offset := (int(day.Weekday()) + 6) % 7 // Senin = 0
		start := day.AddDate(0, 0, -offset)
		return DashboardRange{Type: periodType, Start: start, End: start.AddDate(0, 0, 6)}, nil
	case PeriodMonthly:
		start := time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, jakarta)
		return DashboardRange{Type: periodType, Start: start, End: start.AddDate(0, 1, -1)}, nil
	case PeriodYearly:
		start := time.Date(day.Year(), time.January, 1, 0, 0, 0, 0, jakarta)
		return DashboardRange{Type: periodType, Start: start, End: start.AddDate(1, 0, -1)}, nil
	default:
		return DashboardRange{}, ErrInvalidPeriod
	}
}

// CurrentSalaryPeriod mengembalikan periode gaji bulan berjalan dalam format YYYY-MM.
func CurrentSalaryPeriod(now time.Time) string {
	return now.In(jakarta).Format(PeriodLayout)
}
