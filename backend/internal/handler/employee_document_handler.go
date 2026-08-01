package handler

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/filetype"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
)

// maxDocumentBytes adalah batas 5 MB per berkas sesuai kontrak.
const maxDocumentBytes = 5 << 20

// multipartMemoryBytes membatasi bagian yang di-buffer di memory; sisanya ditulis ke berkas
// sementara oleh multipart reader.
const multipartMemoryBytes = 1 << 20

// signatureHeadBytes cukup untuk seluruh signature format yang disetujui.
const signatureHeadBytes = 512

// ListDocuments mengembalikan metadata dokumen karyawan (Point 12).
func (h *EmployeeHandler) ListDocuments(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	id, err := uuid.Parse(mux.Vars(request)["id"])
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "ID karyawan tidak valid")
		return
	}
	items, err := h.service.ListDocuments(request.Context(), identity, id)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, items, "OK")
}

// UploadDocument menerima multipart `jenis_dokumen` dan `file`, memvalidasi ukuran,
// extension, MIME, serta file signature sebelum meneruskannya ke service.
func (h *EmployeeHandler) UploadDocument(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	id, err := uuid.Parse(mux.Vars(request)["id"])
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_PARAM", "ID karyawan tidak valid")
		return
	}

	// Batas dihitung sebelum parsing agar berkas berlebih tidak pernah dibaca penuh.
	request.Body = http.MaxBytesReader(writer, request.Body, maxDocumentBytes+multipartMemoryBytes)
	if err := request.ParseMultipartForm(multipartMemoryBytes); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			response.Error(writer, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE",
				"Ukuran berkas melebihi batas 5 MB")
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

	documentType := strings.TrimSpace(request.FormValue("jenis_dokumen"))
	if documentType == "" || len(documentType) > 100 {
		response.ValidationError(writer, map[string]string{
			"jenis_dokumen": "Jenis dokumen wajib diisi maksimal 100 karakter",
		})
		return
	}

	file, header, err := request.FormFile("file")
	if err != nil {
		response.ValidationError(writer, map[string]string{"file": "Berkas dokumen wajib diunggah"})
		return
	}
	defer file.Close()

	upload, ok := readDocumentUpload(writer, file, header, documentType)
	if !ok {
		return
	}

	document, err := h.service.UploadDocument(
		request.Context(),
		identity,
		id,
		upload,
		h.requestMeta(request),
	)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusCreated, struct {
		ID      uuid.UUID `json:"id"`
		FileURL string    `json:"file_url"`
	}{ID: document.ID, FileURL: document.FileURL}, "Dokumen berhasil diunggah")
}

func readDocumentUpload(
	writer http.ResponseWriter,
	file multipart.File,
	header *multipart.FileHeader,
	documentType string,
) (service.DocumentUpload, bool) {
	if header.Size > maxDocumentBytes {
		response.Error(writer, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE",
			"Ukuran berkas melebihi batas 5 MB")
		return service.DocumentUpload{}, false
	}
	content, err := io.ReadAll(io.LimitReader(file, maxDocumentBytes+1))
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Berkas tidak dapat dibaca")
		return service.DocumentUpload{}, false
	}
	if len(content) > maxDocumentBytes {
		response.Error(writer, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE",
			"Ukuran berkas melebihi batas 5 MB")
		return service.DocumentUpload{}, false
	}
	if len(content) == 0 {
		response.ValidationError(writer, map[string]string{"file": "Berkas dokumen kosong"})
		return service.DocumentUpload{}, false
	}

	head := content
	if len(head) > signatureHeadBytes {
		head = head[:signatureHeadBytes]
	}
	fileName := filepath.Base(strings.TrimSpace(header.Filename))
	descriptor, err := filetype.ValidateDocument(fileName, header.Header.Get("Content-Type"), head)
	if err != nil {
		response.Error(writer, http.StatusUnsupportedMediaType, "UNSUPPORTED_FILE_TYPE",
			"Format berkas tidak didukung")
		return service.DocumentUpload{}, false
	}
	return service.DocumentUpload{
		Type:      documentType,
		FileName:  fileName,
		Extension: descriptor.Extension,
		MediaType: descriptor.MediaType,
		Content:   content,
	}, true
}
