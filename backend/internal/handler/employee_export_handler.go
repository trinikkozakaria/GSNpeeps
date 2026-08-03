package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
)

// Export mengalirkan berkas XLSX atau PDF. Response berupa file stream, bukan JSON.
func (h *EmployeeHandler) Export(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	employeeID, ok := optionalUUIDQuery(writer, request, "id")
	if !ok {
		return
	}
	departmentID, ok := optionalUUIDQuery(writer, request, "department_id")
	if !ok {
		return
	}
	format := domain.ExportFormat(strings.ToLower(strings.TrimSpace(request.URL.Query().Get("format"))))
	if format == "" {
		format = domain.ExportFormatXLSX
	}
	if !format.Valid() {
		response.ValidationError(writer, map[string]string{
			"format": "Format export harus xlsx atau pdf",
		})
		return
	}

	file, err := h.service.Export(request.Context(), identity, domain.EmployeeExportQuery{
		Format:     format,
		EmployeeID: employeeID,
		Filter: domain.EmployeeFilter{
			Search:       strings.TrimSpace(request.URL.Query().Get("search")),
			DepartmentID: departmentID,
			Status:       strings.TrimSpace(request.URL.Query().Get("status")),
		},
	}, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}

	writer.Header().Set("Content-Type", file.ContentType)
	writer.Header().Set("Content-Disposition", `attachment; filename="`+file.FileName+`"`)
	writer.Header().Set("Content-Length", strconv.Itoa(len(file.Content)))
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write(file.Content); err != nil {
		slog.ErrorContext(request.Context(), "write export stream failed", "error", err)
	}
}
