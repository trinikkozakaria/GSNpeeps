package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

type Envelope struct {
	Success bool            `json:"success"`
	Data    any             `json:"data,omitempty"`
	Meta    *PaginationMeta `json:"meta,omitempty"`
	Message string          `json:"message,omitempty"`
	Error   *ErrorBody      `json:"error,omitempty"`
}

type PaginationMeta struct {
	Page      int `json:"page"`
	Limit     int `json:"limit"`
	TotalData int `json:"total_data"`
	TotalPage int `json:"total_page"`
}

type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func Success(writer http.ResponseWriter, status int, data any, message string) {
	writeJSON(writer, status, Envelope{Success: true, Data: data, Message: message})
}

func EmptySuccess(writer http.ResponseWriter, message string) {
	writeJSON(writer, http.StatusOK, Envelope{
		Success: true,
		Data:    json.RawMessage("null"),
		Message: message,
	})
}

func Paginated(writer http.ResponseWriter, data any, meta PaginationMeta, message string) {
	writeJSON(writer, http.StatusOK, Envelope{Success: true, Data: data, Meta: &meta, Message: message})
}

func Error(writer http.ResponseWriter, status int, code, message string) {
	writeJSON(writer, status, Envelope{Success: false, Error: &ErrorBody{Code: code, Message: message}})
}

func ValidationError(writer http.ResponseWriter, fields map[string]string) {
	writeJSON(writer, http.StatusUnprocessableEntity, Envelope{
		Success: false,
		Error:   &ErrorBody{Code: "VALIDATION_ERROR", Message: "Data belum valid", Fields: fields},
	})
}

func RequestValidationError(writer http.ResponseWriter, fields map[string]string) {
	writeJSON(writer, http.StatusBadRequest, Envelope{
		Success: false,
		Error: &ErrorBody{
			Code:    "INVALID_REQUEST",
			Message: "Request tidak valid",
			Fields:  fields,
		},
	})
}

func FromError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		Error(writer, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email atau password tidak valid")
	case errors.Is(err, domain.ErrAccountLocked):
		Error(writer, http.StatusTooManyRequests, "ACCOUNT_LOCKED", "Akun terkunci dan memerlukan pemulihan")
	case errors.Is(err, domain.ErrInactiveAccount):
		Error(writer, http.StatusForbidden, "ACCOUNT_INACTIVE", "Akun tidak aktif")
	case errors.Is(err, domain.ErrInvalidToken), errors.Is(err, domain.ErrSessionInvalid):
		Error(writer, http.StatusUnauthorized, "INVALID_TOKEN", "Sesi tidak valid atau telah berakhir")
	case errors.Is(err, domain.ErrForbidden):
		Error(writer, http.StatusForbidden, "FORBIDDEN", "Anda tidak memiliki akses")
	case errors.Is(err, domain.ErrRateLimited):
		Error(writer, http.StatusTooManyRequests, "RATE_LIMITED", "Terlalu banyak permintaan, silakan coba lagi")
	case errors.Is(err, domain.ErrPasswordMismatch):
		ValidationError(writer, map[string]string{
			"new_password_confirmation": "Konfirmasi password harus sama",
		})
	case errors.Is(err, domain.ErrPasswordUnchanged):
		ValidationError(writer, map[string]string{
			"new_password": "Password baru harus berbeda dari password saat ini",
		})
	case errors.Is(err, domain.ErrNotFound):
		Error(writer, http.StatusNotFound, "NOT_FOUND", "Data tidak ditemukan")
	case errors.Is(err, domain.ErrConflict):
		Error(writer, http.StatusConflict, "CONFLICT", "Data sudah digunakan atau telah berubah")
	case errors.Is(err, domain.ErrInvalidRequest):
		Error(writer, http.StatusBadRequest, "INVALID_PARAM", "Parameter request tidak valid")
	case errors.Is(err, domain.ErrAlreadyDecided):
		Error(writer, http.StatusConflict, "ALREADY_DECIDED", "Pengajuan sudah diputuskan")
	case errors.Is(err, domain.ErrNonWorkingDay):
		Error(writer, http.StatusUnprocessableEntity, "NON_WORKING_DAY",
			"Absensi reguler hanya tersedia Senin sampai Jumat")
	case errors.Is(err, domain.ErrOutOfRadius):
		Error(writer, http.StatusUnprocessableEntity, "OUT_OF_RADIUS",
			"Lokasi WFO berada di luar radius kantor")
	case errors.Is(err, domain.ErrDuplicateCheckIn):
		Error(writer, http.StatusUnprocessableEntity, "DUPLICATE_CHECKIN",
			"Check-in hari ini sudah tercatat")
	case errors.Is(err, domain.ErrCheckoutWithoutCheckIn):
		Error(writer, http.StatusUnprocessableEntity, "CHECKOUT_WITHOUT_CHECKIN",
			"Check-out memerlukan check-in aktif")
	case errors.Is(err, domain.ErrInvalidOfficeLocation):
		Error(writer, http.StatusUnprocessableEntity, "INVALID_OFFICE_LOCATION",
			"Lokasi kantor WFO tidak valid atau tidak aktif")
	case errors.Is(err, domain.ErrInsufficientLeaveBalance):
		Error(writer, http.StatusUnprocessableEntity, "INSUFFICIENT_LEAVE_BALANCE",
			"Saldo cuti tidak mencukupi")
	case errors.Is(err, domain.ErrDocumentRequired):
		ValidationError(writer, map[string]string{
			"dokumen_pendukung": "Dokumen pendukung wajib untuk jenis izin ini",
		})
	case errors.Is(err, domain.ErrFileTooLarge):
		Error(writer, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE",
			"Ukuran berkas melebihi batas 5 MB")
	case errors.Is(err, domain.ErrUnsupportedFile):
		Error(writer, http.StatusUnsupportedMediaType, "UNSUPPORTED_FILE_TYPE",
			"Format berkas tidak didukung")
	default:
		slog.Error("unmapped application error", "error", err)
		Error(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan internal")
	}
}

func writeJSON(writer http.ResponseWriter, status int, body Envelope) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(body); err != nil {
		slog.Error("write JSON response failed", "error", err)
	}
}
