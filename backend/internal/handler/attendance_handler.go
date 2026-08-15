package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/filetype"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
)

type AttendanceService interface {
	ListOfficeLocations(context.Context) ([]domain.OfficeLocation, error)
	Record(
		context.Context, domain.Identity, domain.RecordAttendance, service.RequestMeta,
	) (domain.Attendance, error)
	LiveFeed(context.Context, domain.Identity, string) ([]domain.AttendanceLiveFeedItem, error)
	ExportLiveFeed(
		context.Context, domain.Identity, string, service.RequestMeta,
	) (domain.ExportFile, error)
	Report(
		context.Context, domain.Identity, service.ReportQuery,
	) (domain.AttendanceReportPage, error)
	ExportReport(
		context.Context, domain.Identity, service.ReportQuery, domain.ExportFormat,
		service.RequestMeta,
	) (domain.ExportFile, error)
}

type AttendanceHandler struct {
	service    AttendanceService
	trustProxy bool
}

func NewAttendanceHandler(service AttendanceService, trustProxy bool) *AttendanceHandler {
	return &AttendanceHandler{service: service, trustProxy: trustProxy}
}

// ListOfficeLocations melayani dropdown lokasi WFO untuk seluruh role terautentikasi.
func (h *AttendanceHandler) ListOfficeLocations(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if _, ok := middleware.IdentityFromContext(request.Context()); !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	locations, err := h.service.ListOfficeLocations(request.Context())
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, locations, "OK")
}

func (h *AttendanceHandler) requestMeta(request *http.Request) service.RequestMeta {
	return service.RequestMeta{
		IPAddress: clientIP(request, h.trustProxy),
		RequestID: middleware.RequestIDFromContext(request.Context()),
	}
}

// Record menangani check-in dan check-out. Field `tipe` membedakan keduanya sesuai D-002.
func (h *AttendanceHandler) Record(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}

	request.Body = http.MaxBytesReader(writer, request.Body, maxDocumentBytes+multipartMemoryBytes)
	if err := request.ParseMultipartForm(multipartMemoryBytes); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			response.FromError(writer, domain.ErrFileTooLarge)
			return
		}
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	defer func() {
		if request.MultipartForm != nil {
			_ = request.MultipartForm.RemoveAll()
		}
	}()

	fields := map[string]string{}
	attendanceType := domain.AttendanceType(strings.TrimSpace(request.FormValue("tipe")))
	if !attendanceType.Valid() {
		fields["tipe"] = "Tipe absensi harus check_in atau check_out"
	}
	workMode := domain.WorkMode(strings.ToUpper(strings.TrimSpace(request.FormValue("mode_kerja"))))
	if !workMode.Valid() {
		fields["mode_kerja"] = "Mode kerja harus WFO, WFH, atau WFA"
	}
	// Koordinat wajib untuk seluruh mode kerja (D-012).
	latitude, err := strconv.ParseFloat(strings.TrimSpace(request.FormValue("gps_lat")), 64)
	if err != nil || latitude < -90 || latitude > 90 {
		fields["gps_lat"] = "Koordinat lintang wajib diisi dan valid"
	}
	longitude, err := strconv.ParseFloat(strings.TrimSpace(request.FormValue("gps_long")), 64)
	if err != nil || longitude < -180 || longitude > 180 {
		fields["gps_long"] = "Koordinat bujur wajib diisi dan valid"
	}

	var officeLocationID *uuid.UUID
	if raw := strings.TrimSpace(request.FormValue("office_location_id")); raw != "" {
		parsed, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			fields["office_location_id"] = "Lokasi kantor tidak valid"
		} else {
			officeLocationID = &parsed
		}
	} else if workMode == domain.WorkModeWFO {
		fields["office_location_id"] = "Lokasi kantor wajib dipilih untuk WFO"
	}
	// WFH dan WFA tidak menyimpan lokasi kantor.
	if workMode != domain.WorkModeWFO {
		officeLocationID = nil
	}

	if len(fields) > 0 {
		response.ValidationError(writer, fields)
		return
	}

	file, header, err := request.FormFile("foto")
	if err != nil {
		response.ValidationError(writer, map[string]string{"foto": "Foto absensi wajib diunggah"})
		return
	}
	defer file.Close()

	if header.Size > maxDocumentBytes {
		response.FromError(writer, domain.ErrFileTooLarge)
		return
	}
	content, err := io.ReadAll(io.LimitReader(file, maxDocumentBytes+1))
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Berkas tidak dapat dibaca")
		return
	}
	if len(content) > maxDocumentBytes {
		response.FromError(writer, domain.ErrFileTooLarge)
		return
	}
	if len(content) == 0 {
		response.ValidationError(writer, map[string]string{"foto": "Berkas foto kosong"})
		return
	}

	head := content
	if len(head) > signatureHeadBytes {
		head = head[:signatureHeadBytes]
	}
	// Watermark adalah tanggung jawab frontend; backend hanya memvalidasi berkas (D-028).
	descriptor, err := filetype.ValidatePhoto(header.Filename, header.Header.Get("Content-Type"), head)
	if err != nil {
		response.FromError(writer, domain.ErrUnsupportedFile)
		return
	}

	record, err := h.service.Record(request.Context(), identity, domain.RecordAttendance{
		UserID:           identity.UserID,
		EmployeeID:       identity.EmployeeID,
		Type:             attendanceType,
		WorkMode:         workMode,
		Latitude:         latitude,
		Longitude:        longitude,
		OfficeLocationID: officeLocationID,
		PhotoExtension:   descriptor.Extension,
		PhotoMediaType:   descriptor.MediaType,
		PhotoContent:     content,
	}, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusCreated, record, "Absensi berhasil dicatat")
}

