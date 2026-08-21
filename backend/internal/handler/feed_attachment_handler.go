package handler

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/filetype"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
	"github.com/jackc/pgx/v5"
)

const maxFeedAttachments = 5

type feedAttachmentUpload struct {
	FileName  string
	Extension string
	MediaType string
	Content   []byte
}

func (h *UATHandler) UploadFeedAttachment(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireHR(w, r); !ok {
		return
	}
	if h.media == nil {
		response.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Penyimpanan berkas belum tersedia")
		return
	}
	feedID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PARAM", "ID feed tidak valid")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentBytes+multipartMemoryBytes)
	if err := r.ParseMultipartForm(multipartMemoryBytes); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			response.Error(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "Ukuran berkas melebihi batas 5 MB")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile("file")
	if err != nil {
		response.ValidationError(w, map[string]string{"file": "Berkas wajib diunggah"})
		return
	}
	defer file.Close()
	upload, ok := readFeedAttachment(w, file, header)
	if !ok {
		return
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment belum dapat disimpan")
		return
	}
	defer tx.Rollback(r.Context())
	var count int
	if err = tx.QueryRow(r.Context(), `
		SELECT (SELECT COUNT(*) FROM company_feed_attachments WHERE feed_id=$1)
		FROM company_feeds WHERE id=$1 FOR UPDATE
	`, feedID).Scan(&count); errors.Is(err, pgx.ErrNoRows) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Company feed tidak ditemukan")
		return
	} else if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment belum dapat disimpan")
		return
	}
	if count >= maxFeedAttachments {
		response.Error(w, http.StatusConflict, "ATTACHMENT_LIMIT", "Maksimal 5 attachment per feed")
		return
	}

	attachmentID := uuid.New()
	objectPath := path.Join("company-feed", feedID.String(), attachmentID.String()+upload.Extension)
	storedPath, err := h.media.Upload(r.Context(), objectPath, bytes.NewReader(upload.Content), upload.MediaType)
	if err != nil {
		response.Error(w, http.StatusBadGateway, "STORAGE_ERROR", "Attachment belum dapat diunggah")
		return
	}
	if _, err = tx.Exec(r.Context(), `
		INSERT INTO company_feed_attachments(id,feed_id,stored_path,file_name,media_type,file_size)
		VALUES($1,$2,$3,$4,$5,$6)
	`, attachmentID, feedID, storedPath, upload.FileName, upload.MediaType, len(upload.Content)); err != nil {
		_ = h.media.Delete(r.Context(), storedPath)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment belum dapat disimpan")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		_ = h.media.Delete(r.Context(), storedPath)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment belum dapat disimpan")
		return
	}
	response.Success(w, http.StatusCreated, map[string]any{
		"id": attachmentID, "file_name": upload.FileName, "media_type": upload.MediaType,
		"file_size": len(upload.Content), "file_url": storedPath,
	}, "Attachment berhasil diunggah")
}

func readFeedAttachment(w http.ResponseWriter, file multipart.File, header *multipart.FileHeader) (feedAttachmentUpload, bool) {
	if header.Size > maxDocumentBytes {
		response.Error(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "Ukuran berkas melebihi batas 5 MB")
		return feedAttachmentUpload{}, false
	}
	content, err := io.ReadAll(io.LimitReader(file, maxDocumentBytes+1))
	if err != nil || len(content) == 0 {
		response.ValidationError(w, map[string]string{"file": "Berkas kosong atau tidak dapat dibaca"})
		return feedAttachmentUpload{}, false
	}
	if len(content) > maxDocumentBytes {
		response.Error(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "Ukuran berkas melebihi batas 5 MB")
		return feedAttachmentUpload{}, false
	}
	head := content
	if len(head) > signatureHeadBytes {
		head = head[:signatureHeadBytes]
	}
	fileName := filepath.Base(strings.TrimSpace(header.Filename))
	descriptor, err := filetype.ValidateSupportingDocument(fileName, header.Header.Get("Content-Type"), head)
	if err != nil {
		response.Error(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_FILE_TYPE", "Format berkas harus PDF, PNG, JPG, atau JPEG")
		return feedAttachmentUpload{}, false
	}
	return feedAttachmentUpload{FileName: fileName, Extension: descriptor.Extension, MediaType: descriptor.MediaType, Content: content}, true
}

func (h *UATHandler) DeleteFeedAttachment(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireHR(w, r); !ok {
		return
	}
	feedID, feedErr := uuid.Parse(mux.Vars(r)["id"])
	attachmentID, attachmentErr := uuid.Parse(mux.Vars(r)["attachmentId"])
	if feedErr != nil || attachmentErr != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PARAM", "ID feed atau attachment tidak valid")
		return
	}
	var storedPath string
	if err := h.db.QueryRow(r.Context(), `
		SELECT stored_path FROM company_feed_attachments WHERE id=$1 AND feed_id=$2
	`, attachmentID, feedID).Scan(&storedPath); errors.Is(err, pgx.ErrNoRows) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Attachment tidak ditemukan")
		return
	} else if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment belum dapat dihapus")
		return
	}
	if h.media == nil || h.media.Delete(r.Context(), storedPath) != nil {
		response.Error(w, http.StatusBadGateway, "STORAGE_ERROR", "Attachment belum dapat dihapus dari penyimpanan")
		return
	}
	if _, err := h.db.Exec(r.Context(), `DELETE FROM company_feed_attachments WHERE id=$1 AND feed_id=$2`, attachmentID, feedID); err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment belum dapat dihapus")
		return
	}
	response.Success(w, http.StatusOK, map[string]any{"id": attachmentID}, "Attachment berhasil dihapus")
}
