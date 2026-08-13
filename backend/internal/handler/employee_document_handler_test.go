package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type employeeServiceStub struct {
	uploadCalled bool
	upload       service.DocumentUpload
	exportFile   domain.ExportFile
	exportQuery  domain.EmployeeExportQuery
	exportErr    error
}

func (s *employeeServiceStub) ListDepartments(context.Context) ([]domain.Department, error) {
	return nil, nil
}

func (s *employeeServiceStub) ListPositions(
	context.Context,
	*uuid.UUID,
) ([]domain.Position, error) {
	return nil, nil
}

func (s *employeeServiceStub) List(
	context.Context,
	domain.Identity,
	domain.EmployeeFilter,
) (domain.EmployeePage, error) {
	return domain.EmployeePage{}, nil
}

func (s *employeeServiceStub) Detail(
	context.Context,
	domain.Identity,
	uuid.UUID,
) (domain.EmployeeDetail, error) {
	return domain.EmployeeDetail{}, nil
}

func (s *employeeServiceStub) Create(
	context.Context,
	domain.Identity,
	dto.CreateEmployeeRequest,
	service.RequestMeta,
) (domain.EmployeeMutationResult, error) {
	return domain.EmployeeMutationResult{}, nil
}

func (s *employeeServiceStub) Update(
	context.Context,
	domain.Identity,
	uuid.UUID,
	dto.UpdateEmployeeRequest,
	service.RequestMeta,
) (domain.EmployeeMutationResult, error) {
	return domain.EmployeeMutationResult{}, nil
}

func (s *employeeServiceStub) Deactivate(
	context.Context,
	domain.Identity,
	uuid.UUID,
	service.RequestMeta,
) (domain.EmployeeMutationResult, error) {
	return domain.EmployeeMutationResult{}, nil
}

func (s *employeeServiceStub) ListDocuments(
	context.Context,
	domain.Identity,
	uuid.UUID,
) ([]domain.EmployeeDocument, error) {
	return []domain.EmployeeDocument{}, nil
}

func (s *employeeServiceStub) UploadDocument(
	_ context.Context,
	_ domain.Identity,
	_ uuid.UUID,
	upload service.DocumentUpload,
	_ service.RequestMeta,
) (domain.EmployeeDocument, error) {
	s.uploadCalled = true
	s.upload = upload
	return domain.EmployeeDocument{
		ID:      uuid.New(),
		FileURL: "GSNpeeps/employee-documents/berkas.pdf",
	}, nil
}

func (s *employeeServiceStub) Export(
	_ context.Context,
	_ domain.Identity,
	query domain.EmployeeExportQuery,
	_ service.RequestMeta,
) (domain.ExportFile, error) {
	s.exportQuery = query
	return s.exportFile, s.exportErr
}

func (s *employeeServiceStub) UpdatePhoto(
	context.Context, domain.Identity, uuid.UUID, service.PhotoUpload, service.RequestMeta,
) (string, error) {
	return "GSNpeeps/employee-photos/foto.jpg", nil
}

type permissiveValidator struct{}

func (permissiveValidator) Struct(any) map[string]string { return nil }

func newDocumentRouter(stub *employeeServiceStub) http.Handler {
	handler := NewEmployeeHandler(stub, permissiveValidator{}, false)
	router := mux.NewRouter()
	router.HandleFunc("/karyawan/export", handler.Export).Methods(http.MethodGet)
	router.HandleFunc("/karyawan/{id}/dokumen", handler.UploadDocument).Methods(http.MethodPost)
	router.HandleFunc("/karyawan/{id}/dokumen", handler.ListDocuments).Methods(http.MethodGet)
	return router
}

func authenticated(request *http.Request) *http.Request {
	return request.WithContext(middleware.WithIdentity(request.Context(), domain.Identity{
		UserID:     uuid.New(),
		EmployeeID: uuid.New(),
		Role:       domain.RoleHR,
	}))
}

func multipartDocument(t *testing.T, fileName, mediaType string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("jenis_dokumen", "Ijazah"))

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition",
		`form-data; name="file"; filename="`+fileName+`"`)
	if mediaType != "" {
		header.Set("Content-Type", mediaType)
	}
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return &body, writer.FormDataContentType()
}

func TestUploadDocumentAcceptsApprovedFile(t *testing.T) {
	stub := &employeeServiceStub{}
	body, contentType := multipartDocument(t, "ijazah.pdf", "application/pdf", []byte("%PDF-1.4 sintetis"))

	request := httptest.NewRequest(http.MethodPost, "/karyawan/"+uuid.NewString()+"/dokumen", body)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	newDocumentRouter(stub).ServeHTTP(recorder, authenticated(request))

	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())
	assert.True(t, stub.uploadCalled)
	assert.Equal(t, ".pdf", stub.upload.Extension)
	assert.Equal(t, "application/pdf", stub.upload.MediaType)
	assert.Equal(t, "Ijazah", stub.upload.Type)
}

// Nama berkas dari client tidak boleh membawa komponen path.
func TestUploadDocumentStripsClientPathFromFileName(t *testing.T) {
	stub := &employeeServiceStub{}
	body, contentType := multipartDocument(
		t, "../../etc/ijazah.pdf", "application/pdf", []byte("%PDF-1.4 sintetis"),
	)

	request := httptest.NewRequest(http.MethodPost, "/karyawan/"+uuid.NewString()+"/dokumen", body)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	newDocumentRouter(stub).ServeHTTP(recorder, authenticated(request))

	require.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, "ijazah.pdf", stub.upload.FileName)
	assert.NotContains(t, stub.upload.FileName, "/")
}

