package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/platform/export"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
)

type EmployeeReader interface {
	ListDepartments(context.Context) ([]domain.Department, error)
	ListPositions(context.Context, *uuid.UUID) ([]domain.Position, error)
	List(context.Context, domain.EmployeeFilter) (domain.EmployeePage, error)
	FindByID(context.Context, uuid.UUID, string) (domain.EmployeeDetail, error)
	ValidateCreate(context.Context, domain.CreateEmployee) error
	Create(context.Context, domain.CreateEmployee) (domain.EmployeeMutationResult, error)
	ValidateMutation(context.Context, uuid.UUID, domain.EmployeeChanges) error
	Update(context.Context, uuid.UUID, domain.EmployeeChanges) (domain.EmployeeMutationResult, error)
	SoftDelete(context.Context, uuid.UUID) (domain.EmployeeMutationResult, error)
	ExistsActive(context.Context, uuid.UUID) error
	FindDocuments(context.Context, uuid.UUID) ([]domain.EmployeeDocument, error)
	ResolveDocumentType(context.Context, string) (uuid.UUID, error)
	UpsertDocument(context.Context, domain.NewEmployeeDocument) (id uuid.UUID, previousFileURL string, replaced bool, err error)
	ExportRows(context.Context, domain.EmployeeExportQuery, int) ([]domain.EmployeeSummary, error)
	UpdatePhoto(context.Context, uuid.UUID, string) error
}

// DocumentStore adalah boundary penyimpanan berkas (Nextcloud WebDAV). Credential teknis
// tidak pernah keluar dari adapter ini.
type DocumentStore interface {
	Upload(ctx context.Context, objectPath string, body io.Reader, contentType string) (string, error)
	Delete(ctx context.Context, objectPath string) error
}

type EmployeeTransactionManager interface {
	Within(context.Context, func(context.Context) error) error
}

type SessionRevoker interface {
	Revoke(context.Context, uuid.UUID) error
}

type EmployeeService struct {
	employees EmployeeReader
	tx        EmployeeTransactionManager
	audit     AuditWriter
	sessions  SessionRevoker
	passwords PasswordHasher
	documents DocumentStore
	now       func() time.Time
}

type BulkEmployeeItem struct {
	Row        int       `json:"baris"`
	EmployeeID uuid.UUID `json:"employee_id"`
	Email      string    `json:"email"`
}
type BulkEmployeeFailure struct {
	Row     int    `json:"baris"`
	Message string `json:"message"`
}
type BulkEmployeeResult struct {
	Created []BulkEmployeeItem    `json:"dibuat"`
	Failed  []BulkEmployeeFailure `json:"gagal"`
}

var emailPartPattern = regexp.MustCompile(`[^a-z0-9]+`)

func (s *EmployeeService) automaticEmployeeEmail(ctx context.Context, name string) (string, error) {
	parts := strings.Fields(strings.ToLower(name))
	if len(parts) == 0 {
		return "", domain.ErrInvalidRequest
	}
	clean := func(value string) string { return strings.Trim(emailPartPattern.ReplaceAllString(value, ""), "-") }
	first := clean(parts[0])
	if first == "" {
		return "", domain.ErrInvalidRequest
	}
	candidates := []string{first}
	if len(parts) > 1 {
		candidates = append(candidates, first+clean(parts[1]))
	}
	checker, ok := s.employees.(interface {
		EmailExists(context.Context, string) (bool, error)
	})
	if !ok {
		return candidates[0] + "@janjikupadamu.id", nil
	}
	for index := 0; index < 1000; index++ {
		local := candidates[len(candidates)-1]
		if index < len(candidates) {
			local = candidates[index]
		} else {
			local += fmt.Sprint(index - len(candidates) + 2)
		}
		email := local + "@janjikupadamu.id"
		exists, err := checker.EmailExists(ctx, email)
		if err != nil {
			return "", err
		}
		if !exists {
			return email, nil
		}
	}
	return "", domain.ErrConflict
}

