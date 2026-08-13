package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/platform/export"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
)

// AttendanceStore adalah kebutuhan penyimpanan absensi dari sisi service.
type AttendanceStore interface {
	FindActiveOfficeLocation(context.Context, uuid.UUID) (domain.OfficeLocation, error)
	ListActiveOfficeLocations(context.Context) ([]domain.OfficeLocation, error)
	ExistsForDate(context.Context, uuid.UUID, string, domain.AttendanceType) (bool, error)
	Create(context.Context, domain.AttendanceRow) (domain.Attendance, error)
	LiveFeed(context.Context, string) ([]domain.AttendanceLiveFeedItem, error)
	Report(context.Context, domain.AttendanceReportFilter, int) (domain.AttendanceReportPage, error)
}

type AttendanceService struct {
	attendances AttendanceStore
	tx          EmployeeTransactionManager
	audit       AuditWriter
	photos      DocumentStore
	now         func() time.Time
}

func NewAttendanceService(
	attendances AttendanceStore,
	tx EmployeeTransactionManager,
	audit AuditWriter,
	photos DocumentStore,
) *AttendanceService {
	return &AttendanceService{
		attendances: attendances,
		tx:          tx,
		audit:       audit,
		photos:      photos,
		now:         time.Now,
	}
}

// ListOfficeLocations mengembalikan master lokasi kantor aktif untuk dropdown WFO. Kontrak
// membuka operasi ini bagi seluruh role terautentikasi karena karyawan bebas memilih kantor
// tempat ia hadir; tidak ada assignment kantor permanen (D-016).
//
// Response dapat kosong sampai lokasi resmi tersedia; tidak ada lokasi contoh yang dibuat.
func (s *AttendanceService) ListOfficeLocations(
	ctx context.Context,
) ([]domain.OfficeLocation, error) {
	locations, err := s.attendances.ListActiveOfficeLocations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list office locations: %w", err)
	}
	return locations, nil
}

