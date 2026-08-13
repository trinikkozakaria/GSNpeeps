package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type employeeReaderStub struct {
	receivedFilter    domain.EmployeeFilter
	detailError       error
	createCommand     domain.CreateEmployee
	detailPeriod      string
	detail            domain.EmployeeDetail
	existsError       error
	documents         []domain.EmployeeDocument
	createdDocument   domain.NewEmployeeDocument
	createDocumentErr error
	exportRows        []domain.EmployeeSummary
	exportQuery       domain.EmployeeExportQuery
	updatedPhotoID    uuid.UUID
	updatedPhotoURL   string
	updatePhotoErr    error
}

func (s *employeeReaderStub) ValidateCreate(context.Context, domain.CreateEmployee) error {
	return nil
}

func (s *employeeReaderStub) Create(
	_ context.Context,
	command domain.CreateEmployee,
) (domain.EmployeeMutationResult, error) {
	s.createCommand = command
	return domain.EmployeeMutationResult{EmployeeID: uuid.New()}, nil
}

func (s *employeeReaderStub) ValidateMutation(
	context.Context,
	uuid.UUID,
	domain.EmployeeChanges,
) error {
	return nil
}

func (s *employeeReaderStub) Update(
	context.Context,
	uuid.UUID,
	domain.EmployeeChanges,
) (domain.EmployeeMutationResult, error) {
	return domain.EmployeeMutationResult{}, nil
}

func (s *employeeReaderStub) SoftDelete(
	context.Context,
	uuid.UUID,
) (domain.EmployeeMutationResult, error) {
	return domain.EmployeeMutationResult{}, nil
}

func (s *employeeReaderStub) UpdatePhoto(_ context.Context, employeeID uuid.UUID, url string) error {
	s.updatedPhotoID = employeeID
	s.updatedPhotoURL = url
	return s.updatePhotoErr
}

type transactionStub struct{}