func (s *EmployeeService) BulkCreate(ctx context.Context, identity domain.Identity, requests []dto.CreateEmployeeRequest, meta RequestMeta) (BulkEmployeeResult, error) {
	if identity.Role != domain.RoleHR {
		return BulkEmployeeResult{}, domain.ErrForbidden
	}
	result := BulkEmployeeResult{Created: make([]BulkEmployeeItem, 0), Failed: make([]BulkEmployeeFailure, 0)}
	for index, request := range requests {
		if strings.TrimSpace(request.Email) == "" {
			email, err := s.automaticEmployeeEmail(ctx, request.Name)
			if err != nil {
				result.Failed = append(result.Failed, BulkEmployeeFailure{Row: index + 2, Message: "Email otomatis tidak dapat dibuat"})
				continue
			}
			request.Email = email
		}
		created, err := s.Create(ctx, identity, request, meta)
		if err != nil {
			result.Failed = append(result.Failed, BulkEmployeeFailure{Row: index + 2, Message: "Data karyawan tidak valid atau duplikat"})
			continue
		}
		result.Created = append(result.Created, BulkEmployeeItem{Row: index + 2, EmployeeID: created.EmployeeID, Email: request.Email})
	}
	return result, nil
}

func NewEmployeeService(
	employees EmployeeReader,
	tx EmployeeTransactionManager,
	audit AuditWriter,
	sessions SessionRevoker,
	passwords PasswordHasher,
	documents DocumentStore,
) *EmployeeService {
	return &EmployeeService{
		employees: employees,
		tx:        tx,
		audit:     audit,
		sessions:  sessions,
		passwords: passwords,
		documents: documents,
		now:       time.Now,
	}
}

func (s *EmployeeService) Create(
	ctx context.Context,
	identity domain.Identity,
	request dto.CreateEmployeeRequest,
	meta RequestMeta,
) (domain.EmployeeMutationResult, error) {
	if identity.Role != domain.RoleHR {
		return domain.EmployeeMutationResult{}, domain.ErrForbidden
	}
	startDate, err := time.Parse("2006-01-02", request.Contract.StartDate)
	if err != nil {
		return domain.EmployeeMutationResult{}, domain.ErrInvalidRequest
	}
	endDate, err := time.Parse("2006-01-02", request.Contract.EndDate)
	if err != nil || endDate.Before(startDate) {
		return domain.EmployeeMutationResult{}, domain.ErrInvalidRequest
	}
	rawCredential, err := randomLockedCredential()
	if err != nil {
		return domain.EmployeeMutationResult{}, fmt.Errorf("generate locked account credential: %w", err)
	}
	passwordHash, err := s.passwords.Hash(rawCredential)
	if err != nil {
		return domain.EmployeeMutationResult{}, fmt.Errorf("hash locked account credential: %w", err)
	}
	command := createEmployeeCommand(request, passwordHash)
	var result domain.EmployeeMutationResult
	err = s.tx.Within(ctx, func(txContext context.Context) error {
		if err := s.employees.ValidateCreate(txContext, command); err != nil {
			return mapEmployeeRepositoryError(err)
		}
		created, err := s.employees.Create(txContext, command)
		if err != nil {
			return mapEmployeeRepositoryError(err)
		}
		result = created
		return s.audit.Append(txContext, domain.AuditEntry{
			UserID: &identity.UserID,
			Action: "CREATE",
			Module: "karyawan",
			DataID: &created.EmployeeID,
			Detail: map[string]any{
				"department_id": command.DepartmentID,
				"position_id":   command.PositionID,
				"role":          command.Role,
				"account_state": "locked_pending_activation",
				"request_id":    meta.RequestID,
			},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		})
	})
	if err != nil {
		return domain.EmployeeMutationResult{}, fmt.Errorf("create employee: %w", err)
	}
	return result, nil
}