func TestUploadDocumentRejectsUnsupportedTypeWith415(t *testing.T) {
	cases := []struct {
		name      string
		fileName  string
		mediaType string
		content   []byte
	}{
		{"arsip zip", "arsip.zip", "application/zip", []byte("PK\x03\x04")},
		{"arsip rar", "arsip.rar", "application/vnd.rar", []byte("Rar!\x1a\x07\x00")},
		{"executable menyamar", "ijazah.pdf", "application/pdf", []byte("MZ\x90\x00")},
		{"skrip", "jalankan.sh", "text/x-shellscript", []byte("#!/bin/sh\n")},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stub := &employeeServiceStub{}
			body, contentType := multipartDocument(
				t, testCase.fileName, testCase.mediaType, testCase.content,
			)

			request := httptest.NewRequest(http.MethodPost, "/karyawan/"+uuid.NewString()+"/dokumen", body)
			request.Header.Set("Content-Type", contentType)
			recorder := httptest.NewRecorder()
			newDocumentRouter(stub).ServeHTTP(recorder, authenticated(request))

			require.Equal(t, http.StatusUnsupportedMediaType, recorder.Code)
			assert.Contains(t, recorder.Body.String(), "UNSUPPORTED_FILE_TYPE")
			assert.False(t, stub.uploadCalled, "berkas tertolak tidak boleh mencapai service")
		})
	}
}

func TestUploadDocumentRejectsOversizeFileWith413(t *testing.T) {
	stub := &employeeServiceStub{}
	content := make([]byte, maxDocumentBytes+1024)
	copy(content, "%PDF-1.4")
	body, contentType := multipartDocument(t, "besar.pdf", "application/pdf", content)

	request := httptest.NewRequest(http.MethodPost, "/karyawan/"+uuid.NewString()+"/dokumen", body)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	newDocumentRouter(stub).ServeHTTP(recorder, authenticated(request))

	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "FILE_TOO_LARGE")
	assert.False(t, stub.uploadCalled)
}

func TestUploadDocumentRequiresDocumentType(t *testing.T) {
	stub := &employeeServiceStub{}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "ijazah.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("%PDF-1.4 sintetis"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/karyawan/"+uuid.NewString()+"/dokumen", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	newDocumentRouter(stub).ServeHTTP(recorder, authenticated(request))

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "jenis_dokumen")
	assert.False(t, stub.uploadCalled)
}

func TestUploadDocumentRejectsUnauthenticatedRequest(t *testing.T) {
	stub := &employeeServiceStub{}
	body, contentType := multipartDocument(t, "ijazah.pdf", "application/pdf", []byte("%PDF-1.4"))

	request := httptest.NewRequest(http.MethodPost, "/karyawan/"+uuid.NewString()+"/dokumen", body)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	newDocumentRouter(stub).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.False(t, stub.uploadCalled)
}

func TestExportStreamsFileWithSafeHeaders(t *testing.T) {
	stub := &employeeServiceStub{exportFile: domain.ExportFile{
		FileName:    "karyawan-20260801-100000.xlsx",
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Content:     []byte("PK\x03\x04konten"),
	}}

	request := httptest.NewRequest(http.MethodGet, "/karyawan/export?format=xlsx&status=aktif", nil)
	recorder := httptest.NewRecorder()
	newDocumentRouter(stub).ServeHTTP(recorder, authenticated(request))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, stub.exportFile.ContentType, recorder.Header().Get("Content-Type"))
	assert.Equal(t,
		`attachment; filename="karyawan-20260801-100000.xlsx"`,
		recorder.Header().Get("Content-Disposition"),
	)
	assert.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, stub.exportFile.Content, recorder.Body.Bytes())
	assert.Equal(t, domain.ExportFormatXLSX, stub.exportQuery.Format)
	assert.Equal(t, "aktif", stub.exportQuery.Filter.Status)
}

func TestExportRejectsFormatOutsideContractEnum(t *testing.T) {
	stub := &employeeServiceStub{}

	request := httptest.NewRequest(http.MethodGet, "/karyawan/export?format=csv", nil)
	recorder := httptest.NewRecorder()
	newDocumentRouter(stub).ServeHTTP(recorder, authenticated(request))

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "format")
}

func TestExportDefaultsToXLSX(t *testing.T) {
	stub := &employeeServiceStub{exportFile: domain.ExportFile{
		FileName:    "karyawan.xlsx",
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Content:     []byte("PK"),
	}}

	request := httptest.NewRequest(http.MethodGet, "/karyawan/export", nil)
	recorder := httptest.NewRecorder()
	newDocumentRouter(stub).ServeHTTP(recorder, authenticated(request))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, domain.ExportFormatXLSX, stub.exportQuery.Format)
}

func TestListDocumentsReturnsJSONEnvelope(t *testing.T) {
	stub := &employeeServiceStub{}

	request := httptest.NewRequest(http.MethodGet, "/karyawan/"+uuid.NewString()+"/dokumen", nil)
	recorder := httptest.NewRecorder()
	newDocumentRouter(stub).ServeHTTP(recorder, authenticated(request))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/json"))

	var envelope struct {
		Success bool              `json:"success"`
		Data    []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	assert.True(t, envelope.Success)
}
