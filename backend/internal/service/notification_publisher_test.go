package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// notificationWriterStub meniru penyimpanan notifikasi beserta resolusi penerima.
type notificationWriterStub struct {
	approvers map[domain.ApprovalStage][]uuid.UUID
	inserted  []domain.NotificationDraft
	// stored menirukan UNIQUE (recipient_user_id, event_key) lintas pemanggilan.
	stored     map[string]struct{}
	insertErr  error
	resolveErr error
}

func newNotificationWriterStub() *notificationWriterStub {
	return &notificationWriterStub{
		approvers: map[domain.ApprovalStage][]uuid.UUID{},
		stored:    map[string]struct{}{},
	}
}

func (s *notificationWriterStub) Insert(
	_ context.Context,
	drafts []domain.NotificationDraft,
) (int, error) {
	if s.insertErr != nil {
		return 0, s.insertErr
	}
	created := 0
	for _, draft := range drafts {
		key := draft.RecipientUserID.String() + "|" + draft.EventKey
		if _, duplicate := s.stored[key]; duplicate {
			continue
		}
		s.stored[key] = struct{}{}
		s.inserted = append(s.inserted, draft)
		created++
	}
	return created, nil
}

func (s *notificationWriterStub) ApproverUserIDs(
	_ context.Context,
	stage domain.ApprovalStage,
	_ uuid.UUID,
) ([]uuid.UUID, error) {
	if s.resolveErr != nil {
		return nil, s.resolveErr
	}
	return s.approvers[stage], nil
}

func (s *notificationWriterStub) recipients() map[uuid.UUID]domain.NotificationType {
	result := map[uuid.UUID]domain.NotificationType{}
	for _, draft := range s.inserted {
		result[draft.RecipientUserID] = draft.Type
	}
	return result
}

func newPublisherForTest() (*NotificationPublisher, *notificationWriterStub) {
	store := newNotificationWriterStub()
	return NewNotificationPublisher(store, slog.New(slog.DiscardHandler)), store
}

var publishedAt = time.Date(2026, 8, 3, 2, 0, 0, 0, time.UTC)

func TestPublishNotifiesSupervisorOnEmployeeSubmission(t *testing.T) {
	publisher, store := newPublisherForTest()
	requester := uuid.New()
	supervisor := uuid.New()
	store.approvers[domain.StageSupervisor] = []uuid.UUID{supervisor}

	require.NoError(t, publisher.Publish(context.Background(), domain.ApprovalEvent{
		Type:            domain.EventLeaveSubmitted,
		RequestID:       uuid.New(),
		RequesterUserID: requester,
		ActorUserID:     &requester,
		Status:          domain.StatusWaitingSupervisor,
		Stage:           domain.StageSupervisor,
		OccurredAt:      publishedAt,
	}))

	assert.Equal(t, map[uuid.UUID]domain.NotificationType{
		supervisor: domain.NotificationLeaveSubmitted,
	}, store.recipients())
}

// Pengajuan HR berjalan ke Top Management; approver tahap itulah yang diberi tahu.
func TestPublishNotifiesTopManagementOnHumanResourcesSubmission(t *testing.T) {
	publisher, store := newPublisherForTest()
	requester := uuid.New()
	topManagement := uuid.New()
	store.approvers[domain.StageTopManagement] = []uuid.UUID{topManagement}

	require.NoError(t, publisher.Publish(context.Background(), domain.ApprovalEvent{
		Type:            domain.EventOvertimeSubmitted,
		RequestID:       uuid.New(),
		RequesterUserID: requester,
		ActorUserID:     &requester,
		Status:          domain.StatusWaitingTopManagement,
		Stage:           domain.StageTopManagement,
		OccurredAt:      publishedAt,
	}))

	assert.Equal(t, map[uuid.UUID]domain.NotificationType{
		topManagement: domain.NotificationOvertimeSubmitted,
	}, store.recipients())
}