func createEmployeeCommand(
	request dto.CreateEmployeeRequest,
	passwordHash string,
) domain.CreateEmployee {
	return domain.CreateEmployee{
		NIP:           strings.TrimSpace(request.NIP),
		Name:          strings.TrimSpace(request.Name),
		Email:         strings.ToLower(strings.TrimSpace(request.Email)),
		PasswordHash:  passwordHash,
		Gender:        request.Gender,
		BirthDate:     request.BirthDate,
		JoinDate:      request.JoinDate,
		DepartmentID:  request.DepartmentID,
		PositionID:    request.PositionID,
		SupervisorID:  request.SupervisorID,
		MaritalStatus: request.MaritalStatus,
		Role:          request.Role,
		Address: domain.EmployeeAddress{
			Street:   strings.TrimSpace(request.Address.Street),
			Village:  trimOptionalString(request.Address.Village),
			District: trimOptionalString(request.Address.District),
			City:     strings.TrimSpace(request.Address.City),
			Province: strings.TrimSpace(request.Address.Province),
		},
		KTPNumber: strings.TrimSpace(request.KTP.Number),
		Contract: domain.CreateEmployeeContract{
			Number:    strings.TrimSpace(request.Contract.Number),
			Type:      request.Contract.Type,
			StartDate: request.Contract.StartDate,
			EndDate:   request.Contract.EndDate,
		},
		BPJS:              mapBPJSRequest(request.BPJS),
		NPWP:              mapNPWPRequest(request.NPWP),
		EmergencyContacts: mapEmergencyContacts(request.EmergencyContacts),
		Education:         mapEducation(request.Education),
		PositionHistory:   mapPositionHistory(request.PositionHistory),
		CurrentSalary:     mapCurrentSalary(request.CurrentSalary),
	}
}

func mapBPJSRequest(request *dto.EmployeeBPJSRequest) *domain.CreateEmployeeBPJS {
	if request == nil {
		return nil
	}
	return &domain.CreateEmployeeBPJS{
		HealthNumber:     trimOptionalString(request.HealthNumber),
		EmploymentNumber: trimOptionalString(request.EmploymentNumber),
	}
}

func mapNPWPRequest(request *dto.EmployeeNPWPRequest) *domain.CreateEmployeeNPWP {
	if request == nil {
		return nil
	}
	return &domain.CreateEmployeeNPWP{Number: strings.TrimSpace(request.Number)}
}

func mapEmergencyContacts(requests []dto.EmergencyContactRequest) []domain.CreateEmergencyContact {
	if requests == nil {
		return nil
	}
	contacts := make([]domain.CreateEmergencyContact, 0, len(requests))
	for _, item := range requests {
		contacts = append(contacts, domain.CreateEmergencyContact{
			Name:         strings.TrimSpace(item.Name),
			Relationship: trimOptionalString(item.Relationship),
			Phone:        strings.TrimSpace(item.Phone),
		})
	}
	return contacts
}

func mapEducation(requests []dto.EducationRequest) []domain.CreateEducation {
	if requests == nil {
		return nil
	}
	entries := make([]domain.CreateEducation, 0, len(requests))
	for _, item := range requests {
		entries = append(entries, domain.CreateEducation{
			Level:          trimOptionalString(item.Level),
			Institution:    trimOptionalString(item.Institution),
			EntryYear:      item.EntryYear,
			GraduationYear: item.GraduationYear,
		})
	}
	return entries
}

func mapPositionHistory(requests []dto.PositionHistoryRequest) []domain.CreatePositionHistory {
	if requests == nil {
		return nil
	}
	entries := make([]domain.CreatePositionHistory, 0, len(requests))
	for _, item := range requests {
		entries = append(entries, domain.CreatePositionHistory{
			DepartmentID: item.DepartmentID,
			PositionID:   item.PositionID,
			StartDate:    item.StartDate,
			EndDate:      item.EndDate,
		})
	}
	return entries
}

