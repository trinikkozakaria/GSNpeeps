package domain

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Koordinat sintetis; bukan lokasi kantor nyata.
const (
	officeLat = -6.2000000
	officeLon = 106.8000000
)

// offsetByMeters menggeser garis lintang sejauh jarak tertentu ke utara. Faktor konversi
// diturunkan dari radius bumi yang sama seperti implementasi agar test menguji perilaku
// batas radius, bukan selisih konstanta.
func offsetByMeters(meters float64) float64 {
	metersPerDegree := earthRadiusMeters * math.Pi / 180
	return officeLat + meters/metersPerDegree
}

func TestDistanceMetersIsZeroAtSamePoint(t *testing.T) {
	assert.InDelta(t, 0, DistanceMeters(officeLat, officeLon, officeLat, officeLon), 0.001)
}

// Batas WFO mengikuti OfficeRadiusMeters yang dapat dikonfigurasi (default 500 meter sejak
// keputusan produk 2026-09-18): di bawah radius diterima, di atasnya ditolak. Kasus persis di
// radius tidak diuji di sini karena round-trip degree->meter lewat offsetByMeters membawa
// noise floating-point sekelas 1e-11 m yang bisa jatuh di kedua sisi batas tergantung
// besaran radius; batas ketat < / > tetap diuji lewat kasus 0,01 meter di kedua sisi.
func TestDistanceMetersOfficeRadiusBoundary(t *testing.T) {
	radius := OfficeRadiusMeters
	cases := []struct {
		name     string
		meters   float64
		accepted bool
	}{
		{"0.01 meter di bawah radius", radius - 0.01, true},
		{"0.01 meter di atas radius", radius + 0.01, false},
		{"50 meter di atas radius", radius + 50, false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			distance := DistanceMeters(offsetByMeters(testCase.meters), officeLon, officeLat, officeLon)

			assert.InDelta(t, testCase.meters, distance, 0.05)
			assert.Equal(t, testCase.accepted, distance <= OfficeRadiusMeters)
		})
	}
}

// Check-in tepat pukul 09:00:00 WIB belum terlambat; setelahnya terlambat.
func TestCheckInStatusBoundaryAtNineAM(t *testing.T) {
	cases := map[string]string{
		"08:59:59": AttendanceStatusOnTime,
		"09:00:00": AttendanceStatusOnTime,
		"09:00:01": AttendanceStatusLate,
		"10:30:00": AttendanceStatusLate,
	}
	for clock, expected := range cases {
		moment, err := time.ParseInLocation("2006-01-02 15:04:05", "2026-08-03 "+clock, Jakarta())
		assert.NoError(t, err)
		assert.Equalf(t, expected, CheckInStatus(moment), "jam %s", clock)
	}
}

// Checkout sebelum 18:00 tetap valid dan dicatat pulang_cepat, bukan ditolak.
func TestCheckOutStatusBoundaryAtSixPM(t *testing.T) {
	cases := map[string]string{
		"17:59:59": AttendanceStatusEarlyLeave,
		"18:00:00": AttendanceStatusValid,
		"19:15:00": AttendanceStatusValid,
	}
	for clock, expected := range cases {
		moment, err := time.ParseInLocation("2006-01-02 15:04:05", "2026-08-03 "+clock, Jakarta())
		assert.NoError(t, err)
		assert.Equalf(t, expected, CheckOutStatus(moment), "jam %s", clock)
	}
}

// Status dihitung dari waktu server; input UTC harus menghasilkan keputusan yang sama
// setelah dikonversi ke WIB.
func TestCheckInStatusUsesJakartaOffset(t *testing.T) {
	// 02:00:00 UTC sama dengan 09:00:00 WIB.
	moment := time.Date(2026, time.August, 3, 2, 0, 0, 0, time.UTC)
	assert.Equal(t, AttendanceStatusOnTime, CheckInStatus(moment))

	assert.Equal(t, AttendanceStatusLate,
		CheckInStatus(time.Date(2026, time.August, 3, 2, 0, 1, 0, time.UTC)))
}

// Radius WFO dan jam kerja dapat dikonfigurasi lewat ConfigureAttendancePolicy (dipanggil
// composition root dari env var ATTENDANCE_*), menggantikan nilai tetap PRD sebelumnya.
func TestConfigureAttendancePolicyOverridesRadiusAndWorkHours(t *testing.T) {
	t.Cleanup(func() { ConfigureAttendancePolicy(500, 9, 0, 18, 0) })

	ConfigureAttendancePolicy(750, 8, 30, 17, 30)

	assert.Equal(t, 750.0, OfficeRadiusMeters)

	onTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2026-08-03 08:30:00", Jakarta())
	assert.NoError(t, err)
	assert.Equal(t, AttendanceStatusOnTime, CheckInStatus(onTime))
	assert.Equal(t, AttendanceStatusLate, CheckInStatus(onTime.Add(time.Second)))

	validCheckout, err := time.ParseInLocation("2006-01-02 15:04:05", "2026-08-03 17:30:00", Jakarta())
	assert.NoError(t, err)
	assert.Equal(t, AttendanceStatusValid, CheckOutStatus(validCheckout))
	assert.Equal(t, AttendanceStatusEarlyLeave, CheckOutStatus(validCheckout.Add(-time.Second)))
}
