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

type overtimeStoreStub struct {
	createdRow domain.OvertimeRequestRow
	lock       domain.RequestLock
	statusErr  error
	appended   []appendedApproval
	filter     domain.OvertimeRequestFilter
	detail     domain.OvertimeRequestDetail
	detailErr  error
	recap      []domain.OvertimeRecapItem
}

func (s *overtimeStoreStub) CreateRequest(
	_ context.Context, row domain.OvertimeRequestRow,
) (uuid.UUID, error) {
	s.createdRow = row
	return uuid.New(), nil
}

func (s *overtimeStoreStub) ListRequests(
	_ context.Context, filter domain.OvertimeRequestFilter,
) (domain.OvertimeRequestPage, error) {
	s.filter = filter
	return domain.OvertimeRequestPage{Page: filter.Page, Limit: filter.Limit}, nil
}

func (s *overtimeStoreStub) FindRequest(
	context.Context, uuid.UUID,
) (domain.OvertimeRequestDetail, error) {
	return s.detail, s.detailErr
}

func (s *overtimeStoreStub) LockRequestForDecision(
	context.Context, uuid.UUID,
) (domain.RequestLock, error) {
	return s.lock, nil
}

func (s *overtimeStoreStub) UpdateRequestStatus(
	context.Context, uuid.UUID, domain.RequestStatus, domain.RequestStatus,
) error {
	return s.statusErr
}

func (s *overtimeStoreStub) AppendApproval(
	_ context.Context, _ uuid.UUID, stage domain.ApprovalStage, approverID *uuid.UUID,
	decision domain.ApprovalDecision, note *string,
) error {
	s.appended = append(s.appended, appendedApproval{stage, approverID, decision, note})
	return nil
}

func (s *overtimeStoreStub) Recap(
	_ context.Context, _ domain.OvertimeRecapFilter,
) ([]domain.OvertimeRecapItem, error) {
	return s.recap, nil
}

func newOvertimeServiceForTest(
	store OvertimeStore, supervisor SupervisorLookup, transactions EmployeeTransactionManager,
) (*OvertimeService, *eventRecorder, *documentStoreStub) {
	recorder := &eventRecorder{}
	documents := &documentStoreStub{}
	service := NewOvertimeService(
		store, supervisor, transactions, auditStub{}, documents, recorder,
	)
	service.now = func() time.Time {
		return time.Date(2026, time.August, 3, 9, 0, 0, 0, domain.Jakarta())
	}
	return service, recorder, documents
}

func overtimeCommand(document *domain.UploadedFile) domain.CreateOvertimeRequest {
	return domain.CreateOvertimeRequest{
		Date:      "2026-08-10",
		StartTime: "18:00:00",
		EndTime:   "20:30:00",
		Reason:    "Penyelesaian rilis sintetis",
		Document:  document,
	}
}

func TestOvertimeCreateRoutesByRequesterRole(t *testing.T) {
	supervisorID := uuid.New()
	cases := []struct {
		role       domain.RoleName
		supervisor *uuid.UUID
		expected   domain.RequestStatus
	}{
		{domain.RoleEmployee, &supervisorID, domain.StatusWaitingSupervisor},
		{domain.RoleEmployee, nil, domain.StatusWaitingHR},
		{domain.RoleSupervisor, nil, domain.StatusWaitingHR},
		{domain.RoleHR, nil, domain.StatusWaitingTopManagement},
	}

	for _, testCase := range cases {
		store := &overtimeStoreStub{}
		service, events, _ := newOvertimeServiceForTest(
			store, supervisorStub{supervisor: testCase.supervisor}, transactionStub{},
		)

		result, err := service.Create(
			context.Background(),
			domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: testCase.role},
			overtimeCommand(nil), RequestMeta{},
		)

		require.NoErrorf(t, err, "role %s", testCase.role)
		assert.Equalf(t, testCase.expected, result.Status, "role %s", testCase.role)
		require.Len(t, events.events, 1)
		assert.Equal(t, domain.EventOvertimeSubmitted, events.events[0].Type)
	}
}

// Dokumen pendukung lembur bersifat opsional, berbeda dengan ketidakhadiran.
func TestOvertimeCreateAcceptsMissingDocument(t *testing.T) {
	store := &overtimeStoreStub{}
	service, _, documents := newOvertimeServiceForTest(
		store, supervisorStub{}, transactionStub{},
	)

	_, err := service.Create(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee},
		overtimeCommand(nil), RequestMeta{},
	)

	require.NoError(t, err)
	assert.Empty(t, documents.uploadedPath)
	assert.Nil(t, store.createdRow.DocumentURL)
}

