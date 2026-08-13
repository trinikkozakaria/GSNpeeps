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

// contractStoreStub meniru penyimpanan kontrak dan notifikasi, termasuk UNIQUE
// (recipient_user_id, event_key) yang menjadi dasar idempotensi job.
type contractStoreStub struct {
	byEndDate   map[string][]domain.ExpiringContract
	supervisors map[uuid.UUID]uuid.UUID
	byRole      map[domain.RoleName][]uuid.UUID
	stored      map[string]struct{}
	inserted    []domain.NotificationDraft
	claimedDate time.Time
	claimLimit  int
	insertErr   error
}

func newContractStoreStub() *contractStoreStub {
	return &contractStoreStub{
		byEndDate:   map[string][]domain.ExpiringContract{},
		supervisors: map[uuid.UUID]uuid.UUID{},
		byRole:      map[domain.RoleName][]uuid.UUID{},
		stored:      map[string]struct{}{},
	}
}

func (s *contractStoreStub) ClaimExpiringContracts(
	_ context.Context, endDate time.Time, limit int,
) ([]domain.ExpiringContract, error) {
	s.claimedDate, s.claimLimit = endDate, limit
	return s.byEndDate[endDate.Format(domain.DateLayout)], nil
}

func (s *contractStoreStub) SupervisorUserID(
	_ context.Context, employeeID uuid.UUID,
) (*uuid.UUID, error) {
	supervisor, ok := s.supervisors[employeeID]
	if !ok {
		return nil, nil
	}
	return &supervisor, nil
}

func (s *contractStoreStub) ActiveUserIDsByRole(
	_ context.Context, role domain.RoleName,
) ([]uuid.UUID, error) {
	return s.byRole[role], nil
}

func (s *contractStoreStub) Insert(
	_ context.Context, drafts []domain.NotificationDraft,
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

func (s *contractStoreStub) recipients() []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(s.inserted))
	for _, draft := range s.inserted {
		ids = append(ids, draft.RecipientUserID)
	}
	return ids
}

// runDate adalah tanggal acuan job pada Asia/Jakarta.
var runDate = time.Date(2026, 8, 3, 6, 30, 0, 0, domain.Jakarta())

func newContractJobForTest(store *contractStoreStub) *ContractExpiryJob {
	job := NewContractExpiryJob(store, transactionStub{}, slog.New(slog.DiscardHandler))
	job.now = func() time.Time { return runDate }
	return job
}

func contractFor(subject uuid.UUID, endDate string) domain.ExpiringContract {
	return domain.ExpiringContract{
		ContractID: uuid.New(),
		EmployeeID: uuid.New(),
		UserID:     subject,
		EndDate:    endDate,
	}
}

// Pemindaian memakai tanggal tepat H+30 pada zona Asia/Jakarta.
func TestContractExpiryScansExactlyThirtyCalendarDaysAhead(t *testing.T) {
	store := newContractStoreStub()

	_, err := newContractJobForTest(store).Run(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "2026-09-02", store.claimedDate.Format(domain.DateLayout))
	assert.Equal(t, domain.Jakarta().String(), store.claimedDate.Location().String())
	assert.Positive(t, store.claimLimit)
}

// Kontrak sebelum dan sesudah H-30 tidak masuk siklus hari ini.
func TestContractExpiryIgnoresContractsOutsideTheWindow(t *testing.T) {
	store := newContractStoreStub()
	store.byEndDate["2026-09-01"] = []domain.ExpiringContract{contractFor(uuid.New(), "2026-09-01")}
	store.byEndDate["2026-09-03"] = []domain.ExpiringContract{contractFor(uuid.New(), "2026-09-03")}

	result, err := newContractJobForTest(store).Run(context.Background())

	require.NoError(t, err)
	assert.Zero(t, result.Scanned)
	assert.Empty(t, store.inserted)
}

func TestContractExpiryNotifiesSupervisorAndEveryActiveHumanResources(t *testing.T) {
	store := newContractStoreStub()
	subject := uuid.New()
	supervisor := uuid.New()
	firstHR, secondHR := uuid.New(), uuid.New()

	contract := contractFor(subject, "2026-09-02")
	store.byEndDate["2026-09-02"] = []domain.ExpiringContract{contract}
	store.supervisors[contract.EmployeeID] = supervisor
	store.byRole[domain.RoleHR] = []uuid.UUID{firstHR, secondHR}
	store.byRole[domain.RoleTopManagement] = []uuid.UUID{uuid.New()}

	result, err := newContractJobForTest(store).Run(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, result.Scanned)
	assert.Equal(t, 3, result.Created)
	assert.ElementsMatch(t, []uuid.UUID{supervisor, firstHR, secondHR}, store.recipients())
}

