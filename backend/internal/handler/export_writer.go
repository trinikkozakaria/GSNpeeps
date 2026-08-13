package handler

import (
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

// writeExportFile mengalirkan berkas export dengan header yang aman dan konsisten.
func writeExportFile(writer http.ResponseWriter, request *http.Request, file domain.ExportFile) {
	writer.Header().Set("Content-Type", file.ContentType)
	writer.Header().Set("Content-Disposition", `attachment; filename="`+file.FileName+`"`)
	writer.Header().Set("Content-Length", strconv.Itoa(len(file.Content)))
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write(file.Content); err != nil {
		slog.ErrorContext(request.Context(), "write export stream failed", "error", err)
	}
}

// totalPages menghitung jumlah halaman dari total data dan limit.
func totalPages(total, limit int) int {
	if total <= 0 || limit <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(limit)))
}