// Record mencatat check-in atau check-out. Waktu server adalah satu-satunya sumber
// kebenaran; tidak ada timestamp dari client maupun dari isi gambar yang dipakai (D-028).
func (s *AttendanceService) Record(
	ctx context.Context,
	identity domain.Identity,
	command domain.RecordAttendance,
	meta RequestMeta,
) (domain.Attendance, error) {
	// Top Management tidak mencatat absensi; kontrak membatasi ke Karyawan, Atasan, dan HR.
	if identity.Role == domain.RoleTopManagement {
		return domain.Attendance{}, domain.ErrForbidden
	}

	networkTime := s.now().UTC()
	localTime := networkTime.In(domain.Jakarta())
	if !domain.IsWorkingDay(networkTime) {
		return domain.Attendance{}, domain.ErrNonWorkingDay
	}
	date := localTime.Format(domain.DateLayout)

	row := domain.AttendanceRow{
		UserID:      identity.UserID,
		Date:        date,
		Type:        command.Type,
		WorkMode:    command.WorkMode,
		NetworkTime: networkTime,
		// waktu_local diisi waktu server dalam WIB karena kontrak tidak mengirim waktu
		// perangkat dan waktu perangkat tidak tepercaya (D-022).
		LocalTime: localTime,
		Latitude:  command.Latitude,
		Longitude: command.Longitude,
	}

	if command.WorkMode == domain.WorkModeWFO {
		if command.OfficeLocationID == nil {
			return domain.Attendance{}, domain.ErrInvalidOfficeLocation
		}
		location, err := s.attendances.FindActiveOfficeLocation(ctx, *command.OfficeLocationID)
		if errors.Is(err, repository.ErrNotFound) {
			return domain.Attendance{}, domain.ErrInvalidOfficeLocation
		}
		if err != nil {
			return domain.Attendance{}, fmt.Errorf("resolve office location: %w", err)
		}
		// Radius dihitung terhadap koordinat tepercaya dari master, bukan dari client.
		distance := domain.DistanceMeters(
			command.Latitude, command.Longitude, location.Latitude, location.Longitude,
		)
		if distance > domain.OfficeRadiusMeters {
			return domain.Attendance{}, domain.ErrOutOfRadius
		}
		row.OfficeLocationID = &location.ID
		row.DistanceMeters = &distance
	}

	if err := s.checkSequence(ctx, identity.UserID, date, command.Type); err != nil {
		return domain.Attendance{}, err
	}

	switch command.Type {
	case domain.AttendanceCheckIn:
		row.Status = domain.CheckInStatus(networkTime)
	case domain.AttendanceCheckOut:
		row.Status = domain.CheckOutStatus(networkTime)
	default:
		return domain.Attendance{}, domain.ErrInvalidRequest
	}

	objectPath := attendancePhotoPath(identity.UserID, date, command.Type, command.PhotoExtension)
	location, err := s.photos.Upload(
		ctx, objectPath, bytes.NewReader(command.PhotoContent), command.PhotoMediaType,
	)
	if err != nil {
		return domain.Attendance{}, fmt.Errorf("upload attendance photo: %w", err)
	}
	row.PhotoURL = location

	var record domain.Attendance
	err = s.tx.Within(ctx, func(txContext context.Context) error {
		created, err := s.attendances.Create(txContext, row)
		if err != nil {
			return mapAttendanceRepositoryError(err, command.Type)
		}
		record = created
		record.EmployeeID = identity.EmployeeID
		return s.audit.Append(txContext, domain.AuditEntry{
			UserID: &identity.UserID,
			Action: "CREATE",
			Module: "absensi",
			DataID: &created.ID,
			Detail: map[string]any{
				"tipe":       string(command.Type),
				"mode_kerja": string(command.WorkMode),
				"status":     created.Status,
				"request_id": meta.RequestID,
			},
			IPAddress: meta.IPAddress,
			CreatedAt: networkTime,
		})
	})
	if err != nil {
		// Kompensasi foto yang sudah terunggah agar tidak menjadi orphan.
		if cleanupErr := s.photos.Delete(ctx, objectPath); cleanupErr != nil {
			slog.ErrorContext(ctx, "orphan attendance photo cleanup failed",
				"object_path", objectPath, "error", cleanupErr)
		}
		return domain.Attendance{}, fmt.Errorf("record attendance: %w", err)
	}
	return record, nil
}

// checkSequence menegakkan urutan absensi: satu check-in per tanggal dan check-out hanya
// setelah check-in tercatat.
func (s *AttendanceService) checkSequence(
	ctx context.Context,
	userID uuid.UUID,
	date string,
	attendanceType domain.AttendanceType,
) error {
	switch attendanceType {
	case domain.AttendanceCheckIn:
		exists, err := s.attendances.ExistsForDate(ctx, userID, date, domain.AttendanceCheckIn)
		if err != nil {
			return fmt.Errorf("check duplicate check-in: %w", err)
		}
		if exists {
			return domain.ErrDuplicateCheckIn
		}
	case domain.AttendanceCheckOut:
		hasCheckIn, err := s.attendances.ExistsForDate(ctx, userID, date, domain.AttendanceCheckIn)
		if err != nil {
			return fmt.Errorf("check check-in presence: %w", err)
		}
		if !hasCheckIn {
			return domain.ErrCheckoutWithoutCheckIn
		}
		exists, err := s.attendances.ExistsForDate(ctx, userID, date, domain.AttendanceCheckOut)
		if err != nil {
			return fmt.Errorf("check duplicate check-out: %w", err)
		}
		if exists {
			return domain.ErrDuplicateCheckIn
		}
	}
	return nil
}

