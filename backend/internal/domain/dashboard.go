package domain

import "github.com/google/uuid"

// NamedCount memetakan schema NamedCount pada OpenAPI.
type NamedCount struct {
	Name  string `json:"nama"`
	Count int    `json:"jumlah"`
}

// GenderCategory adalah kategori rasio gender. Gender kosong masuk `belum_diisi` dan tidak
// pernah dihitung sebagai laki-laki atau perempuan (D-015).
type GenderCategory string

const (
	GenderMale       GenderCategory = "laki_laki"
	GenderFemale     GenderCategory = "perempuan"
	GenderUnassigned GenderCategory = "belum_diisi"
)

type GenderCount struct {
	Category GenderCategory `json:"kategori"`
	Count    int            `json:"jumlah"`
}

// OrganizationNode adalah simpul org chart; root adalah karyawan tanpa atasan aktif.
type OrganizationNode struct {
	EmployeeID   uuid.UUID          `json:"employee_id"`
	Name         string             `json:"nama"`
	Department   string             `json:"departemen"`
	Position     string             `json:"jabatan"`
	Subordinates []OrganizationNode `json:"bawahan"`
}

// OrganizationMember adalah baris mentah untuk membangun org chart.
type OrganizationMember struct {
	EmployeeID   uuid.UUID
	SupervisorID *uuid.UUID
	Name         string
	Department   string
	Position     string
}

// DashboardSnapshot adalah agregasi employee-side yang dibaca dalam satu kali round trip
// per bagian; nilai yang berasal dari modul Attendance/Ketidakhadiran tidak termasuk.
type DashboardSnapshot struct {
	HeadcountStart      int
	ActiveEmployees     int
	InactiveEmployees   int
	NewEmployees        int
	Resigned            int
	ActiveDepartments   []NamedCount
	InactiveDepartments []NamedCount
	GenderRatio         []GenderCount
	OrganizationMembers []OrganizationMember
}

// AttendanceMetrics berasal dari modul Attendance. Sampai epic tersebut selesai nilainya
// nol dan tidak boleh diisi dengan data buatan (D-020).
type AttendanceMetrics struct {
	ValidAttendance int
	Late            int
}

// LeaveMetrics berasal dari modul Ketidakhadiran dan Lembur (D-020).
type LeaveMetrics struct {
	ApprovedLeaveDays int
	PendingRequests   int
}

// DashboardMetrics memetakan schema DashboardMetrics pada OpenAPI.
type DashboardMetrics struct {
	Period                        DashboardRange     `json:"periode"`
	TotalEmployees                int                `json:"total_karyawan"`
	ActiveEmployees               int                `json:"karyawan_aktif"`
	InactiveEmployees             int                `json:"karyawan_nonaktif"`
	NewEmployees                  int                `json:"karyawan_baru"`
	Resigned                      int                `json:"resign"`
	TurnoverRate                  float64            `json:"turnover_rate"`
	ValidAttendance               int                `json:"hadir_valid"`
	Late                          int                `json:"terlambat"`
	ApprovedLeaveDays             int                `json:"hari_izin_disetujui"`
	EstimatedPayrollCost          float64            `json:"estimasi_biaya_payroll"`
	PendingRequests               int                `json:"pengajuan_menunggu"`
	ActiveDepartmentComposition   []NamedCount       `json:"komposisi_departemen_aktif"`
	InactiveDepartmentComposition []NamedCount       `json:"komposisi_departemen_nonaktif"`
	GenderRatio                   []GenderCount      `json:"rasio_gender"`
	OrganizationChart             []OrganizationNode `json:"organization_chart"`
}

// ClockHistory memetakan schema ClockHistory pada OpenAPI.
type ClockHistory struct {
	Date     string  `json:"tanggal"`
	CheckIn  *string `json:"check_in"`
	CheckOut *string `json:"check_out"`
	Status   string  `json:"status"`
}

// PersonalMetrics memetakan schema PersonalMetrics pada OpenAPI.
type PersonalMetrics struct {
	Period        string         `json:"periode"`
	Present       int            `json:"hadir"`
	Late          int            `json:"terlambat"`
	Leave         int            `json:"izin"`
	OvertimeHours float64        `json:"total_lembur_jam"`
	History       []ClockHistory `json:"riwayat_absensi"`
}

// PersonalAttendanceMetrics berasal dari modul Attendance/Ketidakhadiran/Lembur (D-020).
type PersonalAttendanceMetrics struct {
	Present       int
	Late          int
	Leave         int
	OvertimeHours float64
	History       []ClockHistory
}
