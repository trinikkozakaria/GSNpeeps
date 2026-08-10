package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type leaveStoreStub struct {
	leaveType    domain.LeaveType
	leaveTypeErr error
	createdRow   domain.LeaveRequestRow
	createErr    error
	lock         domain.RequestLock
	lockErr      error
	statusFrom   domain.RequestStatus
	statusTo     domain.RequestStatus
	statusErr    error
	appended     []appendedApproval
	deducted     int
	deductCalls  int
	deductErr    error
	filter       domain.LeaveRequestFilter
	detail       domain.LeaveRequestDetail
	detailErr    error
	listActive   *bool
}

type appendedApproval struct {
	stage      domain.ApprovalStage
	approverID *uuid.UUID
	decision   domain.ApprovalDecision
	note       *string
}

func (s *leaveStoreStub) ListLeaveTypes(
	_ context.Context, activeOnly *bool,
) ([]domain.LeaveType, error) {
	s.listActive = activeOnly
	return []domain.LeaveType{s.leaveType}, nil
}

func (s *leaveStoreStub) FindLeaveType(context.Context, uuid.UUID) (domain.LeaveType, error) {
	return s.leaveType, s.leaveTypeErr
}

func (s *leaveStoreStub) CreateLeaveType(
	context.Context, domain.CreateLeaveType,
) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (s *leaveStoreStub) UpdateLeaveType(
	context.Context, uuid.UUID, domain.UpdateLeaveType,
) error {
	return nil
}

func (s *leaveStoreStub) CreateRequest(
	_ context.Context, row domain.LeaveRequestRow,
) (uuid.UUID, error) {
	s.createdRow = row
	if s.createErr != nil {
		return uuid.Nil, s.createErr
	}
	return uuid.New(), nil
}

func (s *leaveStoreStub) ListRequests(
	_ context.Context, filter domain.LeaveRequestFilter,
) (domain.LeaveRequestPage, error) {
	s.filter = filter
	return domain.LeaveRequestPage{Page: filter.Page, Limit: filter.Limit}, nil
}

func (s *leaveStoreStub) FindRequest(
	context.Context, uuid.UUID,
) (domain.LeaveRequestDetail, error) {
	return s.detail, s.detailErr
}

func (s *leaveStoreStub) LockRequestForDecision(
	context.Context, uuid.UUID,
) (domain.RequestLock, error) {
	return s.lock, s.lockErr
}

func (s *leaveStoreStub) UpdateRequestStatus(
	_ context.Context, _ uuid.UUID, from, to domain.RequestStatus,
) error {
	s.statusFrom, s.statusTo = from, to
	return s.statusErr
}

func (s *leaveStoreStub) AppendApproval(
	_ context.Context, _ uuid.UUID, stage domain.ApprovalStage, approverID *uuid.UUID,
	decision domain.ApprovalDecision, note *string,
) error {
	s.appended = append(s.appended, appendedApproval{stage, approverID, decision, note})
	return nil
}

func (s *leaveStoreStub) DeductLeaveBalance(
	_ context.Context, _ uuid.UUID, _ int, days int,
) error {
	s.deductCalls++
	s.deducted = days
	return s.deductErr
}

type supervisorStub struct{ supervisor *uuid.UUID }

func (s supervisorStub) SupervisorEmployeeID(
	context.Context, uuid.UUID,
) (*uuid.UUID, error) {
	return s.supervisor, nil
}

type eventRecorder struct{ events []domain.ApprovalEvent }

func (r *eventRecorder) Publish(_ context.Context, events ...domain.ApprovalEvent) error {
	r.events = append(r.events, events...)
	return nil
}

func newLeaveServiceForTest(
	store LeaveStore, supervisor SupervisorLookup, transactions EmployeeTransactionManager,
) (*LeaveService, *eventRecorder, *documentStoreStub) {
	recorder := &eventRecorder{}
	documents := &documentStoreStub{}
	service := NewLeaveService(store, supervisor, transactions, auditStub{}, documents, recorder)
	service.now = func() time.Time {
		return time.Date(2026, time.August, 3, 9, 0, 0, 0, domain.Jakarta())
	}
	return service, recorder, documents
}

func activeLeaveType(requiresDocument bool, quota int) domain.LeaveType {
	return domain.LeaveType{
		ID: uuid.New(), Code: "UJI", Name: "Uji Cuti",
		AnnualQuota: quota, RequiresDocument: requiresDocument, IsActive: true,
	}
}

