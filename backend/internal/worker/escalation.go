// Package worker berisi job terjadwal yang berjalan pada container terpisah dengan
// codebase yang sama seperti API.
package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

// EscalationStore adalah kebutuhan penyimpanan job auto-escalation. Kontrak ini dipenuhi
// oleh repository ketidakhadiran maupun lembur sehingga satu job melayani keduanya.
type EscalationStore interface {
	ClaimEscalatableRequests(context.Context, time.Time, int) ([]domain.EscalationCandidate, error)
	UpdateRequestStatus(context.Context, uuid.UUID, domain.RequestStatus, domain.RequestStatus) error
	AppendApproval(
		context.Context, uuid.UUID, domain.ApprovalStage, *uuid.UUID,
		domain.ApprovalDecision, *string,
	) error
}

type TransactionManager interface {
	Within(context.Context, func(context.Context) error) error
}

type AuditWriter interface {
	Append(context.Context, domain.AuditEntry) error
}

// EscalationJob memindahkan pengajuan tahap Atasan yang melewati SLA 2x24 jam ke HR.
// Tidak ada escalation dari HR ke Top Management.
type EscalationJob struct {
	name      string
	store     EscalationStore
	tx        TransactionManager
	audit     AuditWriter
	events    domain.ApprovalEventPublisher
	eventType domain.ApprovalEventType
	logger    *slog.Logger
	batchSize int
	now       func() time.Time
}

func NewEscalationJob(
	name string,
	store EscalationStore,
	tx TransactionManager,
	audit AuditWriter,
	events domain.ApprovalEventPublisher,
	eventType domain.ApprovalEventType,
	logger *slog.Logger,
) *EscalationJob {
	return &EscalationJob{
		name:      name,
		store:     store,
		tx:        tx,
		audit:     audit,
		events:    events,
		eventType: eventType,
		logger:    logger,
		batchSize: 100,
		now:       time.Now,
	}
}

// Run memproses satu batch. Job aman dijalankan ulang maupun bersamaan: kandidat diklaim
// dengan SKIP LOCKED dan perubahan status bersifat kondisional, sehingga pengajuan yang
// sudah diputus approver di antara klaim dan update tidak akan tereskalasi.
func (j *EscalationJob) Run(ctx context.Context) (int, error) {
	threshold := j.now().UTC().Add(-domain.EscalationThreshold)
	escalated := 0

	err := j.tx.Within(ctx, func(txContext context.Context) error {
		candidates, err := j.store.ClaimEscalatableRequests(txContext, threshold, j.batchSize)
		if err != nil {
			return err
		}
		for _, candidate := range candidates {
			if err := j.escalate(txContext, candidate); err != nil {
				return err
			}
			escalated++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("%s: %w", j.name, err)
	}

	// Log agregat tanpa identitas karyawan.
	j.logger.InfoContext(ctx, "escalation job finished",
		"job", j.name, "escalated", escalated, "threshold", threshold)
	return escalated, nil
}

func (j *EscalationJob) escalate(
	ctx context.Context,
	candidate domain.EscalationCandidate,
) error {
	err := j.store.UpdateRequestStatus(
		ctx, candidate.RequestID, domain.StatusWaitingSupervisor, domain.StatusWaitingHR,
	)
	if err != nil {
		return err
	}
	// approver_id NULL menandai keputusan dipicu sistem.
	if err := j.store.AppendApproval(
		ctx, candidate.RequestID, domain.StageSupervisor, nil,
		domain.DecisionAutoEscalate, nil,
	); err != nil {
		return err
	}
	if err := j.audit.Append(ctx, domain.AuditEntry{
		// UserID nil menandai aktor sistem, bukan pengguna.
		UserID: nil,
		Action: "AUTO_ESCALATE",
		Module: j.name,
		DataID: &candidate.RequestID,
		Detail: map[string]any{
			"status_baru": string(domain.StatusWaitingHR),
			"alasan":      "SLA atasan 2x24 jam terlampaui",
		},
		CreatedAt: j.now().UTC(),
	}); err != nil {
		return err
	}
	stage := domain.StageHR
	return j.events.Publish(domain.ApprovalEvent{
		Type:            j.eventType,
		RequestID:       candidate.RequestID,
		RequesterUserID: candidate.RequesterUserID,
		ActorUserID:     nil,
		Status:          domain.StatusWaitingHR,
		NextStage:       &stage,
		OccurredAt:      j.now().UTC(),
	})
}
