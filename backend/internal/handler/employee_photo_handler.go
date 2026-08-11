package handler

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/filetype"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
)

// UpdateMyPhoto memperbarui foto profil identitas yang sedang login. Tersedia untuk seluruh
// role terautentikasi (termasuk Top Management yang tidak memiliki Profil Saya) karena
// navbar menampilkan foto untuk semua role.
func (h *EmployeeHandler) UpdateMyPhoto(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	h.updatePhoto(writer, request, identity, identity.EmployeeID)
}

// UpdatePhoto memperbarui foto profil karyawan tertentu. Hanya HR yang dapat memakai
// endpoint ini untuk karyawan lain; service tetap memeriksa ulang otorisasi (D-037).
func (h *EmployeeHandler) UpdatePhoto(writer http.ResponseWriter, request *http.Request) {
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
	h.updatePhoto(writer, request, identity, id)
}

func (h *EmployeeHandler) updatePhoto(
	writer http.ResponseWriter,
	request *http.Request,
	identity domain.Identity,
	employeeID uuid.UUID,
) {
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

	file, header, err := request.FormFile("foto")
	if err != nil {
		response.ValidationError(writer, map[string]string{"foto": "Foto profil wajib diunggah"})
		return
	}
	defer file.Close()

	upload, ok := readPhotoUpload(writer, file, header)
	if !ok {
		return
	}

	location, err := h.service.UpdatePhoto(
		request.Context(),
		identity,
		employeeID,
		upload,
		h.requestMeta(request),
	)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, struct {
		PhotoURL string `json:"foto_profil_url"`
	}{PhotoURL: location}, "Foto profil berhasil diperbarui")
}

func readPhotoUpload(
	writer http.ResponseWriter,
	file multipart.File,
	header *multipart.FileHeader,
) (service.PhotoUpload, bool) {
	if header.Size > maxDocumentBytes {
		response.Error(writer, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE",
			"Ukuran berkas melebihi batas 5 MB")
		return service.PhotoUpload{}, false
	}
	content, err := io.ReadAll(io.LimitReader(file, maxDocumentBytes+1))
	if err != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Berkas tidak dapat dibaca")
		return service.PhotoUpload{}, false
	}
	if len(content) > maxDocumentBytes {
		response.Error(writer, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE",
			"Ukuran berkas melebihi batas 5 MB")
		return service.PhotoUpload{}, false
	}
	if len(content) == 0 {
		response.ValidationError(writer, map[string]string{"foto": "Berkas foto kosong"})
		return service.PhotoUpload{}, false
	}

	head := content
	if len(head) > signatureHeadBytes {
		head = head[:signatureHeadBytes]
	}
	fileName := strings.TrimSpace(header.Filename)
	descriptor, err := filetype.ValidatePhoto(fileName, header.Header.Get("Content-Type"), head)
	if err != nil {
		response.Error(writer, http.StatusUnsupportedMediaType, "UNSUPPORTED_FILE_TYPE",
			"Format berkas tidak didukung")
		return service.PhotoUpload{}, false
	}
	return service.PhotoUpload{
		Extension: descriptor.Extension,
		MediaType: descriptor.MediaType,
		Content:   content,
	}, true
}