func leaveCommand(typeID uuid.UUID, document *domain.UploadedFile) domain.CreateLeaveRequest {
	return domain.CreateLeaveRequest{
		LeaveTypeID: typeID,
		StartDate:   "2026-08-10",
		EndDate:     "2026-08-12",
		Reason:      "Keperluan keluarga sintetis",
		Document:    document,
	}
}

func syntheticDocument() *domain.UploadedFile {
	return &domain.UploadedFile{
		FileName: "surat.pdf", Extension: ".pdf", MediaType: "application/pdf",
		Content: []byte("%PDF-1.4 sintetis"),
	}
}

// Empat jalur routing pemohon menghasilkan status awal yang benar.
func TestLeaveCreateRoutesByRequesterRole(t *testing.T) {
	supervisorID := uuid.New()
	cases := []struct {
		name       string
		role       domain.RoleName
		supervisor *uuid.UUID
		expected   domain.RequestStatus
	}{
		{"karyawan dengan atasan", domain.RoleEmployee, &supervisorID, domain.StatusWaitingSupervisor},
		{"karyawan tanpa atasan", domain.RoleEmployee, nil, domain.StatusWaitingHR},
		{"atasan", domain.RoleSupervisor, nil, domain.StatusWaitingHR},
		{"hr", domain.RoleHR, nil, domain.StatusWaitingTopManagement},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			store := &leaveStoreStub{leaveType: activeLeaveType(false, 12)}
			service, events, _ := newLeaveServiceForTest(
				store, supervisorStub{supervisor: testCase.supervisor}, transactionStub{},
			)

			result, err := service.Create(
				context.Background(),
				domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: testCase.role},
				leaveCommand(store.leaveType.ID, nil), RequestMeta{},
			)

			require.NoError(t, err)
			assert.Equal(t, testCase.expected, result.Status)
			assert.Equal(t, testCase.expected, store.createdRow.Status)
			assert.Equal(t, 3, store.createdRow.TotalDays, "rentang inklusif tiga hari")
			require.Len(t, events.events, 1)
			assert.Equal(t, domain.EventLeaveSubmitted, events.events[0].Type)
		})
	}
}

func TestLeaveCreateRejectsTopManagement(t *testing.T) {
	store := &leaveStoreStub{leaveType: activeLeaveType(false, 12)}
	service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.Create(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleTopManagement},
		leaveCommand(store.leaveType.ID, nil), RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrForbidden)
}

// Kewajiban dokumen ditentukan master jenis izin (D-024).
func TestLeaveCreateEnforcesDocumentRequirementFromLeaveType(t *testing.T) {
	store := &leaveStoreStub{leaveType: activeLeaveType(true, 12)}
	service, _, documents := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})
	identity := domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee}

	_, err := service.Create(
		context.Background(), identity, leaveCommand(store.leaveType.ID, nil), RequestMeta{},
	)
	require.ErrorIs(t, err, domain.ErrDocumentRequired)
	assert.Empty(t, documents.uploadedPath)

	_, err = service.Create(
		context.Background(), identity,
		leaveCommand(store.leaveType.ID, syntheticDocument()), RequestMeta{},
	)
	require.NoError(t, err)
	assert.NotEmpty(t, documents.uploadedPath)
	require.NotNil(t, store.createdRow.DocumentURL)
}

// Jenis izin tanpa kewajiban dokumen tetap menerima pengajuan tanpa berkas.
func TestLeaveCreateAllowsMissingDocumentWhenNotRequired(t *testing.T) {
	store := &leaveStoreStub{leaveType: activeLeaveType(false, 12)}
	service, _, documents := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.Create(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee},
		leaveCommand(store.leaveType.ID, nil), RequestMeta{},
	)

	require.NoError(t, err)
	assert.Empty(t, documents.uploadedPath)
	assert.Nil(t, store.createdRow.DocumentURL)
}