func mapCurrentSalary(request *dto.CurrentSalaryRequest) *domain.CreateCurrentSalary {
	if request == nil {
		return nil
	}
	return &domain.CreateCurrentSalary{
		Period:    request.Period,
		BasePay:   request.BasePay,
		Allowance: request.Allowance,
		Deduction: request.Deduction,
	}
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func randomLockedCredential() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func (s *EmployeeService) Update(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
	request dto.UpdateEmployeeRequest,
	meta RequestMeta,
) (domain.EmployeeMutationResult, error) {
	if identity.Role != domain.RoleHR {
		return domain.EmployeeMutationResult{}, domain.ErrForbidden
	}
	if request.Empty() {
		return domain.EmployeeMutationResult{}, domain.ErrInvalidRequest
	}
	changes := employeeChanges(request)
	var result domain.EmployeeMutationResult
	err := s.tx.Within(ctx, func(txContext context.Context) error {
		if err := s.employees.ValidateMutation(txContext, id, changes); err != nil {
			return mapEmployeeRepositoryError(err)
		}
		updated, err := s.employees.Update(txContext, id, changes)
		if err != nil {
			return mapEmployeeRepositoryError(err)
		}
		result = updated
		if requiresSessionRevocation(changes) && updated.UserID != nil {
			if err := s.sessions.Revoke(txContext, *updated.UserID); err != nil {
				return fmt.Errorf("revoke employee session: %w", err)
			}
		}
		return s.audit.Append(txContext, domain.AuditEntry{
			UserID:    &identity.UserID,
			Action:    "UPDATE",
			Module:    "karyawan",
			DataID:    &id,
			Detail:    map[string]any{"fields": changedEmployeeFields(changes), "request_id": meta.RequestID},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		})
	})
	if err != nil {
		return domain.EmployeeMutationResult{}, fmt.Errorf("update employee: %w", err)
	}
	return result, nil
}

func (s *EmployeeService) Deactivate(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
	meta RequestMeta,
) (domain.EmployeeMutationResult, error) {
	if identity.Role != domain.RoleHR {
		return domain.EmployeeMutationResult{}, domain.ErrForbidden
	}
	var result domain.EmployeeMutationResult
	err := s.tx.Within(ctx, func(txContext context.Context) error {
		deactivated, err := s.employees.SoftDelete(txContext, id)
		if err != nil {
			return mapEmployeeRepositoryError(err)
		}
		result = deactivated
		if deactivated.UserID != nil {
			if err := s.sessions.Revoke(txContext, *deactivated.UserID); err != nil {
				return fmt.Errorf("revoke deactivated employee session: %w", err)
			}
		}
		return s.audit.Append(txContext, domain.AuditEntry{
			UserID:    &identity.UserID,
			Action:    "DELETE",
			Module:    "karyawan",
			DataID:    &id,
			Detail:    map[string]any{"status": "nonaktif", "request_id": meta.RequestID},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		})
	})
	if err != nil {
		return domain.EmployeeMutationResult{}, fmt.Errorf("deactivate employee: %w", err)
	}
	return result, nil
}

func employeeChanges(request dto.UpdateEmployeeRequest) domain.EmployeeChanges {
	changes := domain.EmployeeChanges{
		Name:          request.Name,
		Email:         request.Email,
		Gender:        request.Gender,
		BirthDate:     request.BirthDate,
		JoinDate:      request.JoinDate,
		DepartmentID:  request.DepartmentID,
		PositionID:    request.PositionID,
		SupervisorID:  request.SupervisorID.Value,
		SupervisorSet: request.SupervisorID.Set,
		MaritalStatus: request.MaritalStatus,
		Status:        request.Status,
		Role:          request.Role,
		BPJS:          mapBPJSRequest(request.BPJS),
		NPWP:          mapNPWPRequest(request.NPWP),
		CurrentSalary: mapCurrentSalary(request.CurrentSalary),
	}
	if changes.Name != nil {
		normalized := strings.TrimSpace(*changes.Name)
		changes.Name = &normalized
	}
	if changes.Email != nil {
		normalized := strings.ToLower(strings.TrimSpace(*changes.Email))
		changes.Email = &normalized
	}
	if request.EmergencyContacts != nil {
		contacts := mapEmergencyContacts(*request.EmergencyContacts)
		changes.EmergencyContacts = &contacts
	}
	if request.Education != nil {
		education := mapEducation(*request.Education)
		changes.Education = &education
	}
	if request.PositionHistory != nil {
		history := mapPositionHistory(*request.PositionHistory)
		changes.PositionHistory = &history
	}
	return changes
}

func mapEmployeeRepositoryError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return domain.ErrNotFound
	case errors.Is(err, repository.ErrConflict):
		return domain.ErrConflict
	default:
		return err
	}
}