// Persetujuan Atasan memberi tahu pemohon sekaligus seluruh HR sebagai approver berikutnya.
func TestPublishNotifiesRequesterAndNextStageOnApproval(t *testing.T) {
	publisher, store := newPublisherForTest()
	requester := uuid.New()
	supervisor := uuid.New()
	firstHR, secondHR := uuid.New(), uuid.New()
	store.approvers[domain.StageHR] = []uuid.UUID{firstHR, secondHR}
	nextStage := domain.StageHR

	require.NoError(t, publisher.Publish(context.Background(), domain.ApprovalEvent{
		Type:            domain.EventLeaveDecisionChanged,
		RequestID:       uuid.New(),
		RequesterUserID: requester,
		ActorUserID:     &supervisor,
		Status:          domain.StatusWaitingHR,
		Stage:           domain.StageSupervisor,
		NextStage:       &nextStage,
		OccurredAt:      publishedAt,
	}))

	assert.Equal(t, map[uuid.UUID]domain.NotificationType{
		requester: domain.NotificationDecisionApproved,
		firstHR:   domain.NotificationLeaveSubmitted,
		secondHR:  domain.NotificationLeaveSubmitted,
	}, store.recipients())
}

func TestPublishNotifiesOnlyRequesterOnFinalDecision(t *testing.T) {
	publisher, store := newPublisherForTest()
	requester := uuid.New()
	approver := uuid.New()
	store.approvers[domain.StageHR] = []uuid.UUID{approver}

	require.NoError(t, publisher.Publish(context.Background(), domain.ApprovalEvent{
		Type:            domain.EventLeaveDecisionChanged,
		RequestID:       uuid.New(),
		RequesterUserID: requester,
		ActorUserID:     &approver,
		Status:          domain.StatusRejected,
		Stage:           domain.StageHR,
		NextStage:       nil,
		OccurredAt:      publishedAt,
	}))

	assert.Equal(t, map[uuid.UUID]domain.NotificationType{
		requester: domain.NotificationDecisionRejected,
	}, store.recipients())
}

func TestPublishNotifiesHumanResourcesAndRequesterOnDelegation(t *testing.T) {
	publisher, store := newPublisherForTest()
	requester := uuid.New()
	supervisor := uuid.New()
	humanResources := uuid.New()
	store.approvers[domain.StageHR] = []uuid.UUID{humanResources}

	require.NoError(t, publisher.Publish(context.Background(), domain.ApprovalEvent{
		Type:            domain.EventLeaveDelegated,
		RequestID:       uuid.New(),
		RequesterUserID: requester,
		ActorUserID:     &supervisor,
		Status:          domain.StatusWaitingHR,
		Stage:           domain.StageSupervisor,
		OccurredAt:      publishedAt,
	}))

	assert.Equal(t, map[uuid.UUID]domain.NotificationType{
		humanResources: domain.NotificationDelegated,
		requester:      domain.NotificationDelegated,
	}, store.recipients())
}

// Eskalasi dipicu sistem sehingga tidak ada aktor; HR dan pemohon tetap diberi tahu.
func TestPublishNotifiesHumanResourcesAndRequesterOnEscalation(t *testing.T) {
	publisher, store := newPublisherForTest()
	requester := uuid.New()
	humanResources := uuid.New()
	store.approvers[domain.StageHR] = []uuid.UUID{humanResources}

	require.NoError(t, publisher.Publish(context.Background(), domain.ApprovalEvent{
		Type:            domain.EventOvertimeAutoEscalated,
		RequestID:       uuid.New(),
		RequesterUserID: requester,
		ActorUserID:     nil,
		Status:          domain.StatusWaitingHR,
		Stage:           domain.StageSupervisor,
		OccurredAt:      publishedAt,
	}))

	assert.Equal(t, map[uuid.UUID]domain.NotificationType{
		humanResources: domain.NotificationAutoEscalated,
		requester:      domain.NotificationAutoEscalated,
	}, store.recipients())
}