func TestLeaveCreateRejectsInactiveTypeAndInvalidRange(t *testing.T) {
	inactive := activeLeaveType(false, 12)
	inactive.IsActive = false
	store := &leaveStoreStub{leaveType: inactive}
	service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})
	identity := domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee}

	_, err := service.Create(
		context.Background(), identity, leaveCommand(inactive.ID, nil), RequestMeta{},
	)
	require.ErrorIs(t, err, domain.ErrInvalidRequest)

	store.leaveType = activeLeaveType(false, 12)
	reversed := leaveCommand(store.leaveType.ID, nil)
	reversed.StartDate, reversed.EndDate = "2026-08-12", "2026-08-10"
	_, err = service.Create(context.Background(), identity, reversed, RequestMeta{})
	require.ErrorIs(t, err, domain.ErrInvalidRequest)
}

// Pengajuan melebihi kuota tahunan ditolak sebelum masuk alur approval.
func TestLeaveCreateRejectsRequestBeyondAnnualQuota(t *testing.T) {
	store := &leaveStoreStub{leaveType: activeLeaveType(false, 2)}
	service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.Create(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee},
		leaveCommand(store.leaveType.ID, nil), RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrInsufficientLeaveBalance)
}

// Dokumen yang sudah terunggah dibersihkan ketika transaction gagal.
func TestLeaveCreateRemovesOrphanDocumentWhenTransactionFails(t *testing.T) {
	store := &leaveStoreStub{leaveType: activeLeaveType(true, 12)}
	service, _, documents := newLeaveServiceForTest(
		store, supervisorStub{}, failingTransactionStub{err: errors.New("database unavailable")},
	)

	_, err := service.Create(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee},
		leaveCommand(store.leaveType.ID, syntheticDocument()), RequestMeta{},
	)

	require.Error(t, err)
	assert.NotEmpty(t, documents.uploadedPath)
	assert.Equal(t, documents.uploadedPath, documents.deletedPath)
}

func decisionLock(status domain.RequestStatus, supervisor *uuid.UUID, quota int) domain.RequestLock {
	return domain.RequestLock{
		RequestID: uuid.New(), RequesterUserID: uuid.New(), RequesterEmployeeID: uuid.New(),
		SupervisorEmployeeID: supervisor, Status: status, TotalDays: 3,
		AnnualQuota: quota, Year: 2026,
	}
}

// Penolakan wajib memiliki catatan.
func TestLeaveDecideRequiresNoteOnReject(t *testing.T) {
	store := &leaveStoreStub{lock: decisionLock(domain.StatusWaitingHR, nil, 12)}
	service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.Decide(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR},
		uuid.New(), domain.DecisionInput{Approve: false}, RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrInvalidRequest)
	assert.Empty(t, store.appended, "keputusan tidak sah tidak menulis riwayat")
}

// Persetujuan Atasan memindahkan ke HR dan belum mengurangi saldo.
func TestLeaveDecideSupervisorApprovalMovesToHRWithoutDeduction(t *testing.T) {
	supervisorEmployee := uuid.New()
	store := &leaveStoreStub{lock: decisionLock(
		domain.StatusWaitingSupervisor, &supervisorEmployee, 12,
	)}
	service, events, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})

	result, err := service.Decide(
		context.Background(),
		domain.Identity{
			UserID: uuid.New(), EmployeeID: supervisorEmployee, Role: domain.RoleSupervisor,
		},
		uuid.New(), domain.DecisionInput{Approve: true}, RequestMeta{},
	)

	require.NoError(t, err)
	assert.Equal(t, domain.StatusWaitingHR, result.Status)
	assert.Equal(t, 0, store.deductCalls, "saldo belum berkurang pada tahap atasan")
	require.Len(t, store.appended, 1)
	assert.Equal(t, domain.StageSupervisor, store.appended[0].stage)
	assert.Equal(t, domain.DecisionApprove, store.appended[0].decision)
	require.Len(t, events.events, 1)
	assert.Equal(t, domain.EventLeaveDecisionChanged, events.events[0].Type)
}

// Final approval mengurangi saldo tepat satu kali.
func TestLeaveDecideFinalApprovalDeductsBalanceOnce(t *testing.T) {
	store := &leaveStoreStub{lock: decisionLock(domain.StatusWaitingHR, nil, 12)}
	service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})

	result, err := service.Decide(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR},
		uuid.New(), domain.DecisionInput{Approve: true}, RequestMeta{},
	)

	require.NoError(t, err)
	assert.Equal(t, domain.StatusApproved, result.Status)
	assert.Equal(t, 1, store.deductCalls)
	assert.Equal(t, 3, store.deducted)
}