func requiresSessionRevocation(changes domain.EmployeeChanges) bool {
	return changes.Email != nil || changes.Role != nil || changes.Status != nil
}

func changedEmployeeFields(changes domain.EmployeeChanges) []string {
	fields := make([]string, 0, 11)
	candidates := []struct {
		name    string
		changed bool
	}{
		{"nama", changes.Name != nil},
		{"email", changes.Email != nil},
		{"jenis_kelamin", changes.Gender != nil},
		{"tanggal_lahir", changes.BirthDate != nil},
		{"tanggal_join", changes.JoinDate != nil},
		{"department_id", changes.DepartmentID != nil},
		{"position_id", changes.PositionID != nil},
		{"atasan_id", changes.SupervisorSet},
		{"status_pernikahan", changes.MaritalStatus != nil},
		{"status", changes.Status != nil},
		{"role", changes.Role != nil},
	}
	for _, candidate := range candidates {
		if candidate.changed {
			fields = append(fields, candidate.name)
		}
	}
	return fields
}

func (s *EmployeeService) ListDepartments(ctx context.Context) ([]domain.Department, error) {
	items, err := s.employees.ListDepartments(ctx)
	if err != nil {
		return nil, fmt.Errorf("list department master: %w", err)
	}
	return items, nil
}

func (s *EmployeeService) ListPositions(
	ctx context.Context,
	departmentID *uuid.UUID,
) ([]domain.Position, error) {
	items, err := s.employees.ListPositions(ctx, departmentID)
	if err != nil {
		return nil, fmt.Errorf("list position master: %w", err)
	}
	return items, nil
}

func (s *EmployeeService) List(
	ctx context.Context,
	identity domain.Identity,
	filter domain.EmployeeFilter,
) (domain.EmployeePage, error) {
	if identity.Role != domain.RoleHR {
		return domain.EmployeePage{}, domain.ErrForbidden
	}
	filter.Search = strings.TrimSpace(filter.Search)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Status != "" && filter.Status != "aktif" && filter.Status != "nonaktif" {
		return domain.EmployeePage{}, domain.ErrInvalidRequest
	}
	page, err := s.employees.List(ctx, filter)
	if err != nil {
		return domain.EmployeePage{}, fmt.Errorf("list employee database: %w", err)
	}
	return page, nil
}