func (transactionStub) Within(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type auditStub struct{}

func (auditStub) Append(context.Context, domain.AuditEntry) error { return nil }

type sessionRevokerStub struct{}

func (sessionRevokerStub) Revoke(context.Context, uuid.UUID) error { return nil }

type passwordHasherStub struct{}

func (passwordHasherStub) Hash(string) (string, error)         { return "locked-hash", nil }
func (passwordHasherStub) Verify(string, string) (bool, error) { return false, nil }

type failingTransactionStub struct{ err error }

func (s failingTransactionStub) Within(context.Context, func(context.Context) error) error {
	return s.err
}

func newEmployeeServiceForTest(reader EmployeeReader) *EmployeeService {
	return newEmployeeServiceWith(reader, transactionStub{}, &documentStoreStub{})
}

func newEmployeeServiceWith(
	reader EmployeeReader,
	transactions EmployeeTransactionManager,
	documents DocumentStore,
) *EmployeeService {
	service := NewEmployeeService(
		reader,
		transactions,
		auditStub{},
		sessionRevokerStub{},
		passwordHasherStub{},
		documents,
	)
	service.now = func() time.Time {
		return time.Date(2026, time.August, 1, 10, 0, 0, 0, domain.Jakarta())
	}
	return service
}

func (s *employeeReaderStub) ListDepartments(context.Context) ([]domain.Department, error) {
	return []domain.Department{}, nil
}

func (s *employeeReaderStub) ListPositions(
	context.Context,
	*uuid.UUID,
) ([]domain.Position, error) {
	return []domain.Position{}, nil
}

func (s *employeeReaderStub) List(
	_ context.Context,
	filter domain.EmployeeFilter,
) (domain.EmployeePage, error) {
	s.receivedFilter = filter
	return domain.EmployeePage{Items: []domain.EmployeeSummary{}, Page: filter.Page, Limit: filter.Limit}, nil
}

func (s *employeeReaderStub) FindByID(
	_ context.Context,
	_ uuid.UUID,
	salaryPeriod string,
) (domain.EmployeeDetail, error) {
	s.detailPeriod = salaryPeriod
	return s.detail, s.detailError
}

func (s *employeeReaderStub) ExistsActive(context.Context, uuid.UUID) error {
	return s.existsError
}

func (s *employeeReaderStub) FindDocuments(
	context.Context,
	uuid.UUID,
) ([]domain.EmployeeDocument, error) {
	return s.documents, nil
}

func (s *employeeReaderStub) CreateDocument(
	_ context.Context,
	document domain.NewEmployeeDocument,
) (uuid.UUID, error) {
	s.createdDocument = document
	if s.createDocumentErr != nil {
		return uuid.Nil, s.createDocumentErr
	}
	return uuid.New(), nil
}

func (s *employeeReaderStub) ExportRows(
	_ context.Context,
	query domain.EmployeeExportQuery,
	_ int,
) ([]domain.EmployeeSummary, error) {
	s.exportQuery = query
	return s.exportRows, nil
}

// documentStoreStub merekam object yang diunggah dan dihapus agar kompensasi orphan dapat
// diverifikasi tanpa Nextcloud sungguhan.
type documentStoreStub struct {
	uploadedPath string
	deletedPath  string
	uploadErr    error
}

func (s *documentStoreStub) Upload(
	_ context.Context,
	objectPath string,
	body io.Reader,
	_ string,
) (string, error) {
	if s.uploadErr != nil {
		return "", s.uploadErr
	}
	if _, err := io.ReadAll(body); err != nil {
		return "", err
	}
	s.uploadedPath = objectPath
	return "GSNpeeps/" + objectPath, nil
}

func (s *documentStoreStub) Delete(_ context.Context, objectPath string) error {
	s.deletedPath = objectPath
	return nil
}

func TestEmployeeListRestrictsRoleAndAppliesBounds(t *testing.T) {
	stub := &employeeReaderStub{}
	service := newEmployeeServiceForTest(stub)

	_, err := service.List(context.Background(), domain.Identity{Role: domain.RoleEmployee}, domain.EmployeeFilter{})
	require.ErrorIs(t, err, domain.ErrForbidden)

	_, err = service.List(context.Background(), domain.Identity{Role: domain.RoleHR}, domain.EmployeeFilter{
		Page:  -1,
		Limit: 999,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, stub.receivedFilter.Page)
	assert.Equal(t, 100, stub.receivedFilter.Limit)
}

func TestEmployeeListRejectsUnknownStatus(t *testing.T) {
	service := newEmployeeServiceForTest(&employeeReaderStub{})

	_, err := service.List(context.Background(), domain.Identity{Role: domain.RoleHR}, domain.EmployeeFilter{
		Status: "resign",
	})

	require.ErrorIs(t, err, domain.ErrInvalidRequest)
}

func TestEmployeeDetailMapsRepositoryNotFound(t *testing.T) {
	service := newEmployeeServiceForTest(&employeeReaderStub{detailError: repository.ErrNotFound})

	_, err := service.Detail(
		context.Background(),
		domain.Identity{Role: domain.RoleHR},
		uuid.New(),
	)

	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestEmployeeDetailRequestsCurrentMonthSalaryOnly(t *testing.T) {
	stub := &employeeReaderStub{}
	service := newEmployeeServiceForTest(stub)

	_, err := service.Detail(
		context.Background(),
		domain.Identity{Role: domain.RoleHR},
		uuid.New(),
	)

	require.NoError(t, err)
	assert.Equal(t, "2026-08", stub.detailPeriod)
}

func TestEmployeeDocumentAccessMatrix(t *testing.T) {
	forbidden := []domain.RoleName{domain.RoleEmployee, domain.RoleSupervisor, domain.RoleTopManagement}
	for _, role := range forbidden {
		service := newEmployeeServiceForTest(&employeeReaderStub{})

		_, listErr := service.ListDocuments(
			context.Background(),
			domain.Identity{Role: role},
			uuid.New(),
		)
		require.ErrorIs(t, listErr, domain.ErrForbidden)

		_, uploadErr := service.UploadDocument(
			context.Background(),
			domain.Identity{Role: role},
			uuid.New(),
			validDocumentUpload(),
			RequestMeta{},
		)
		require.ErrorIs(t, uploadErr, domain.ErrForbidden)
	}

}

func TestEmployeeUploadDocumentDoesNotTouchStorageForForbiddenRole(t *testing.T) {
	store := &documentStoreStub{}
	service := newEmployeeServiceWith(&employeeReaderStub{}, transactionStub{}, store)

	_, err := service.UploadDocument(
		context.Background(),
		domain.Identity{Role: domain.RoleEmployee},
		uuid.New(),
		validDocumentUpload(),
		RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrForbidden)
	assert.Empty(t, store.uploadedPath, "request tertolak tidak boleh mengunggah berkas")
}

func TestEmployeeUploadDocumentBuildsNamespacedServerSidePath(t *testing.T) {
	stub := &employeeReaderStub{}
	store := &documentStoreStub{}
	service := newEmployeeServiceWith(stub, transactionStub{}, store)
	employeeID := uuid.New()

	document, err := service.UploadDocument(
		context.Background(),
		domain.Identity{UserID: uuid.New(), Role: domain.RoleHR},
		employeeID,
		validDocumentUpload(),
		RequestMeta{},
	)

	require.NoError(t, err)
	assert.True(
		t,
		strings.HasPrefix(store.uploadedPath, "employee-documents/"+employeeID.String()+"/"),
		"object path harus ter-namespace per karyawan: %s", store.uploadedPath,
	)
	assert.True(t, strings.HasSuffix(store.uploadedPath, ".pdf"))
	assert.NotContains(t, store.uploadedPath, "ijazah-asli.pdf",
		"nama berkas dari client tidak boleh menjadi nama object")
	assert.Equal(t, "ijazah-asli.pdf", stub.createdDocument.FileName)
	assert.Equal(t, store.uploadedPath, strings.TrimPrefix(document.FileURL, "GSNpeeps/"))
}

func TestEmployeeUploadDocumentRemovesOrphanWhenTransactionFails(t *testing.T) {
	store := &documentStoreStub{}
	service := newEmployeeServiceWith(
		&employeeReaderStub{},
		failingTransactionStub{err: errors.New("database unavailable")},
		store,
	)

	_, err := service.UploadDocument(
		context.Background(),
		domain.Identity{UserID: uuid.New(), Role: domain.RoleHR},
		uuid.New(),
		validDocumentUpload(),
		RequestMeta{},
	)

	require.Error(t, err)
	assert.NotEmpty(t, store.uploadedPath)
	assert.Equal(t, store.uploadedPath, store.deletedPath,
		"object yang sudah terunggah harus dihapus ketika transaction gagal")
}

func TestEmployeeUploadDocumentRejectsOversizeContent(t *testing.T) {
	store := &documentStoreStub{}
	service := newEmployeeServiceWith(&employeeReaderStub{}, transactionStub{}, store)
	upload := validDocumentUpload()
	upload.Content = make([]byte, maxDocumentBytes+1)

	_, err := service.UploadDocument(
		context.Background(),
		domain.Identity{UserID: uuid.New(), Role: domain.RoleHR},
		uuid.New(),
		upload,
		RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrInvalidRequest)
	assert.Empty(t, store.uploadedPath)
}

func validPhotoUpload() PhotoUpload {
	return PhotoUpload{
		Extension: ".jpg",
		MediaType: "image/jpeg",
		Content:   []byte("\xff\xd8\xff\xe0 foto sintetis"),
	}
}

// HR boleh memperbarui foto karyawan mana pun; karyawan/atasan hanya boleh memperbarui
// fotonya sendiri (D-037).
func TestEmployeeUpdatePhotoAuthorization(t *testing.T) {
	employeeID := uuid.New()

	store := &documentStoreStub{}
	service := newEmployeeServiceWith(&employeeReaderStub{}, transactionStub{}, store)
	_, err := service.UpdatePhoto(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR},
		employeeID,
		validPhotoUpload(),
		RequestMeta{},
	)
	require.NoError(t, err, "HR dapat memperbarui foto karyawan lain")

	store = &documentStoreStub{}
	service = newEmployeeServiceWith(&employeeReaderStub{}, transactionStub{}, store)
	_, err = service.UpdatePhoto(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: employeeID, Role: domain.RoleEmployee},
		employeeID,
		validPhotoUpload(),
		RequestMeta{},
	)
	require.NoError(t, err, "karyawan dapat memperbarui foto miliknya sendiri")

	store = &documentStoreStub{}
	service = newEmployeeServiceWith(&employeeReaderStub{}, transactionStub{}, store)
	_, err = service.UpdatePhoto(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee},
		employeeID,
		validPhotoUpload(),
		RequestMeta{},
	)
	require.ErrorIs(t, err, domain.ErrForbidden, "karyawan tidak boleh mengubah foto orang lain")
	assert.Empty(t, store.uploadedPath, "request tertolak tidak boleh mengunggah berkas")
}