// Penolakan tidak mengurangi saldo.
func TestLeaveDecideRejectDoesNotDeductBalance(t *testing.T) {
	note := "Tidak dapat disetujui karena beban kerja"
	store := &leaveStoreStub{lock: decisionLock(domain.StatusWaitingHR, nil, 12)}
	service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})

	result, err := service.Decide(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR},
		uuid.New(), domain.DecisionInput{Approve: false, Note: &note}, RequestMeta{},
	)

	require.NoError(t, err)
	assert.Equal(t, domain.StatusRejected, result.Status)
	assert.Equal(t, 0, store.deductCalls)
}

// Saldo tidak cukup pada final approval menghasilkan error bisnis, bukan saldo negatif.
func TestLeaveDecideSurfacesInsufficientBalance(t *testing.T) {
	store := &leaveStoreStub{
		lock:      decisionLock(domain.StatusWaitingHR, nil, 12),
		deductErr: repository.ErrConflict,
	}
	service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.Decide(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR},
		uuid.New(), domain.DecisionInput{Approve: true}, RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrInsufficientLeaveBalance)
}

// Pengajuan yang sudah final menghasilkan ALREADY_DECIDED.
func TestLeaveDecideRejectsAlreadyDecidedRequest(t *testing.T) {
	store := &leaveStoreStub{lock: decisionLock(domain.StatusApproved, nil, 12)}
	service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.Decide(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR},
		uuid.New(), domain.DecisionInput{Approve: true}, RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrAlreadyDecided)
}

// Keputusan yang kalah bersaing menerima ALREADY_DECIDED, bukan menimpa status.
func TestLeaveDecideMapsConcurrentUpdateToAlreadyDecided(t *testing.T) {
	store := &leaveStoreStub{
		lock:      decisionLock(domain.StatusWaitingHR, nil, 12),
		statusErr: repository.ErrConflict,
	}
	service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.Decide(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR},
		uuid.New(), domain.DecisionInput{Approve: true}, RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrAlreadyDecided)
}

// Hanya approver tahap aktif yang boleh memutus.
func TestLeaveDecideEnforcesStageAuthorization(t *testing.T) {
	cases := []struct {
		status domain.RequestStatus
		role   domain.RoleName
	}{
		{domain.StatusWaitingSupervisor, domain.RoleHR},
		{domain.StatusWaitingSupervisor, domain.RoleTopManagement},
		{domain.StatusWaitingHR, domain.RoleSupervisor},
		{domain.StatusWaitingHR, domain.RoleTopManagement},
		{domain.StatusWaitingTopManagement, domain.RoleHR},
		{domain.StatusWaitingTopManagement, domain.RoleEmployee},
	}

	for _, testCase := range cases {
		store := &leaveStoreStub{lock: decisionLock(testCase.status, nil, 12)}
		service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})

		_, err := service.Decide(
			context.Background(),
			domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: testCase.role},
			uuid.New(), domain.DecisionInput{Approve: true}, RequestMeta{},
		)

		require.ErrorIsf(t, err, domain.ErrForbidden,
			"role %s tidak boleh memutus status %s", testCase.role, testCase.status)
	}
}

// Atasan hanya boleh memutus pengajuan bawahan langsungnya.
func TestLeaveDecideRejectsSupervisorOfAnotherTeam(t *testing.T) {
	otherSupervisor := uuid.New()
	store := &leaveStoreStub{lock: decisionLock(
		domain.StatusWaitingSupervisor, &otherSupervisor, 12,
	)}
	service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.Decide(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleSupervisor},
		uuid.New(), domain.DecisionInput{Approve: true}, RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrForbidden)
}

// Delegasi memindahkan ke HR dan mencatat riwayat delegate.
func TestLeaveDelegateMovesToHR(t *testing.T) {
	supervisorEmployee := uuid.New()
	store := &leaveStoreStub{lock: decisionLock(
		domain.StatusWaitingSupervisor, &supervisorEmployee, 12,
	)}
	service, events, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})

	result, err := service.Delegate(
		context.Background(),
		domain.Identity{
			UserID: uuid.New(), EmployeeID: supervisorEmployee, Role: domain.RoleSupervisor,
		},
		uuid.New(), "Didelegasikan karena cuti", RequestMeta{},
	)

	require.NoError(t, err)
	assert.Equal(t, domain.StatusWaitingHR, result.Status)
	require.Len(t, store.appended, 1)
	assert.Equal(t, domain.DecisionDelegate, store.appended[0].decision)
	require.Len(t, events.events, 1)
	assert.Equal(t, domain.EventLeaveDelegated, events.events[0].Type)
}