func (s *EmployeeService) Detail(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
) (domain.EmployeeDetail, error) {
	if identity.Role != domain.RoleHR {
		return domain.EmployeeDetail{}, domain.ErrForbidden
	}
	// Gaji dibatasi ke periode bulan berjalan sesuai PRD; histori gaji tidak pernah dibaca.
	item, err := s.employees.FindByID(ctx, id, domain.CurrentSalaryPeriod(s.now()))
	if errors.Is(err, repository.ErrNotFound) {
		return domain.EmployeeDetail{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.EmployeeDetail{}, fmt.Errorf("get employee detail: %w", err)
	}
	return item, nil
}

// maxExportRows membatasi dataset export agar permintaan tidak menarik tabel tanpa batas.
const maxExportRows = 5000

// maxDocumentBytes adalah batas 5 MB per berkas sesuai kontrak dan PRD.
const maxDocumentBytes = 5 << 20

// ListDocuments mengembalikan metadata dokumen karyawan. HR membaca penuh dan Top Management
// read-only; Karyawan/Atasan tidak memiliki akses Point 12.
func (s *EmployeeService) ListDocuments(
	ctx context.Context,
	identity domain.Identity,
	employeeID uuid.UUID,
) ([]domain.EmployeeDocument, error) {
	if identity.Role != domain.RoleHR {
		return nil, domain.ErrForbidden
	}
	if err := s.employees.ExistsActive(ctx, employeeID); err != nil {
		return nil, mapEmployeeRepositoryError(err)
	}
	items, err := s.employees.FindDocuments(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("list employee documents: %w", err)
	}
	return items, nil
}

// DocumentUpload adalah berkas yang sudah divalidasi handler dan siap disimpan.
type DocumentUpload struct {
	Type      string
	FileName  string
	Extension string
	MediaType string
	Content   []byte
}

// UploadDocument menyimpan dokumen melalui backend ke Nextcloud lalu mencatat locator-nya.
// Nama object dibentuk server-side dan tidak pernah menerima path dari client. Setiap
// karyawan hanya boleh memiliki satu dokumen per jenis dokumen master (defect: dokumen
// dobel); upload berikutnya untuk jenis yang sama menggantikan dokumen lama, termasuk
// menghapus berkas lama dari Nextcloud setelah baris baru berhasil dicatat. Bila transaction
// database gagal setelah upload, object yang baru terunggah dihapus kembali.
func (s *EmployeeService) UploadDocument(
	ctx context.Context,
	identity domain.Identity,
	employeeID uuid.UUID,
	upload DocumentUpload,
	meta RequestMeta,
) (domain.EmployeeDocument, error) {
	if identity.Role != domain.RoleHR {
		return domain.EmployeeDocument{}, domain.ErrForbidden
	}
	if len(upload.Content) == 0 || len(upload.Content) > maxDocumentBytes {
		return domain.EmployeeDocument{}, domain.ErrInvalidRequest
	}
	documentTypeID, err := s.employees.ResolveDocumentType(ctx, upload.Type)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.EmployeeDocument{}, domain.ErrInvalidRequest
	}
	if err != nil {
		return domain.EmployeeDocument{}, fmt.Errorf("resolve document type: %w", err)
	}
	if err := s.employees.ExistsActive(ctx, employeeID); err != nil {
		return domain.EmployeeDocument{}, mapEmployeeRepositoryError(err)
	}

	objectPath := documentObjectPath(employeeID, upload.Extension)
	location, err := s.documents.Upload(
		ctx,
		objectPath,
		bytes.NewReader(upload.Content),
		upload.MediaType,
	)
	if err != nil {
		return domain.EmployeeDocument{}, fmt.Errorf("upload employee document: %w", err)
	}
	// rootFolderPrefix menyusutkan file_url dokumen lama (juga berbentuk rootFolder + "/" +
	// objectPath) kembali menjadi objectPath relatif yang dipahami DocumentStore.Delete.
	rootFolderPrefix := strings.TrimSuffix(location, objectPath)

	var documentID uuid.UUID
	var previousFileURL string
	var replaced bool
	err = s.tx.Within(ctx, func(txContext context.Context) error {
		id, previous, isReplace, err := s.employees.UpsertDocument(txContext, domain.NewEmployeeDocument{
			EmployeeID:     employeeID,
			DocumentTypeID: documentTypeID,
			Type:           upload.Type,
			FileName:       upload.FileName,
			FileURL:        location,
		})
		if err != nil {
			return mapEmployeeRepositoryError(err)
		}
		documentID, previousFileURL, replaced = id, previous, isReplace
		return s.audit.Append(txContext, domain.AuditEntry{
			UserID: &identity.UserID,
			Action: map[bool]string{true: "UPDATE", false: "CREATE"}[replaced],
			Module: "karyawan_dokumen",
			DataID: &id,
			Detail: map[string]any{
				"employee_id":   employeeID,
				"jenis_dokumen": upload.Type,
				"ukuran_byte":   len(upload.Content),
				"request_id":    meta.RequestID,
			},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		})
	})
	if err != nil {
		// Kompensasi object yang sudah terunggah agar tidak menjadi orphan.
		if cleanupErr := s.documents.Delete(ctx, objectPath); cleanupErr != nil {
			slog.ErrorContext(ctx, "orphan document cleanup failed",
				"object_path", objectPath, "error", cleanupErr)
		}
		return domain.EmployeeDocument{}, fmt.Errorf("record employee document: %w", err)
	}
	if replaced && previousFileURL != "" {
		oldObjectPath := strings.TrimPrefix(previousFileURL, rootFolderPrefix)
		if cleanupErr := s.documents.Delete(ctx, oldObjectPath); cleanupErr != nil {
			slog.ErrorContext(ctx, "previous document cleanup failed",
				"object_path", oldObjectPath, "error", cleanupErr)
		}
	}
	return domain.EmployeeDocument{
		ID:        documentID,
		Type:      upload.Type,
		FileName:  upload.FileName,
		FileURL:   location,
		CreatedAt: s.now().UTC(),
	}, nil
}