// Subjek kontrak tidak pernah menerima pengingat kontraknya sendiri, bahkan ketika ia HR.
func TestContractExpiryExcludesTheContractSubject(t *testing.T) {
	store := newContractStoreStub()
	subject := uuid.New()
	otherHR := uuid.New()

	contract := contractFor(subject, "2026-09-02")
	store.byEndDate["2026-09-02"] = []domain.ExpiringContract{contract}
	store.byRole[domain.RoleHR] = []uuid.UUID{subject, otherHR}

	_, err := newContractJobForTest(store).Run(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{otherHR}, store.recipients())
}

// Tanpa HR aktif selain subjek, pengingat jatuh ke satu-satunya Top Management aktif.
func TestContractExpiryFallsBackToSingleTopManagement(t *testing.T) {
	store := newContractStoreStub()
	subject := uuid.New()
	topManagement := uuid.New()

	contract := contractFor(subject, "2026-09-02")
	store.byEndDate["2026-09-02"] = []domain.ExpiringContract{contract}
	store.byRole[domain.RoleHR] = []uuid.UUID{subject}
	store.byRole[domain.RoleTopManagement] = []uuid.UUID{topManagement}

	result, err := newContractJobForTest(store).Run(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{topManagement}, store.recipients())
	assert.Equal(t, 1, result.Created)
	assert.Zero(t, result.Failed)
}

// Fallback yang tidak tersedia adalah kegagalan yang diukur, bukan sukses diam-diam.
func TestContractExpiryMeasuresMissingRecipientAsFailure(t *testing.T) {
	store := newContractStoreStub()
	subject := uuid.New()
	store.byEndDate["2026-09-02"] = []domain.ExpiringContract{contractFor(subject, "2026-09-02")}

	result, err := newContractJobForTest(store).Run(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, result.Scanned)
	assert.Equal(t, 1, result.NoRecipient)
	assert.Equal(t, 1, result.Failed)
	assert.Zero(t, result.Created)
	assert.Empty(t, store.inserted)
}

// Run berulang pada hari yang sama tidak menambah baris.
func TestContractExpiryRepeatRunCreatesNothingNew(t *testing.T) {
	store := newContractStoreStub()
	contract := contractFor(uuid.New(), "2026-09-02")
	store.byEndDate["2026-09-02"] = []domain.ExpiringContract{contract}
	store.byRole[domain.RoleHR] = []uuid.UUID{uuid.New()}
	job := newContractJobForTest(store)

	first, err := job.Run(context.Background())
	require.NoError(t, err)
	second, err := job.Run(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, first.Created)
	assert.Zero(t, second.Created)
	assert.Equal(t, 1, second.Duplicate)
	assert.Len(t, store.inserted, 1)
}

// Dua replica yang berjalan bersamaan menghasilkan satu notifikasi per penerima.
func TestContractExpiryConcurrentRunsStayIdempotent(t *testing.T) {
	store := newContractStoreStub()
	contract := contractFor(uuid.New(), "2026-09-02")
	store.byEndDate["2026-09-02"] = []domain.ExpiringContract{contract}
	store.byRole[domain.RoleHR] = []uuid.UUID{uuid.New(), uuid.New()}

	for range 3 {
		_, err := newContractJobForTest(store).Run(context.Background())
		require.NoError(t, err)
	}

	assert.Len(t, store.inserted, 2)
}

// Satu kontrak yang gagal tidak boleh menghentikan sisa batch.
func TestContractExpiryContinuesAfterItemFailure(t *testing.T) {
	store := newContractStoreStub()
	failing := contractFor(uuid.New(), "2026-09-02")
	healthy := contractFor(uuid.New(), "2026-09-02")
	store.byEndDate["2026-09-02"] = []domain.ExpiringContract{failing, healthy}
	// Hanya kontrak kedua memiliki penerima; yang pertama gagal karena tidak ada penerima.
	store.supervisors[healthy.EmployeeID] = uuid.New()

	result, err := newContractJobForTest(store).Run(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, result.Scanned)
	assert.Equal(t, 1, result.Failed)
	assert.Equal(t, 1, result.Created)
}

func TestContractExpiryReportsStoreFailurePerItem(t *testing.T) {
	store := newContractStoreStub()
	store.byEndDate["2026-09-02"] = []domain.ExpiringContract{contractFor(uuid.New(), "2026-09-02")}
	store.byRole[domain.RoleHR] = []uuid.UUID{uuid.New()}
	store.insertErr = errors.New("insert gagal")

	result, err := newContractJobForTest(store).Run(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, result.Failed)
	assert.Zero(t, result.Created)
}

// Penerima yang muncul sebagai atasan sekaligus HR hanya dihitung sekali.
func TestContractExpiryDeduplicatesRecipientsByUserID(t *testing.T) {
	store := newContractStoreStub()
	shared := uuid.New()
	contract := contractFor(uuid.New(), "2026-09-02")
	store.byEndDate["2026-09-02"] = []domain.ExpiringContract{contract}
	store.supervisors[contract.EmployeeID] = shared
	store.byRole[domain.RoleHR] = []uuid.UUID{shared}

	result, err := newContractJobForTest(store).Run(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, []uuid.UUID{shared}, store.recipients())
}