// mapAttendanceRepositoryError menerjemahkan pelanggaran unique index menjadi error bisnis
// yang benar bila dua request bersaing lolos pemeriksaan awal.
func mapAttendanceRepositoryError(err error, attendanceType domain.AttendanceType) error {
	if errors.Is(err, repository.ErrConflict) {
		if attendanceType == domain.AttendanceCheckOut {
			return domain.ErrDuplicateCheckIn
		}
		return domain.ErrDuplicateCheckIn
	}
	return err
}

func attendancePhotoPath(
	userID uuid.UUID,
	date string,
	attendanceType domain.AttendanceType,
	extension string,
) string {
	return path.Join(
		"attendance-photos",
		userID.String(),
		date,
		fmt.Sprintf("%s-%s%s", attendanceType, uuid.NewString(), extension),
	)
}

// LiveFeed hanya untuk HR dan Top Management; Karyawan dan Atasan ditolak.
func (s *AttendanceService) LiveFeed(
	ctx context.Context,
	identity domain.Identity,
	date string,
) ([]domain.AttendanceLiveFeedItem, error) {
	if identity.Role != domain.RoleHR {
		return nil, domain.ErrForbidden
	}
	if date == "" {
		date = s.now().In(domain.Jakarta()).Format(domain.DateLayout)
	} else if _, err := time.ParseInLocation(domain.DateLayout, date, domain.Jakarta()); err != nil {
		return nil, domain.ErrInvalidRequest
	}
	items, err := s.attendances.LiveFeed(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("read attendance live feed: %w", err)
	}
	return items, nil
}

// ReportQuery adalah parameter laporan sebelum rentang diselesaikan.
type ReportQuery struct {
	Period       string
	Start        string
	End          string
	DepartmentID *uuid.UUID
	Page         int
	Limit        int
}

// Report menghitung rekap kehadiran. Rentang mengikuti keputusan D-026.
func (s *AttendanceService) Report(
	ctx context.Context,
	identity domain.Identity,
	query ReportQuery,
) (domain.AttendanceReportPage, error) {
	if identity.Role != domain.RoleHR {
		return domain.AttendanceReportPage{}, domain.ErrForbidden
	}
	filter, err := s.resolveReportRange(query)
	if err != nil {
		return domain.AttendanceReportPage{}, err
	}
	workingDays := workingDaysBetween(filter.Start, filter.End)
	page, err := s.attendances.Report(ctx, filter, workingDays)
	if err != nil {
		return domain.AttendanceReportPage{}, fmt.Errorf("read attendance report: %w", err)
	}
	return page, nil
}

func (s *AttendanceService) resolveReportRange(
	query ReportQuery,
) (domain.AttendanceReportFilter, error) {
	filter := domain.AttendanceReportFilter{
		DepartmentID: query.DepartmentID,
		Page:         query.Page,
		Limit:        query.Limit,
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	hasStart := query.Start != ""
	hasEnd := query.End != ""
	switch {
	case hasStart != hasEnd:
		// Rentang custom harus lengkap; setengah rentang tidak dapat ditafsirkan.
		return domain.AttendanceReportFilter{}, domain.ErrInvalidRequest
	case hasStart && hasEnd:
		start, err := time.ParseInLocation(domain.DateLayout, query.Start, domain.Jakarta())
		if err != nil {
			return domain.AttendanceReportFilter{}, domain.ErrInvalidRequest
		}
		end, err := time.ParseInLocation(domain.DateLayout, query.End, domain.Jakarta())
		if err != nil || end.Before(start) {
			return domain.AttendanceReportFilter{}, domain.ErrInvalidRequest
		}
		filter.Start, filter.End = start, end
	case query.Period != "":
		month, err := time.ParseInLocation(domain.PeriodLayout, query.Period, domain.Jakarta())
		if err != nil {
			return domain.AttendanceReportFilter{}, domain.ErrInvalidRequest
		}
		filter.Start = month
		filter.End = month.AddDate(0, 1, -1)
	default:
		now := s.now().In(domain.Jakarta())
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, domain.Jakarta())
		filter.Start = start
		filter.End = start.AddDate(0, 1, -1)
	}
	return filter, nil
}