// documentObjectPath membentuk path yang ter-namespace per karyawan. Nama berkas asli tidak
// pernah dipakai sebagai nama object agar path traversal dan tabrakan nama tidak mungkin.
func documentObjectPath(employeeID uuid.UUID, extension string) string {
	return path.Join("employee-documents", employeeID.String(), uuid.NewString()+extension)
}

// PhotoUpload adalah foto profil yang sudah divalidasi handler dan siap disimpan.
type PhotoUpload struct {
	Extension string
	MediaType string
	Content   []byte
}

// UpdatePhoto mengganti foto profil karyawan. HR boleh memperbarui foto karyawan mana pun;
// karyawan/atasan/HR sendiri hanya boleh memperbarui foto miliknya sendiri (D-037). Foto
// lama tidak dihapus dari Nextcloud, konsisten dengan dokumen karyawan lain yang juga tidak
// pernah dibersihkan otomatis di luar retensi foto absensi.
func (s *EmployeeService) UpdatePhoto(
	ctx context.Context,
	identity domain.Identity,
	employeeID uuid.UUID,
	upload PhotoUpload,
	meta RequestMeta,
) (string, error) {
	if identity.Role != domain.RoleHR && identity.EmployeeID != employeeID {
		return "", domain.ErrForbidden
	}
	if len(upload.Content) == 0 || len(upload.Content) > maxDocumentBytes {
		return "", domain.ErrInvalidRequest
	}
	if err := s.employees.ExistsActive(ctx, employeeID); err != nil {
		return "", mapEmployeeRepositoryError(err)
	}

	objectPath := profilePhotoObjectPath(employeeID, upload.Extension)
	location, err := s.documents.Upload(ctx, objectPath, bytes.NewReader(upload.Content), upload.MediaType)
	if err != nil {
		return "", fmt.Errorf("upload profile photo: %w", err)
	}

	err = s.tx.Within(ctx, func(txContext context.Context) error {
		if err := s.employees.UpdatePhoto(txContext, employeeID, location); err != nil {
			return mapEmployeeRepositoryError(err)
		}
		return s.audit.Append(txContext, domain.AuditEntry{
			UserID:    &identity.UserID,
			Action:    "UPDATE",
			Module:    "karyawan_foto",
			DataID:    &employeeID,
			Detail:    map[string]any{"request_id": meta.RequestID},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		})
	})
	if err != nil {
		// Kompensasi object yang sudah terunggah agar tidak menjadi orphan.
		if cleanupErr := s.documents.Delete(ctx, objectPath); cleanupErr != nil {
			slog.ErrorContext(ctx, "orphan profile photo cleanup failed",
				"object_path", objectPath, "error", cleanupErr)
		}
		return "", fmt.Errorf("record employee photo: %w", err)
	}
	return location, nil
}