func TestEmployeeUpdatePhotoBuildsNamespacedServerSidePathAndPersistsURL(t *testing.T) {
	stub := &employeeReaderStub{}
	store := &documentStoreStub{}
	service := newEmployeeServiceWith(stub, transactionStub{}, store)
	employeeID := uuid.New()

	location, err := service.UpdatePhoto(
		context.Background(),
		domain.Identity{UserID: uuid.New(), Role: domain.RoleHR},
		employeeID,
		validPhotoUpload(),
		RequestMeta{},
	)

	require.NoError(t, err)
	assert.True(
		t,
		strings.HasPrefix(store.uploadedPath, "employee-photos/"+employeeID.String()+"/"),
		"object path harus ter-namespace per karyawan: %s", store.uploadedPath,
	)
	assert.True(t, strings.HasSuffix(store.uploadedPath, ".jpg"))
	assert.Equal(t, employeeID, stub.updatedPhotoID)
	assert.Equal(t, location, stub.updatedPhotoURL)
	assert.Equal(t, "GSNpeeps/"+store.uploadedPath, location)
}

func TestEmployeeUpdatePhotoRemovesOrphanWhenTransactionFails(t *testing.T) {
	store := &documentStoreStub{}
	service := newEmployeeServiceWith(
		&employeeReaderStub{},
		failingTransactionStub{err: errors.New("database unavailable")},
		store,
	)

	_, err := service.UpdatePhoto(
		context.Background(),
		domain.Identity{UserID: uuid.New(), Role: domain.RoleHR},
		uuid.New(),
		validPhotoUpload(),
		RequestMeta{},
	)

	require.Error(t, err)
	assert.NotEmpty(t, store.uploadedPath)
	assert.Equal(t, store.uploadedPath, store.deletedPath,
		"object yang sudah terunggah harus dihapus ketika transaction gagal")
}

