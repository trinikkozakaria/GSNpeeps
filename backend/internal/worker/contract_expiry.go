package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

// ContractLeadDays adalah jarak pengingat kontrak: tepat 30 hari kalender sebelum tanggal
// berakhir, dihitung pada zona Asia/Jakarta.
const ContractLeadDays = 30

// ContractNotificationStore adalah kebutuhan penyimpanan job pengingat kontrak.
type ContractNotificationStore interface {
	ClaimExpiringContracts(context.Context, time.Time, int) ([]domain.ExpiringContract, error)
	SupervisorUserID(context.Context, uuid.UUID) (*uuid.UUID, error)
	ActiveUserIDsByRole(context.Context, domain.RoleName) ([]uuid.UUID, error)
	Insert(context.Context, []domain.NotificationDraft) (int, error)
}

// ContractExpiryResult adalah metrik satu run.
type ContractExpiryResult struct {
	Scanned     int
	Created     int
	Duplicate   int
	NoRecipient int
	Failed      int
}

// ContractExpiryJob mengirim pengingat kontrak H-30.
//
// Idempotensi berasal dari event key deterministik `kontrak_akan_habis:<kontrak>:<tanggal
// berakhir>:<penerima>` dan UNIQUE (recipient_user_id, event_key). Karena itu run berulang
// pada hari yang sama, maupun dua replica yang berjalan bersamaan, tidak menambah baris dan
// tidak memerlukan lock terdistribusi untuk menjaga kebenaran.
type ContractExpiryJob struct {
	store     ContractNotificationStore
	tx        TransactionManager
	logger    *slog.Logger
	batchSize int
	now       func() time.Time
}

func NewContractExpiryJob(
	store ContractNotificationStore,
	tx TransactionManager,
	logger *slog.Logger,
) *ContractExpiryJob {
	return &ContractExpiryJob{
		store:     store,
		tx:        tx,
		logger:    logger,
		batchSize: 200,
		now:       time.Now,
	}
}

// WithClock mengganti sumber waktu job. Tanggal acuan menentukan siklus H-30, sehingga test
// memerlukan jam yang deterministik agar tidak bergantung pada hari eksekusi.
func (j *ContractExpiryJob) WithClock(clock func() time.Time) *ContractExpiryJob {
	j.now = clock
	return j
}

func (j *ContractExpiryJob) Run(ctx context.Context) (ContractExpiryResult, error) {
	runID := uuid.NewString()
	started := j.now()

	// Tanggal acuan dihitung pada Asia/Jakarta agar batas hari mengikuti kalender pengguna,
	// bukan zona server.
	today := j.now().In(domain.Jakarta())
	target := time.Date(
		today.Year(), today.Month(), today.Day()+ContractLeadDays,
		0, 0, 0, 0, domain.Jakarta(),
	)

	contracts, err := j.store.ClaimExpiringContracts(ctx, target, j.batchSize)
	if err != nil {
		return ContractExpiryResult{}, fmt.Errorf("contract expiry: %w", err)
	}

	result := ContractExpiryResult{Scanned: len(contracts)}
	for _, contract := range contracts {
		if err := j.notify(ctx, contract, &result); err != nil {
			// Satu kontrak yang gagal tidak boleh menghentikan sisa batch; item dihitung
			// gagal dan run berikutnya mencobanya lagi karena tidak ada baris yang ditulis.
			result.Failed++
			j.logger.ErrorContext(ctx, "contract expiry item failed",
				"job_run_id", runID, "contract_id", contract.ContractID, "error", err)
		}
	}

	// Log agregat tanpa nama, NIP, maupun nomor kontrak.
	j.logger.InfoContext(ctx, "contract expiry job finished",
		"job_run_id", runID,
		"target_date", target.Format(domain.DateLayout),
		"scanned", result.Scanned,
		"created", result.Created,
		"duplicate", result.Duplicate,
		"no_recipient", result.NoRecipient,
		"failed", result.Failed,
		"duration_ms", j.now().Sub(started).Milliseconds(),
	)
	return result, nil
}

func (j *ContractExpiryJob) notify(
	ctx context.Context,
	contract domain.ExpiringContract,
	result *ContractExpiryResult,
) error {
	recipients, err := j.recipients(ctx, contract)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		// Invariant produk mensyaratkan minimal satu penerima. Kondisi ini ditandai sebagai
		// gagal, bukan sukses, dan tidak pernah dialihkan ke subjek kontrak sendiri.
		result.NoRecipient++
		return fmt.Errorf("no eligible recipient for contract %s", contract.ContractID)
	}

	drafts := make([]domain.NotificationDraft, 0, len(recipients))
	for _, recipient := range recipients {
		drafts = append(drafts, domain.ContractExpiringNotification(
			recipient, contract.EmployeeID, contract.ContractID, contract.EndDate, j.now().UTC(),
		))
	}

	created := 0
	err = j.tx.Within(ctx, func(txContext context.Context) error {
		inserted, err := j.store.Insert(txContext, drafts)
		created = inserted
		return err
	})
	if err != nil {
		return err
	}
	result.Created += created
	result.Duplicate += len(drafts) - created
	return nil
}

// recipients menyusun penerima pengingat: atasan langsung aktif bila ada, ditambah seluruh HR
// aktif selain subjek kontrak. Bila tidak ada HR yang memenuhi syarat, satu-satunya Top
// Management aktif menjadi fallback. Subjek kontrak tidak pernah menerima pengingatnya
// sendiri, dan penerima dideduplikasi berdasarkan user ID sebelum event key dibentuk.
func (j *ContractExpiryJob) recipients(
	ctx context.Context,
	contract domain.ExpiringContract,
) ([]uuid.UUID, error) {
	ordered := []uuid.UUID{}
	seen := map[uuid.UUID]struct{}{contract.UserID: {}}

	add := func(candidate uuid.UUID) {
		if _, duplicate := seen[candidate]; duplicate {
			return
		}
		seen[candidate] = struct{}{}
		ordered = append(ordered, candidate)
	}

	supervisor, err := j.store.SupervisorUserID(ctx, contract.EmployeeID)
	if err != nil {
		return nil, fmt.Errorf("resolve contract supervisor: %w", err)
	}
	if supervisor != nil {
		add(*supervisor)
	}

	humanResources, err := j.store.ActiveUserIDsByRole(ctx, domain.RoleHR)
	if err != nil {
		return nil, fmt.Errorf("resolve active hr: %w", err)
	}
	eligibleHR := 0
	for _, candidate := range humanResources {
		if candidate == contract.UserID {
			continue
		}
		eligibleHR++
		add(candidate)
	}
	if eligibleHR > 0 {
		return ordered, nil
	}

	// Fallback hanya berlaku ketika tidak ada satu pun HR aktif selain subjek.
	topManagement, err := j.store.ActiveUserIDsByRole(ctx, domain.RoleTopManagement)
	if err != nil {
		return nil, fmt.Errorf("resolve active top management: %w", err)
	}
	for _, candidate := range topManagement {
		if candidate == contract.UserID {
			continue
		}
		add(candidate)
	}
	return ordered, nil
}