func TestOvertimeCreateAcceptsBrowserTimeAndNormalizesIt(t *testing.T) {
	store := &overtimeStoreStub{}
	service, _, _ := newOvertimeServiceForTest(
		store, supervisorStub{}, transactionStub{},
	)
	command := overtimeCommand(nil)
	command.StartTime, command.EndTime = "18:00", "20:30"

	_, err := service.Create(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee},
		command, RequestMeta{},
	)

	require.NoError(t, err)
	assert.Equal(t, "18:00:00", store.createdRow.StartTime)
	assert.Equal(t, "20:30:00", store.createdRow.EndTime)
}

// Jam selesai harus setelah jam mulai.
func TestOvertimeCreateRejectsInvalidTimeRange(t *testing.T) {
	service, _, _ := newOvertimeServiceForTest(
		&overtimeStoreStub{}, supervisorStub{}, transactionStub{},
	)
	identity := domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee}

	reversed := overtimeCommand(nil)
	reversed.StartTime, reversed.EndTime = "20:00:00", "18:00:00"
	_, err := service.Create(context.Background(), identity, reversed, RequestMeta{})
	require.ErrorIs(t, err, domain.ErrInvalidRequest)

	equal := overtimeCommand(nil)
	equal.StartTime, equal.EndTime = "18:00:00", "18:00:00"
	_, err = service.Create(context.Background(), identity, equal, RequestMeta{})
	require.ErrorIs(t, err, domain.ErrInvalidRequest)

	invalidDate := overtimeCommand(nil)
	invalidDate.Date = "10 Agustus 2026"
	_, err = service.Create(context.Background(), identity, invalidDate, RequestMeta{})
	require.ErrorIs(t, err, domain.ErrInvalidRequest)
}

func TestOvertimeCreateRemovesOrphanDocumentWhenTransactionFails(t *testing.T) {
	service, _, documents := newOvertimeServiceForTest(
		&overtimeStoreStub{}, supervisorStub{},
		failingTransactionStub{err: errors.New("database unavailable")},
	)

	_, err := service.Create(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee},
		overtimeCommand(syntheticDocument()), RequestMeta{},
	)

	require.Error(t, err)
	assert.NotEmpty(t, documents.uploadedPath)
	assert.Equal(t, documents.uploadedPath, documents.deletedPath)
}

func TestOvertimeDecideFollowsStageRules(t *testing.T) {
	supervisorEmployee := uuid.New()
	store := &overtimeStoreStub{lock: decisionLock(
		domain.StatusWaitingSupervisor, &supervisorEmployee, 0,
	)}
	service, events, _ := newOvertimeServiceForTest(store, supervisorStub{}, transactionStub{})

	result, err := service.Decide(
		context.Background(),
		domain.Identity{
			UserID: uuid.New(), EmployeeID: supervisorEmployee, Role: domain.RoleSupervisor,
		},
		uuid.New(), domain.DecisionInput{Approve: true}, RequestMeta{},
	)

	require.NoError(t, err)
	assert.Equal(t, domain.StatusWaitingHR, result.Status)
	require.Len(t, store.appended, 1)
	assert.Equal(t, domain.DecisionApprove, store.appended[0].decision)
	require.Len(t, events.events, 1)
	assert.Equal(t, domain.EventOvertimeDecisionChanged, events.events[0].Type)
}

func TestOvertimeDecideRequiresNoteOnReject(t *testing.T) {
	store := &overtimeStoreStub{lock: decisionLock(domain.StatusWaitingHR, nil, 0)}
	service, _, _ := newOvertimeServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.Decide(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR},
		uuid.New(), domain.DecisionInput{Approve: false}, RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrInvalidRequest)
	assert.Empty(t, store.appended)
}

func TestOvertimeDecideMapsConcurrentUpdateToAlreadyDecided(t *testing.T) {
	store := &overtimeStoreStub{
		lock:      decisionLock(domain.StatusWaitingHR, nil, 0),
		statusErr: repository.ErrConflict,
	}
	service, _, _ := newOvertimeServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.Decide(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR},
		uuid.New(), domain.DecisionInput{Approve: true}, RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrAlreadyDecided)
}