// profilePhotoObjectPath membentuk path yang ter-namespace per karyawan, terpisah dari
// dokumen umum agar retensi/kebijakan berbeda dapat diterapkan di kemudian hari.
func profilePhotoObjectPath(employeeID uuid.UUID, extension string) string {
	return path.Join("employee-photos", employeeID.String(), uuid.NewString()+extension)
}

// Export menghasilkan berkas XLSX atau PDF berisi dataset yang sama dengan list. Hanya HR
// yang diizinkan dan setiap unduhan dicatat pada Audit Log.
func (s *EmployeeService) Export(
	ctx context.Context,
	identity domain.Identity,
	query domain.EmployeeExportQuery,
	meta RequestMeta,
) (domain.ExportFile, error) {
	if identity.Role != domain.RoleHR {
		return domain.ExportFile{}, domain.ErrForbidden
	}
	if !query.Format.Valid() {
		return domain.ExportFile{}, domain.ErrInvalidRequest
	}
	if status := query.Filter.Status; status != "" && status != "aktif" && status != "nonaktif" {
		return domain.ExportFile{}, domain.ErrInvalidRequest
	}
	rows, err := s.employees.ExportRows(ctx, query, maxExportRows)
	if err != nil {
		return domain.ExportFile{}, fmt.Errorf("read employee export dataset: %w", err)
	}
	if len(rows) == 0 {
		return domain.ExportFile{}, domain.ErrNotFound
	}

	table := employeeExportTable(rows, s.now())
	var buffer bytes.Buffer
	contentType := export.XLSXContentType
	if query.Format == domain.ExportFormatPDF {
		contentType = export.PDFContentType
		err = export.WritePDF(&buffer, table)
	} else {
		err = export.WriteXLSX(&buffer, table)
	}
	if err != nil {
		return domain.ExportFile{}, fmt.Errorf("render employee export: %w", err)
	}

	file := domain.ExportFile{
		FileName: export.SanitizeFileName(fmt.Sprintf(
			"karyawan-%s.%s", s.now().In(domain.Jakarta()).Format("20060102-150405"), query.Format,
		)),
		ContentType: contentType,
		Content:     buffer.Bytes(),
	}
	if err := s.audit.Append(ctx, domain.AuditEntry{
		UserID: &identity.UserID,
		Action: "DOWNLOAD",
		Module: "karyawan",
		DataID: query.EmployeeID,
		Detail: map[string]any{
			"format":     string(query.Format),
			"jumlah_row": len(rows),
			"request_id": meta.RequestID,
		},
		IPAddress: meta.IPAddress,
		CreatedAt: s.now().UTC(),
	}); err != nil {
		return domain.ExportFile{}, fmt.Errorf("audit employee export: %w", err)
	}
	return file, nil
}

func employeeExportTable(rows []domain.EmployeeSummary, generatedAt time.Time) export.Table {
	table := export.Table{
		Title: fmt.Sprintf(
			"GSNpeeps - Data Karyawan (%s WIB)",
			generatedAt.In(domain.Jakarta()).Format("2006-01-02 15:04"),
		),
		Headers: []string{"NIP", "Nama", "Email", "Departemen", "Jabatan", "Status"},
		Rows:    make([][]string, 0, len(rows)),
	}
	for _, row := range rows {
		table.Rows = append(table.Rows, []string{
			row.NIP, row.Name, row.Email, row.Department, row.Position, row.Status,
		})
	}
	return table
}