// Approver yang juga pemohon atau aktor tidak boleh menerima notifikasi atas aksinya sendiri.
func TestPublishNeverSelfNotifiesApprover(t *testing.T) {
	publisher, store := newPublisherForTest()
	requester := uuid.New()
	actor := uuid.New()
	other := uuid.New()
	store.approvers[domain.StageHR] = []uuid.UUID{requester, actor, other}

	require.NoError(t, publisher.Publish(context.Background(), domain.ApprovalEvent{
		Type:            domain.EventLeaveSubmitted,
		RequestID:       uuid.New(),
		RequesterUserID: requester,
		ActorUserID:     &actor,
		Status:          domain.StatusWaitingHR,
		Stage:           domain.StageHR,
		OccurredAt:      publishedAt,
	}))

	assert.Equal(t, map[uuid.UUID]domain.NotificationType{
		other: domain.NotificationLeaveSubmitted,
	}, store.recipients())
}

// Retry producer atas event yang sama tidak boleh menambah baris.
func TestPublishIsIdempotentAcrossRetries(t *testing.T) {
	publisher, store := newPublisherForTest()
	requester := uuid.New()
	supervisor := uuid.New()
	store.approvers[domain.StageSupervisor] = []uuid.UUID{supervisor}
	event := domain.ApprovalEvent{
		Type:            domain.EventLeaveSubmitted,
		RequestID:       uuid.New(),
		RequesterUserID: requester,
		ActorUserID:     &requester,
		Status:          domain.StatusWaitingSupervisor,
		Stage:           domain.StageSupervisor,
		OccurredAt:      publishedAt,
	}

	require.NoError(t, publisher.Publish(context.Background(), event))
	require.NoError(t, publisher.Publish(context.Background(), event))

	assert.Len(t, store.inserted, 1)
}

// Event duplikat dalam satu batch tidak menghasilkan draft ganda.
func TestPublishDeduplicatesWithinOneBatch(t *testing.T) {
	publisher, store := newPublisherForTest()
	requester := uuid.New()
	supervisor := uuid.New()
	store.approvers[domain.StageSupervisor] = []uuid.UUID{supervisor}
	event := domain.ApprovalEvent{
		Type:            domain.EventLeaveSubmitted,
		RequestID:       uuid.New(),
		RequesterUserID: requester,
		ActorUserID:     &requester,
		Status:          domain.StatusWaitingSupervisor,
		Stage:           domain.StageSupervisor,
		OccurredAt:      publishedAt,
	}

	require.NoError(t, publisher.Publish(context.Background(), event, event))

	assert.Len(t, store.inserted, 1)
}

// Kegagalan penulisan harus naik ke pemanggil agar transaction approval ikut dibatalkan.
func TestPublishPropagatesStoreFailure(t *testing.T) {
	publisher, store := newPublisherForTest()
	store.approvers[domain.StageSupervisor] = []uuid.UUID{uuid.New()}
	store.insertErr = errors.New("insert gagal")

	err := publisher.Publish(context.Background(), domain.ApprovalEvent{
		Type:            domain.EventLeaveSubmitted,
		RequestID:       uuid.New(),
		RequesterUserID: uuid.New(),
		Status:          domain.StatusWaitingSupervisor,
		Stage:           domain.StageSupervisor,
		OccurredAt:      publishedAt,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "write approval notifications")
}

// Pengajuan tanpa approver aktif tidak boleh menghasilkan notifikasi tanpa penerima.
func TestPublishSkipsWhenNoApproverIsActive(t *testing.T) {
	publisher, store := newPublisherForTest()

	require.NoError(t, publisher.Publish(context.Background(), domain.ApprovalEvent{
		Type:            domain.EventLeaveSubmitted,
		RequestID:       uuid.New(),
		RequesterUserID: uuid.New(),
		Status:          domain.StatusWaitingSupervisor,
		Stage:           domain.StageSupervisor,
		OccurredAt:      publishedAt,
	}))

	assert.Empty(t, store.inserted)
}