func TestEmployeeUpdatePhotoRejectsOversizeContent(t *testing.T) {
	store := &documentStoreStub{}
	service := newEmployeeServiceWith(&employeeReaderStub{}, transactionStub{}, store)
	upload := validPhotoUpload()
	upload.Content = make([]byte, maxDocumentBytes+1)

	_, err := service.UpdatePhoto(
		context.Background(),
		domain.Identity{UserID: uuid.New(), Role: domain.RoleHR},
		uuid.New(),
		upload,
		RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrInvalidRequest)
	assert.Empty(t, store.uploadedPath)
}

func TestEmployeeExportRequiresHR(t *testing.T) {
	for _, role := range []domain.RoleName{
		domain.RoleEmployee,
		domain.RoleSupervisor,
		domain.RoleTopManagement,
	} {
		service := newEmployeeServiceForTest(&employeeReaderStub{exportRows: exportFixture()})

		_, err := service.Export(
			context.Background(),
			domain.Identity{Role: role},
			domain.EmployeeExportQuery{Format: domain.ExportFormatXLSX},
			RequestMeta{},
		)

		require.ErrorIsf(t, err, domain.ErrForbidden, "role %s harus ditolak", role)
	}
}

func TestEmployeeExportRejectsUnknownFormat(t *testing.T) {
	service := newEmployeeServiceForTest(&employeeReaderStub{exportRows: exportFixture()})

	_, err := service.Export(
		context.Background(),
		domain.Identity{Role: domain.RoleHR},
		domain.EmployeeExportQuery{Format: domain.ExportFormat("csv")},
		RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrInvalidRequest)
}