func workingDaysBetween(start, end time.Time) int {
	count := 0
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		if day.Weekday() != time.Saturday && day.Weekday() != time.Sunday {
			count++
		}
	}
	return count
}

// maxReportExportRows membatasi dataset export laporan.
const maxReportExportRows = 5000

// ExportReport menghasilkan berkas laporan kehadiran. Hanya HR sesuai API Contract.
func (s *AttendanceService) ExportReport(
	ctx context.Context,
	identity domain.Identity,
	query ReportQuery,
	format domain.ExportFormat,
	meta RequestMeta,
) (domain.ExportFile, error) {
	if identity.Role != domain.RoleHR {
		return domain.ExportFile{}, domain.ErrForbidden
	}
	if !format.Valid() {
		return domain.ExportFile{}, domain.ErrInvalidRequest
	}
	filter, err := s.resolveReportRange(query)
	if err != nil {
		return domain.ExportFile{}, err
	}
	filter.Page = 1
	filter.Limit = maxReportExportRows

	page, err := s.attendances.Report(ctx, filter, workingDaysBetween(filter.Start, filter.End))
	if err != nil {
		return domain.ExportFile{}, fmt.Errorf("read attendance export dataset: %w", err)
	}
	if len(page.Items) == 0 {
		return domain.ExportFile{}, domain.ErrNotFound
	}

	table := attendanceReportTable(page.Items, filter)
	var buffer bytes.Buffer
	contentType := export.XLSXContentType
	if format == domain.ExportFormatPDF {
		contentType = export.PDFContentType
		err = export.WritePDF(&buffer, table)
	} else {
		err = export.WriteXLSX(&buffer, table)
	}
	if err != nil {
		return domain.ExportFile{}, fmt.Errorf("render attendance export: %w", err)
	}

	if err := s.audit.Append(ctx, domain.AuditEntry{
		UserID: &identity.UserID,
		Action: "DOWNLOAD",
		Module: "laporan_kehadiran",
		Detail: map[string]any{
			"format":          string(format),
			"tanggal_mulai":   filter.Start.Format(domain.DateLayout),
			"tanggal_selesai": filter.End.Format(domain.DateLayout),
			"jumlah_row":      len(page.Items),
			"request_id":      meta.RequestID,
		},
		IPAddress: meta.IPAddress,
		CreatedAt: s.now().UTC(),
	}); err != nil {
		return domain.ExportFile{}, fmt.Errorf("audit attendance export: %w", err)
	}

	return domain.ExportFile{
		FileName: export.SanitizeFileName(fmt.Sprintf(
			"laporan-kehadiran-%s-sd-%s.%s",
			filter.Start.Format(domain.DateLayout), filter.End.Format(domain.DateLayout), format,
		)),
		ContentType: contentType,
		Content:     buffer.Bytes(),
	}, nil
}

func attendanceReportTable(
	items []domain.AttendanceReportItem,
	filter domain.AttendanceReportFilter,
) export.Table {
	table := export.Table{
		Title: fmt.Sprintf(
			"GSNpeeps - Laporan Kehadiran %s s.d. %s WIB",
			filter.Start.Format(domain.DateLayout), filter.End.Format(domain.DateLayout),
		),
		Headers: []string{"Nama", "Departemen", "Hadir", "Terlambat", "Izin", "Alpha", "Total Jam"},
		Rows:    make([][]string, 0, len(items)),
	}
	for _, item := range items {
		table.Rows = append(table.Rows, []string{
			item.EmployeeName,
			item.Department,
			fmt.Sprintf("%d", item.Present),
			fmt.Sprintf("%d", item.Late),
			fmt.Sprintf("%d", item.Leave),
			fmt.Sprintf("%d", item.Absent),
			fmt.Sprintf("%.2f", item.TotalHours),
		})
	}
	return table
}