func (h *AttendanceHandler) LiveFeed(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	items, err := h.service.LiveFeed(
		request.Context(), identity, strings.TrimSpace(request.URL.Query().Get("tanggal")),
	)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, items, "OK")
}

// ExportLiveFeed mengunduh live feed absensi pada satu tanggal sebagai XLSX. Hanya HR.
func (h *AttendanceHandler) ExportLiveFeed(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	file, err := h.service.ExportLiveFeed(
		request.Context(), identity, strings.TrimSpace(request.URL.Query().Get("tanggal")),
		h.requestMeta(request),
	)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	writeExportFile(writer, request, file)
}

func (h *AttendanceHandler) reportQuery(
	writer http.ResponseWriter,
	request *http.Request,
) (service.ReportQuery, bool) {
	departmentID, ok := optionalUUIDQuery(writer, request, "department_id")
	if !ok {
		return service.ReportQuery{}, false
	}
	page, ok := positiveIntQuery(writer, request, "page", 1)
	if !ok {
		return service.ReportQuery{}, false
	}
	limit, ok := positiveIntQuery(writer, request, "limit", 10)
	if !ok {
		return service.ReportQuery{}, false
	}
	query := request.URL.Query()
	return service.ReportQuery{
		Period:       strings.TrimSpace(query.Get("periode")),
		Start:        strings.TrimSpace(query.Get("tanggal_mulai")),
		End:          strings.TrimSpace(query.Get("tanggal_selesai")),
		DepartmentID: departmentID,
		Page:         page,
		Limit:        limit,
	}, true
}

func (h *AttendanceHandler) Report(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	query, ok := h.reportQuery(writer, request)
	if !ok {
		return
	}
	result, err := h.service.Report(request.Context(), identity, query)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Paginated(writer, result.Items, response.PaginationMeta{
		Page:      result.Page,
		Limit:     result.Limit,
		TotalData: result.Total,
		TotalPage: totalPages(result.Total, result.Limit),
	}, "OK")
}

func (h *AttendanceHandler) ExportReport(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	query, ok := h.reportQuery(writer, request)
	if !ok {
		return
	}
	format := domain.ExportFormat(
		strings.ToLower(strings.TrimSpace(request.URL.Query().Get("format"))),
	)
	if format == "" {
		format = domain.ExportFormatXLSX
	}
	if !format.Valid() {
		response.ValidationError(writer, map[string]string{
			"format": "Format export harus xlsx atau pdf",
		})
		return
	}

	file, err := h.service.ExportReport(
		request.Context(), identity, query, format, h.requestMeta(request),
	)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	writeExportFile(writer, request, file)
}