func TestLeaveDelegateRestrictedToSupervisorStage(t *testing.T) {
	supervisorEmployee := uuid.New()

	// Role selain Atasan tidak boleh mendelegasikan.
	store := &leaveStoreStub{lock: decisionLock(domain.StatusWaitingHR, nil, 12)}
	service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})
	_, err := service.Delegate(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR},
		uuid.New(), "catatan delegasi", RequestMeta{},
	)
	require.ErrorIs(t, err, domain.ErrForbidden)

	// Setelah pengajuan meninggalkan tahap Atasan, delegasi kalah bersaing.
	store = &leaveStoreStub{lock: decisionLock(domain.StatusWaitingHR, &supervisorEmployee, 12)}
	service, _, _ = newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})
	_, err = service.Delegate(
		context.Background(),
		domain.Identity{
			UserID: uuid.New(), EmployeeID: supervisorEmployee, Role: domain.RoleSupervisor,
		},
		uuid.New(), "catatan delegasi", RequestMeta{},
	)
	require.ErrorIs(t, err, domain.ErrAlreadyDecided)
}

// Inbox approver dibatasi tahap dan relasi yang diizinkan.
func TestLeaveListForApprovalAppliesRoleScope(t *testing.T) {
	supervisorEmployee := uuid.New()

	store := &leaveStoreStub{}
	service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})
	_, err := service.ListForApproval(context.Background(), domain.Identity{
		UserID: uuid.New(), EmployeeID: supervisorEmployee, Role: domain.RoleSupervisor,
	}, nil, 1, 10)
	require.NoError(t, err)
	require.NotNil(t, store.filter.Scope.SupervisorEmployeeID)
	assert.Equal(t, supervisorEmployee, *store.filter.Scope.SupervisorEmployeeID)
	require.NotNil(t, store.filter.Scope.Stage)
	assert.Equal(t, domain.StatusWaitingSupervisor, *store.filter.Scope.Stage)

	store = &leaveStoreStub{}
	service, _, _ = newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})
	_, err = service.ListForApproval(context.Background(), domain.Identity{
		UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR,
	}, nil, 1, 10)
	require.NoError(t, err)
	require.NotNil(t, store.filter.Scope.Stage)
	assert.Equal(t, domain.StatusWaitingHR, *store.filter.Scope.Stage)
	assert.Nil(t, store.filter.Scope.SupervisorEmployeeID)

	// Top Management hanya melihat pengajuan milik HR pada tahapnya.
	store = &leaveStoreStub{}
	service, _, _ = newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})
	_, err = service.ListForApproval(context.Background(), domain.Identity{
		UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleTopManagement,
	}, nil, 1, 10)
	require.NoError(t, err)
	require.NotNil(t, store.filter.Scope.Stage)
	assert.Equal(t, domain.StatusWaitingTopManagement, *store.filter.Scope.Stage)
	require.NotNil(t, store.filter.Scope.RequesterRole)
	assert.Equal(t, domain.RoleHR, *store.filter.Scope.RequesterRole)

	// Karyawan tidak memiliki inbox approval.
	service, _, _ = newLeaveServiceForTest(&leaveStoreStub{}, supervisorStub{}, transactionStub{})
	_, err = service.ListForApproval(context.Background(), domain.Identity{
		UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee,
	}, nil, 1, 10)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLeaveListMineScopesToRequester(t *testing.T) {
	userID := uuid.New()
	store := &leaveStoreStub{}
	service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.ListMine(context.Background(), domain.Identity{
		UserID: userID, EmployeeID: uuid.New(), Role: domain.RoleEmployee,
	}, 0, 999)

	require.NoError(t, err)
	require.NotNil(t, store.filter.Scope.RequesterUserID)
	assert.Equal(t, userID, *store.filter.Scope.RequesterUserID)
	// Paging dibatasi agar query selalu terikat.
	assert.Equal(t, 1, store.filter.Page)
	assert.Equal(t, 100, store.filter.Limit)
}

