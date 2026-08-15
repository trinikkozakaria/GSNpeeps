package domain

import (
	"math"
	"time"

	"github.com/google/uuid"
)

type AttendanceType string

const (
	AttendanceCheckIn  AttendanceType = "check_in"
	AttendanceCheckOut AttendanceType = "check_out"
)

func (t AttendanceType) Valid() bool {
	return t == AttendanceCheckIn || t == AttendanceCheckOut
}

type WorkMode string

const (
	WorkModeWFO WorkMode = "WFO"
	WorkModeWFH WorkMode = "WFH"
	WorkModeWFA WorkMode = "WFA"
)

func (m WorkMode) Valid() bool {
	return m == WorkModeWFO || m == WorkModeWFH || m == WorkModeWFA
}

// Status absensi sesuai enum schema Attendance pada OpenAPI (D-022).
const (
	AttendanceStatusOnTime     = "tepat_waktu"
	AttendanceStatusLate       = "terlambat"
	AttendanceStatusEarlyLeave = "pulang_cepat"
	AttendanceStatusValid      = "valid"
)

// OfficeRadiusMeters adalah batas WFO yang ditetapkan PRD.
const OfficeRadiusMeters = 100.0

// workStartHour dan workEndHour adalah jam kerja reguler 09:00-18:00 WIB. Check-in tepat
// pukul 09:00:00 belum terlambat; checkout sebelum 18:00:00 dicatat pulang_cepat.
const (
	workStartHour = 9
	workEndHour   = 18
)

type OfficeLocation struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"kode"`
	Name      string    `json:"nama"`
	Address   *string   `json:"alamat"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	IsActive  bool      `json:"is_active"`
}

// Attendance memetakan schema Attendance pada OpenAPI.
type Attendance struct {
	ID               uuid.UUID      `json:"id"`
	EmployeeID       uuid.UUID      `json:"employee_id"`
	Date             string         `json:"tanggal"`
	Type             AttendanceType `json:"tipe"`
	WorkMode         WorkMode       `json:"mode_kerja"`
	Time             time.Time      `json:"waktu"`
	Latitude         float64        `json:"gps_lat"`
	Longitude        float64        `json:"gps_long"`
	OfficeLocationID *uuid.UUID     `json:"office_location_id"`
	DistanceMeters   *float64       `json:"distance_meters"`
	PhotoURL         *string        `json:"foto_url"`
	Status           string         `json:"status"`
}

// AttendanceLiveFeedItem menambahkan identitas karyawan pada live feed.
type AttendanceLiveFeedItem struct {
	Attendance
	EmployeeName string `json:"nama_karyawan"`
	Department   string `json:"departemen"`
}

// RecordAttendance adalah perintah pencatatan absensi yang sudah tervalidasi handler.
type RecordAttendance struct {
	UserID           uuid.UUID
	EmployeeID       uuid.UUID
	Type             AttendanceType
	WorkMode         WorkMode
	Latitude         float64
	Longitude        float64
	OfficeLocationID *uuid.UUID
	PhotoExtension   string
	PhotoMediaType   string
	PhotoContent     []byte
}

// AttendanceRow adalah baris yang siap disimpan repository.
type AttendanceRow struct {
	UserID           uuid.UUID
	Date             string
	Type             AttendanceType
	WorkMode         WorkMode
	NetworkTime      time.Time
	LocalTime        time.Time
	Latitude         float64
	Longitude        float64
	OfficeLocationID *uuid.UUID
	DistanceMeters   *float64
	PhotoURL         string
	Status           string
}

// AttendanceReportItem memetakan schema AttendanceReportItem.
type AttendanceReportItem struct {
	EmployeeID   uuid.UUID `json:"employee_id"`
	EmployeeName string    `json:"nama_karyawan"`
	Department   string    `json:"departemen"`
	Present      int       `json:"hadir"`
	Late         int       `json:"terlambat"`
	Leave        int       `json:"izin"`
	Absent       int       `json:"alpha"`
	TotalHours   float64   `json:"total_jam_kerja"`
}

type AttendanceReportPage struct {
	Items []AttendanceReportItem
	Total int
	Page  int
	Limit int
}

// AttendanceReportFilter memakai rentang tanggal yang sudah diselesaikan service (D-026).
type AttendanceReportFilter struct {
	Start        time.Time
	End          time.Time
	DepartmentID *uuid.UUID
	Page         int
	Limit        int
}

// ExpiredPhoto adalah foto absensi yang melewati masa retensi tiga bulan.
type ExpiredPhoto struct {
	AttendanceID uuid.UUID
	PhotoURL     string
}

// PhotoRetention adalah masa simpan foto absensi menurut PRD.
const PhotoRetention = 3 * 30 * 24 * time.Hour

// CheckInStatus mengembalikan `terlambat` hanya bila waktu server melewati 09:00:00 WIB.
// Tepat pukul 09:00:00 belum terlambat.
func CheckInStatus(moment time.Time) string {
	local := moment.In(Jakarta())
	threshold := time.Date(
		local.Year(), local.Month(), local.Day(), workStartHour, 0, 0, 0, Jakarta(),
	)
	if local.After(threshold) {
		return AttendanceStatusLate
	}
	return AttendanceStatusOnTime
}

// CheckOutStatus mencatat `pulang_cepat` bila checkout terjadi sebelum 18:00:00 WIB.
// Checkout lebih awal tetap valid dan tidak ditolak.
func CheckOutStatus(moment time.Time) string {
	local := moment.In(Jakarta())
	threshold := time.Date(
		local.Year(), local.Month(), local.Day(), workEndHour, 0, 0, 0, Jakarta(),
	)
	if local.Before(threshold) {
		return AttendanceStatusEarlyLeave
	}
	return AttendanceStatusValid
}

// earthRadiusMeters adalah radius rata-rata bumi menurut IUGG.
const earthRadiusMeters = 6371008.8

// DistanceMeters menghitung jarak great-circle memakai formula haversine dan mengembalikan
// hasil dalam satuan meter. Presisi haversine cukup untuk radius kantor 100 meter: pada
// jarak sekecil ini simpangan terhadap model ellipsoid berada jauh di bawah satu meter.
func DistanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	deltaPhi := (lat2 - lat1) * math.Pi / 180
	deltaLambda := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMeters * c
}