func TestEmployeeExportReturnsNotFoundForEmptyResult(t *testing.T) {
	service := newEmployeeServiceForTest(&employeeReaderStub{exportRows: nil})

	_, err := service.Export(
		context.Background(),
		domain.Identity{Role: domain.RoleHR},
		domain.EmployeeExportQuery{Format: domain.ExportFormatXLSX},
		RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestEmployeeExportProducesXLSXWithNeutralizedFormula(t *testing.T) {
	rows := exportFixture()
	rows[0].Name = "=HYPERLINK(\"http://jahat.test\")"
	service := newEmployeeServiceForTest(&employeeReaderStub{exportRows: rows})

	file, err := service.Export(
		context.Background(),
		domain.Identity{Role: domain.RoleHR},
		domain.EmployeeExportQuery{Format: domain.ExportFormatXLSX},
		RequestMeta{},
	)

	require.NoError(t, err)
	assert.Equal(
		t,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		file.ContentType,
	)
	assert.True(t, strings.HasSuffix(file.FileName, ".xlsx"))

	archive, err := zip.NewReader(bytes.NewReader(file.Content), int64(len(file.Content)))
	require.NoError(t, err)
	sheet := readZipEntry(t, archive, "xl/worksheets/sheet1.xml")
	assert.Contains(t, sheet, "&#39;=HYPERLINK", "formula harus dinetralkan dengan apostrof")
	assert.NotContains(t, sheet, "<t xml:space=\"preserve\">=HYPERLINK")
}

func TestEmployeeExportProducesPDFStream(t *testing.T) {
	service := newEmployeeServiceForTest(&employeeReaderStub{exportRows: exportFixture()})

	file, err := service.Export(
		context.Background(),
		domain.Identity{Role: domain.RoleHR},
		domain.EmployeeExportQuery{Format: domain.ExportFormatPDF},
		RequestMeta{},
	)

	require.NoError(t, err)
	assert.Equal(t, "application/pdf", file.ContentType)
	assert.True(t, strings.HasSuffix(file.FileName, ".pdf"))
	assert.True(t, bytes.HasPrefix(file.Content, []byte("%PDF-")))
	assert.Contains(t, string(file.Content), "%%EOF")
}

func validDocumentUpload() DocumentUpload {
	return DocumentUpload{
		Type:      "Ijazah",
		FileName:  "ijazah-asli.pdf",
		Extension: ".pdf",
		MediaType: "application/pdf",
		Content:   []byte("%PDF-1.4 dokumen sintetis"),
	}
}

func exportFixture() []domain.EmployeeSummary {
	return []domain.EmployeeSummary{{
		ID:         uuid.New(),
		NIP:        "EMP-001",
		Name:       "Karyawan Uji",
		Email:      "employee@example.test",
		Department: "Teknologi",
		Position:   "Staff",
		Status:     "aktif",
	}}
}

func readZipEntry(t *testing.T, archive *zip.Reader, name string) string {
	t.Helper()
	for _, entry := range archive.File {
		if entry.Name != name {
			continue
		}
		reader, err := entry.Open()
		require.NoError(t, err)
		defer reader.Close()
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		return string(content)
	}
	t.Fatalf("entry %s tidak ditemukan di dalam workbook", name)
	return ""
}

func TestEmployeeCreateRequiresHRAndBuildsLockedAccount(t *testing.T) {
	stub := &employeeReaderStub{}
	service := newEmployeeServiceForTest(stub)
	request := dto.CreateEmployeeRequest{
		NIP:          " EMP-001 ",
		Name:         " Karyawan Uji ",
		Email:        " EMPLOYEE@EXAMPLE.TEST ",
		Gender:       "P",
		BirthDate:    "1995-04-10",
		JoinDate:     "2026-07-29",
		DepartmentID: uuid.New(),
		PositionID:   uuid.New(),
		Role:         domain.RoleEmployee,
		Address: dto.EmployeeAddressRequest{
			Street:   "Jalan Sintetis",
			City:     "Jakarta",
			Province: "DKI Jakarta",
		},
		KTP: dto.EmployeeKTPRequest{Number: "3174000000000001"},
		Contract: dto.EmployeeContractRequest{
			Number:    "PKWT-TEST-001",
			Type:      "PKWT",
			StartDate: "2026-07-29",
			EndDate:   "2027-07-28",
		},
	}

	_, err := service.Create(
		context.Background(),
		domain.Identity{Role: domain.RoleTopManagement},
		request,
		RequestMeta{},
	)
	require.ErrorIs(t, err, domain.ErrForbidden)

	_, err = service.Create(
		context.Background(),
		domain.Identity{UserID: uuid.New(), Role: domain.RoleHR},
		request,
		RequestMeta{},
	)
	require.NoError(t, err)
	assert.Equal(t, "EMP-001", stub.createCommand.NIP)
	assert.Equal(t, "employee@example.test", stub.createCommand.Email)
	assert.Equal(t, "locked-hash", stub.createCommand.PasswordHash)
}

// Detail seperti BPJS/NPWP/kontak darurat bersifat opsional pada create; field yang tidak
// disertakan pada request harus tetap nil pada command sehingga repository tidak menulis
// baris kosong untuk bagian tersebut.
func TestEmployeeCreateMapsOptionalDetailSections(t *testing.T) {
	stub := &employeeReaderStub{}
	service := newEmployeeServiceForTest(stub)
	departmentID := uuid.New()
	positionID := uuid.New()
	request := dto.CreateEmployeeRequest{
		NIP:          "EMP-002",
		Name:         "Karyawan Uji",
		Email:        "employee2@example.test",
		Gender:       "L",
		BirthDate:    "1995-04-10",
		JoinDate:     "2026-07-29",
		DepartmentID: uuid.New(),
		PositionID:   uuid.New(),
		Role:         domain.RoleEmployee,
		Address: dto.EmployeeAddressRequest{
			Street: "Jalan Sintetis", City: "Jakarta", Province: "DKI Jakarta",
		},
		KTP: dto.EmployeeKTPRequest{Number: "3174000000000002"},
		Contract: dto.EmployeeContractRequest{
			Number: "PKWT-TEST-002", Type: "PKWT",
			StartDate: "2026-07-29", EndDate: "2027-07-28",
		},
		BPJS: &dto.EmployeeBPJSRequest{HealthNumber: strPtr(" 000111 ")},
		NPWP: &dto.EmployeeNPWPRequest{Number: " 12.345.678.9-012.000 "},
		EmergencyContacts: []dto.EmergencyContactRequest{
			{Name: " Ibu Uji ", Phone: " 0812xxxx "},
		},
		Education: []dto.EducationRequest{
			{Level: strPtr("S1"), Institution: strPtr("Universitas Uji")},
		},
		PositionHistory: []dto.PositionHistoryRequest{
			{DepartmentID: &departmentID, PositionID: &positionID, StartDate: "2020-01-01"},
		},
		CurrentSalary: &dto.CurrentSalaryRequest{Period: "2026-08", BasePay: 5_000_000},
	}

	_, err := service.Create(
		context.Background(),
		domain.Identity{UserID: uuid.New(), Role: domain.RoleHR},
		request,
		RequestMeta{},
	)
	require.NoError(t, err)

	require.NotNil(t, stub.createCommand.BPJS)
	assert.Equal(t, "000111", *stub.createCommand.BPJS.HealthNumber)
	assert.Nil(t, stub.createCommand.BPJS.EmploymentNumber)
	require.NotNil(t, stub.createCommand.NPWP)
	assert.Equal(t, "12.345.678.9-012.000", stub.createCommand.NPWP.Number)
	require.Len(t, stub.createCommand.EmergencyContacts, 1)
	assert.Equal(t, "Ibu Uji", stub.createCommand.EmergencyContacts[0].Name)
	require.Len(t, stub.createCommand.Education, 1)
	require.Len(t, stub.createCommand.PositionHistory, 1)
	assert.Equal(t, departmentID, *stub.createCommand.PositionHistory[0].DepartmentID)
	require.NotNil(t, stub.createCommand.CurrentSalary)
	assert.Equal(t, "2026-08", stub.createCommand.CurrentSalary.Period)
}

// Bagian yang tidak disertakan pada UpdateEmployeeRequest tidak boleh membuat
// EmployeeChanges menandakan "replace" (pointer harus tetap nil).
func TestEmployeeChangesLeavesOmittedDetailSectionsNil(t *testing.T) {
	changes := employeeChanges(dto.UpdateEmployeeRequest{Name: strPtr("Baru")})
	assert.Nil(t, changes.BPJS)
	assert.Nil(t, changes.NPWP)
	assert.Nil(t, changes.EmergencyContacts)
	assert.Nil(t, changes.Education)
	assert.Nil(t, changes.PositionHistory)
	assert.Nil(t, changes.CurrentSalary)
}

// Array kosong yang disertakan eksplisit berarti "hapus semua baris", bukan "tidak diubah";
// harus tetap non-nil setelah mapping agar repository menjalankan replace-all.
func TestEmployeeChangesTreatsEmptyArrayAsClearAll(t *testing.T) {
	empty := []dto.EmergencyContactRequest{}
	changes := employeeChanges(dto.UpdateEmployeeRequest{EmergencyContacts: &empty})
	require.NotNil(t, changes.EmergencyContacts)
	assert.Len(t, *changes.EmergencyContacts, 0)
}

func strPtr(value string) *string { return &value }