func TestOvertimeDecideRejectsWrongStageRole(t *testing.T) {
	store := &overtimeStoreStub{lock: decisionLock(domain.StatusWaitingHR, nil, 0)}
	service, _, _ := newOvertimeServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.Decide(
		context.Background(),
		domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleSupervisor},
		uuid.New(), domain.DecisionInput{Approve: true}, RequestMeta{},
	)

	require.ErrorIs(t, err, domain.ErrForbidden)
}

// Detail lembur milik orang lain tidak bocor ke karyawan lain.
func TestOvertimeDetailHidesExistenceFromUnauthorisedRole(t *testing.T) {
	owner := uuid.New()
	store := &overtimeStoreStub{detail: domain.OvertimeRequestDetail{
		OvertimeRequestSummary: domain.OvertimeRequestSummary{ID: uuid.New(), EmployeeID: owner},
	}}
	service, _, _ := newOvertimeServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.Detail(context.Background(), domain.Identity{
		UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee,
	}, uuid.New())
	require.ErrorIs(t, err, domain.ErrNotFound)

	_, err = service.Detail(context.Background(), domain.Identity{
		UserID: uuid.New(), EmployeeID: owner, Role: domain.RoleEmployee,
	}, uuid.New())
	require.NoError(t, err)
}

// Rekap hanya untuk HR dan tidak menghitung kompensasi.
func TestOvertimeRecapRestrictedToHR(t *testing.T) {
	store := &overtimeStoreStub{recap: []domain.OvertimeRecapItem{
		{EmployeeName: "Karyawan Uji", TotalRequest: 2, TotalHours: 5},
	}}
	service, _, _ := newOvertimeServiceForTest(store, supervisorStub{}, transactionStub{})

	for _, role := range []domain.RoleName{
		domain.RoleEmployee, domain.RoleSupervisor, domain.RoleTopManagement,
	} {
		_, err := service.Recap(
			context.Background(), domain.Identity{Role: role}, domain.OvertimeRecapFilter{},
		)
		require.ErrorIsf(t, err, domain.ErrForbidden, "role %s harus ditolak", role)
	}

	items, err := service.Recap(
		context.Background(), domain.Identity{Role: domain.RoleHR}, domain.OvertimeRecapFilter{},
	)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, 2, items[0].TotalRequest)
	assert.InDelta(t, 5, items[0].TotalHours, 0.001)
}

func TestOvertimeListAppliesApprovalScope(t *testing.T) {
	supervisorEmployee := uuid.New()
	store := &overtimeStoreStub{}
	service, _, _ := newOvertimeServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.List(context.Background(), domain.Identity{
		UserID: uuid.New(), EmployeeID: supervisorEmployee, Role: domain.RoleSupervisor,
	}, domain.OvertimeRequestFilter{Page: 1, Limit: 10})

	require.NoError(t, err)
	require.NotNil(t, store.filter.Scope.SupervisorEmployeeID)
	assert.Equal(t, supervisorEmployee, *store.filter.Scope.SupervisorEmployeeID)
	require.NotNil(t, store.filter.Scope.Stage)
	assert.Equal(t, domain.StatusWaitingSupervisor, *store.filter.Scope.Stage)
}

// Karyawan bukan approver tidak boleh memakai inbox approval untuk melihat pengajuan
// sendiri; List harus tetap menolaknya (mirip TestLeaveListForApprovalScopesByRole).
func TestOvertimeListRejectsRequesterWithoutApprovalRole(t *testing.T) {
	store := &overtimeStoreStub{}
	service, _, _ := newOvertimeServiceForTest(store, supervisorStub{}, transactionStub{})

	_, err := service.List(context.Background(), domain.Identity{
		UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee,
	}, domain.OvertimeRequestFilter{Page: 1, Limit: 10})

	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestOvertimeListMineScopesToRequester(t *testing.T) {
	userID := uuid.New()
	store := &overtimeStoreStub{}
	service, _, _ := newOvertimeServiceForTest(store, supervisorStub{}, transactionStub{})

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

func TestOvertimeListMineRejectsTopManagement(t *testing.T) {
	service, _, _ := newOvertimeServiceForTest(&overtimeStoreStub{}, supervisorStub{}, transactionStub{})

	_, err := service.ListMine(context.Background(), domain.Identity{
		UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleTopManagement,
	}, 1, 10)

	require.ErrorIs(t, err, domain.ErrForbidden)
}
