package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

// PhotoStore adalah kebutuhan penyimpanan job retensi foto absensi.
type PhotoStore interface {
	ClaimExpiredPhotos(context.Context, time.Time, int) ([]domain.ExpiredPhoto, error)
	ClearPhotoURL(context.Context, uuid.UUID) error
}

// PhotoDeleter menghapus objek dari storage.
type PhotoDeleter interface {
	Delete(context.Context, string) error
}

// PhotoRetentionJob menghapus foto absensi yang melewati masa simpan tiga bulan. Baris
// absensi tidak pernah dihapus; hanya `foto_url` yang dikosongkan.
type PhotoRetentionJob struct {
	store     PhotoStore
	storage   PhotoDeleter
	tx        TransactionManager
	logger    *slog.Logger
	batchSize int
	now       func() time.Time
}

func NewPhotoRetentionJob(
	store PhotoStore,
	storage PhotoDeleter,
	tx TransactionManager,
	logger *slog.Logger,
) *PhotoRetentionJob {
	return &PhotoRetentionJob{
		store:     store,
		storage:   storage,
		tx:        tx,
		logger:    logger,
		batchSize: 100,
		now:       time.Now,
	}
}

// PhotoRetentionResult adalah ringkasan agregat satu eksekusi.
type PhotoRetentionResult struct {
	Deleted int
	Failed  int
}

// Run memproses satu batch. URL hanya dikosongkan setelah berkas benar-benar terhapus,
// sehingga kegagalan storage menyisakan baris untuk dicoba ulang pada eksekusi berikutnya
// dan tidak pernah menghasilkan referensi yang hilang tanpa berkasnya terhapus.
func (j *PhotoRetentionJob) Run(ctx context.Context) (PhotoRetentionResult, error) {
	cutoff := j.now().UTC().Add(-domain.PhotoRetention)
	var result PhotoRetentionResult

	err := j.tx.Within(ctx, func(txContext context.Context) error {
		expired, err := j.store.ClaimExpiredPhotos(txContext, cutoff, j.batchSize)
		if err != nil {
			return err
		}
		for _, photo := range expired {
			if err := j.storage.Delete(txContext, photo.PhotoURL); err != nil {
				// Kegagalan satu berkas tidak menggagalkan batch; baris tetap memiliki
				// foto_url sehingga akan diklaim ulang pada eksekusi berikutnya.
				result.Failed++
				j.logger.WarnContext(txContext, "attendance photo delete failed",
					"attendance_id", photo.AttendanceID, "error", err)
				continue
			}
			if err := j.store.ClearPhotoURL(txContext, photo.AttendanceID); err != nil {
				return err
			}
			result.Deleted++
		}
		return nil
	})
	if err != nil {
		return PhotoRetentionResult{}, fmt.Errorf("photo retention: %w", err)
	}

	// Log agregat tanpa PII maupun locator berkas.
	j.logger.InfoContext(ctx, "photo retention job finished",
		"deleted", result.Deleted, "failed", result.Failed, "cutoff", cutoff)
	return result, nil
}