// Detail yang tidak berhak dibaca menghasilkan NOT_FOUND agar keberadaan record tidak bocor.
func TestLeaveDetailHidesExistenceFromUnauthorisedRole(t *testing.T) {
	owner := uuid.New()
	store := &leaveStoreStub{detail: domain.LeaveRequestDetail{
		LeaveRequestSummary: domain.LeaveRequestSummary{ID: uuid.New(), EmployeeID: owner},
	}}
	service, _, _ := newLeaveServiceForTest(
		store, supervisorStub{supervisor: nil}, transactionStub{},
	)

	// Karyawan lain tidak boleh membaca pengajuan milik orang lain.
	_, err := service.Detail(context.Background(), domain.Identity{
		UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee,
	}, uuid.New())
	require.ErrorIs(t, err, domain.ErrNotFound)

	// Pemilik pengajuan tetap dapat membacanya.
	_, err = service.Detail(context.Background(), domain.Identity{
		UserID: uuid.New(), EmployeeID: owner, Role: domain.RoleEmployee,
	}, uuid.New())
	require.NoError(t, err)
}

func TestLeaveTypeAdministrationRestrictedToHR(t *testing.T) {
	service, _, _ := newLeaveServiceForTest(&leaveStoreStub{}, supervisorStub{}, transactionStub{})

	for _, role := range []domain.RoleName{
		domain.RoleEmployee, domain.RoleSupervisor, domain.RoleTopManagement,
	} {
		identity := domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: role}

		_, createErr := service.CreateLeaveType(context.Background(), identity,
			domain.CreateLeaveType{Code: "X", Name: "Y", AnnualQuota: 1}, RequestMeta{})
		require.ErrorIsf(t, createErr, domain.ErrForbidden, "role %s", role)

		quota := 5
		updateErr := service.UpdateLeaveType(context.Background(), identity, uuid.New(),
			domain.UpdateLeaveType{AnnualQuota: &quota}, RequestMeta{})
		require.ErrorIsf(t, updateErr, domain.ErrForbidden, "role %s", role)
	}
}

// Pemohon membutuhkan daftar jenis izin untuk mengisi jenis_izin_id pada POST
// /ketidakhadiran, sehingga read master terbuka untuk seluruh role terautentikasi.
func TestListLeaveTypesForcesActiveOnlyForNonHR(t *testing.T) {
	for _, role := range []domain.RoleName{
		domain.RoleEmployee, domain.RoleSupervisor, domain.RoleTopManagement,
	} {
		store := &leaveStoreStub{}
		service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})
		identity := domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: role}

		items, err := service.ListLeaveTypes(context.Background(), identity, nil)
		require.NoErrorf(t, err, "role %s", role)
		require.Lenf(t, items, 1, "role %s", role)
		require.NotNilf(t, store.listActive, "role %s", role)
		assert.Truef(t, *store.listActive, "role %s", role)

		// Permintaan eksplisit untuk master nonaktif tidak boleh melonggarkan batas ini.
		inactive := false
		_, err = service.ListLeaveTypes(context.Background(), identity, &inactive)
		require.NoErrorf(t, err, "role %s", role)
		require.NotNilf(t, store.listActive, "role %s", role)
		assert.Truef(t, *store.listActive, "role %s", role)
	}
}

func TestListLeaveTypesKeepsHRFilterUntouched(t *testing.T) {
	store := &leaveStoreStub{}
	service, _, _ := newLeaveServiceForTest(store, supervisorStub{}, transactionStub{})
	identity := domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR}

	_, err := service.ListLeaveTypes(context.Background(), identity, nil)
	require.NoError(t, err)
	assert.Nil(t, store.listActive)

	inactive := false
	_, err = service.ListLeaveTypes(context.Background(), identity, &inactive)
	require.NoError(t, err)
	require.NotNil(t, store.listActive)
	assert.False(t, *store.listActive)
}

func TestLeaveTypeRejectsNegativeQuota(t *testing.T) {
	service, _, _ := newLeaveServiceForTest(&leaveStoreStub{}, supervisorStub{}, transactionStub{})
	identity := domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR}

	_, err := service.CreateLeaveType(context.Background(), identity,
		domain.CreateLeaveType{Code: "X", Name: "Y", AnnualQuota: -1}, RequestMeta{})
	require.ErrorIs(t, err, domain.ErrInvalidRequest)

	negative := -3
	err = service.UpdateLeaveType(context.Background(), identity, uuid.New(),
		domain.UpdateLeaveType{AnnualQuota: &negative}, RequestMeta{})
	require.ErrorIs(t, err, domain.ErrInvalidRequest)
}
