package worker

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

type transactionStub struct{}

func (transactionStub) Within(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type auditStub struct{ entries []domain.AuditEntry }

func (a *auditStub) Append(_ context.Context, entry domain.AuditEntry) error {
	a.entries = append(a.entries, entry)
	return nil
}

type eventRecorder struct{ events []domain.ApprovalEvent }

func (r *eventRecorder) Publish(_ context.Context, events ...domain.ApprovalEvent) error {
	r.events = append(r.events, events...)
	return nil
}

type escalationStoreStub struct {
	// pending mensimulasikan baris yang masih berada pada tahap atasan.
	pending    map[uuid.UUID]bool
	claimed    []domain.EscalationCandidate
	threshold  time.Time
	appended   []domain.ApprovalDecision
	approvers  []*uuid.UUID
	claimCalls int
}

func (s *escalationStoreStub) ClaimEscalatableRequests(
	_ context.Context, before time.Time, _ int,
) ([]domain.EscalationCandidate, error) {
	s.threshold = before
	s.claimCalls++
	remaining := make([]domain.EscalationCandidate, 0)
	for _, candidate := range s.claimed {
		if s.pending[candidate.RequestID] {
			remaining = append(remaining, candidate)
		}
	}
	return remaining, nil
}

func (s *escalationStoreStub) UpdateRequestStatus(
	_ context.Context, id uuid.UUID, from, to domain.RequestStatus,
) error {
	if from != domain.StatusWaitingSupervisor || to != domain.StatusWaitingHR {
		return errors.New("transisi eskalasi tidak sah")
	}
	if !s.pending[id] {
		return errors.New("status sudah berubah")
	}
	s.pending[id] = false
	return nil
}

func (s *escalationStoreStub) AppendApproval(
	_ context.Context, _ uuid.UUID, _ domain.ApprovalStage, approverID *uuid.UUID,
	decision domain.ApprovalDecision, _ *string,
) error {
	s.appended = append(s.appended, decision)
	s.approvers = append(s.approvers, approverID)
	return nil
}

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func newEscalationJobForTest(
	store EscalationStore, audit AuditWriter, events domain.ApprovalEventPublisher, now time.Time,
) *EscalationJob {
	job := NewEscalationJob(
		"ketidakhadiran", store, transactionStub{}, audit, events,
		domain.EventLeaveAutoEscalated, discardLogger(),
	)
	job.now = func() time.Time { return now }
	return job
}

func TestEscalationMovesStaleRequestsToHR(t *testing.T) {
	requestID := uuid.New()
	requester := uuid.New()
	store := &escalationStoreStub{
		pending: map[uuid.UUID]bool{requestID: true},
		claimed: []domain.EscalationCandidate{{RequestID: requestID, RequesterUserID: requester}},
	}
	audit := &auditStub{}
	events := &eventRecorder{}
	job := newEscalationJobForTest(store, audit, events, time.Now())

	count, err := job.Run(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.Len(t, store.appended, 1)
	assert.Equal(t, domain.DecisionAutoEscalate, store.appended[0])
	// Keputusan sistem tidak memiliki approver.
	assert.Nil(t, store.approvers[0])

	require.Len(t, audit.entries, 1)
	assert.Equal(t, "AUTO_ESCALATE", audit.entries[0].Action)
	assert.Nil(t, audit.entries[0].UserID, "aktor sistem tidak memiliki user")

	require.Len(t, events.events, 1)
	assert.Equal(t, domain.EventLeaveAutoEscalated, events.events[0].Type)
	assert.Equal(t, domain.StatusWaitingHR, events.events[0].Status)
	assert.Nil(t, events.events[0].ActorUserID)
	require.NotNil(t, events.events[0].NextStage)
	assert.Equal(t, domain.StageHR, *events.events[0].NextStage)
}

// Threshold yang dikirim ke repository adalah waktu sekarang dikurangi SLA 2x24 jam.
func TestEscalationUsesTwoDayThreshold(t *testing.T) {
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.UTC)
	store := &escalationStoreStub{pending: map[uuid.UUID]bool{}}
	job := newEscalationJobForTest(store, &auditStub{}, &eventRecorder{}, now)

	_, err := job.Run(context.Background())

	require.NoError(t, err)
	assert.Equal(t, now.Add(-48*time.Hour), store.threshold)
}

// Eksekusi berulang tidak mengeskalasi ulang pengajuan yang sudah berpindah tahap.
func TestEscalationIsIdempotentAcrossRuns(t *testing.T) {
	requestID := uuid.New()
	store := &escalationStoreStub{
		pending: map[uuid.UUID]bool{requestID: true},
		claimed: []domain.EscalationCandidate{{RequestID: requestID, RequesterUserID: uuid.New()}},
	}
	audit := &auditStub{}
	events := &eventRecorder{}
	job := newEscalationJobForTest(store, audit, events, time.Now())

	first, err := job.Run(context.Background())
	require.NoError(t, err)
	second, err := job.Run(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, first)
	assert.Equal(t, 0, second, "eksekusi kedua tidak menemukan kandidat")
	assert.Len(t, store.appended, 1, "riwayat eskalasi tidak terduplikasi")
	assert.Len(t, audit.entries, 1)
	assert.Len(t, events.events, 1)
}

func TestEscalationWithoutCandidatesIsNoop(t *testing.T) {
	store := &escalationStoreStub{pending: map[uuid.UUID]bool{}}
	audit := &auditStub{}
	job := newEscalationJobForTest(store, audit, &eventRecorder{}, time.Now())

	count, err := job.Run(context.Background())

	require.NoError(t, err)
	assert.Zero(t, count)
	assert.Empty(t, audit.entries)
}

type photoStoreStub struct {
	claimed []domain.ExpiredPhoto
	cleared []uuid.UUID
	cutoff  time.Time
}

func (s *photoStoreStub) ClaimExpiredPhotos(
	_ context.Context, before time.Time, _ int,
) ([]domain.ExpiredPhoto, error) {
	s.cutoff = before
	remaining := make([]domain.ExpiredPhoto, 0)
	for _, photo := range s.claimed {
		alreadyCleared := false
		for _, id := range s.cleared {
			if id == photo.AttendanceID {
				alreadyCleared = true
			}
		}
		if !alreadyCleared {
			remaining = append(remaining, photo)
		}
	}
	return remaining, nil
}

func (s *photoStoreStub) ClearPhotoURL(_ context.Context, id uuid.UUID) error {
	s.cleared = append(s.cleared, id)
	return nil
}

type photoDeleterStub struct {
	deleted []string
	failOn  map[string]bool
}

func (s *photoDeleterStub) Delete(_ context.Context, objectPath string) error {
	if s.failOn[objectPath] {
		return errors.New("storage unavailable")
	}
	s.deleted = append(s.deleted, objectPath)
	return nil
}

func TestPhotoRetentionDeletesExpiredPhotosAndKeepsRows(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	store := &photoStoreStub{claimed: []domain.ExpiredPhoto{
		{AttendanceID: first, PhotoURL: "attendance-photos/a.jpg"},
		{AttendanceID: second, PhotoURL: "attendance-photos/b.jpg"},
	}}
	storage := &photoDeleterStub{failOn: map[string]bool{}}
	job := NewPhotoRetentionJob(store, storage, transactionStub{}, discardLogger())

	result, err := job.Run(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, result.Deleted)
	assert.Zero(t, result.Failed)
	assert.Len(t, storage.deleted, 2)
	// Baris absensi tidak dihapus; hanya locator yang dikosongkan.
	assert.ElementsMatch(t, []uuid.UUID{first, second}, store.cleared)
}

// URL hanya dikosongkan setelah berkas benar-benar terhapus.
func TestPhotoRetentionKeepsURLWhenStorageFails(t *testing.T) {
	failing, working := uuid.New(), uuid.New()
	store := &photoStoreStub{claimed: []domain.ExpiredPhoto{
		{AttendanceID: failing, PhotoURL: "attendance-photos/gagal.jpg"},
		{AttendanceID: working, PhotoURL: "attendance-photos/berhasil.jpg"},
	}}
	storage := &photoDeleterStub{failOn: map[string]bool{"attendance-photos/gagal.jpg": true}}
	job := NewPhotoRetentionJob(store, storage, transactionStub{}, discardLogger())

	result, err := job.Run(context.Background())

	require.NoError(t, err, "kegagalan satu berkas tidak menggagalkan batch")
	assert.Equal(t, 1, result.Deleted)
	assert.Equal(t, 1, result.Failed)
	assert.Equal(t, []uuid.UUID{working}, store.cleared,
		"berkas yang gagal dihapus harus mempertahankan foto_url untuk dicoba ulang")
}

// Berkas yang gagal pada eksekusi pertama diproses ulang pada eksekusi berikutnya.
func TestPhotoRetentionRetriesFailedPhotoOnNextRun(t *testing.T) {
	failing := uuid.New()
	store := &photoStoreStub{claimed: []domain.ExpiredPhoto{
		{AttendanceID: failing, PhotoURL: "attendance-photos/gagal.jpg"},
	}}
	storage := &photoDeleterStub{failOn: map[string]bool{"attendance-photos/gagal.jpg": true}}
	job := NewPhotoRetentionJob(store, storage, transactionStub{}, discardLogger())

	first, err := job.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, first.Failed)

	// Storage pulih; eksekusi berikutnya menyelesaikan berkas yang sama.
	storage.failOn = map[string]bool{}
	second, err := job.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, second.Deleted)
	assert.Equal(t, []uuid.UUID{failing}, store.cleared)
}

// Cutoff retensi adalah tiga bulan sebelum waktu eksekusi.
func TestPhotoRetentionUsesThreeMonthCutoff(t *testing.T) {
	now := time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC)
	store := &photoStoreStub{}
	job := NewPhotoRetentionJob(store, &photoDeleterStub{}, transactionStub{}, discardLogger())
	job.now = func() time.Time { return now }

	_, err := job.Run(context.Background())

	require.NoError(t, err)
	assert.Equal(t, now.Add(-domain.PhotoRetention), store.cutoff)
	assert.True(t, store.cutoff.Before(now.AddDate(0, -2, 0)), "cutoff minimal dua bulan lampau")
}

func TestPhotoRetentionWithoutExpiredPhotosIsNoop(t *testing.T) {
	store := &photoStoreStub{}
	storage := &photoDeleterStub{}
	job := NewPhotoRetentionJob(store, storage, transactionStub{}, discardLogger())

	result, err := job.Run(context.Background())

	require.NoError(t, err)
	assert.Zero(t, result.Deleted)
	assert.Zero(t, result.Failed)
	assert.Empty(t, storage.deleted)
}
