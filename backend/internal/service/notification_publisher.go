package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

// NotificationWriter adalah kebutuhan penulisan notifikasi dari sisi producer event.
type NotificationWriter interface {
	Insert(context.Context, []domain.NotificationDraft) (int, error)
	ApproverUserIDs(context.Context, domain.ApprovalStage, uuid.UUID) ([]uuid.UUID, error)
}

// NotificationPublisher menerjemahkan domain event approval menjadi notifikasi.
//
// Publisher berjalan di dalam transaction pemanggil (D-033): notifikasi tersimpan bersama
// perubahan status, dan kegagalan penulisan membatalkan keputusan. Tidak ada antrean
// in-memory yang dapat hilang saat proses berhenti.
type NotificationPublisher struct {
	store  NotificationWriter
	logger *slog.Logger
}

func NewNotificationPublisher(store NotificationWriter, logger *slog.Logger) *NotificationPublisher {
	return &NotificationPublisher{store: store, logger: logger}
}

func (p *NotificationPublisher) Publish(
	ctx context.Context,
	events ...domain.ApprovalEvent,
) error {
	drafts := []domain.NotificationDraft{}
	seen := map[string]struct{}{}

	for _, event := range events {
		eventDrafts, err := p.draftsFor(ctx, event)
		if err != nil {
			return err
		}
		for _, draft := range eventDrafts {
			// Deduplikasi dalam satu batch agar hitungan yang dicatat log jujur; database
			// tetap menjadi penegak idempotensi lintas batch dan lintas proses.
			if _, duplicate := seen[draft.EventKey]; duplicate {
				continue
			}
			seen[draft.EventKey] = struct{}{}
			drafts = append(drafts, draft)
		}
	}
	if len(drafts) == 0 {
		return nil
	}

	created, err := p.store.Insert(ctx, drafts)
	if err != nil {
		return fmt.Errorf("write approval notifications: %w", err)
	}
	// Log agregat tanpa judul, pesan, maupun identitas penerima.
	p.logger.InfoContext(ctx, "approval notifications written",
		"drafted", len(drafts), "created", created, "duplicate", len(drafts)-created)
	return nil
}

func (p *NotificationPublisher) draftsFor(
	ctx context.Context,
	event domain.ApprovalEvent,
) ([]domain.NotificationDraft, error) {
	reference := event.Reference()

	switch event.Type {
	case domain.EventLeaveSubmitted, domain.EventOvertimeSubmitted:
		return p.approverDrafts(ctx, event, reference, event.Stage)

	case domain.EventLeaveDecisionChanged, domain.EventOvertimeDecisionChanged:
		// Pemohon selalu diberi tahu hasil tahap yang baru saja diputus.
		drafts := []domain.NotificationDraft{domain.DecisionNotification(
			event.RequesterUserID, reference, event.RequestID,
			event.Stage, event.Status, event.OccurredAt,
		)}
		if event.NextStage == nil {
			return drafts, nil
		}
		// Pengajuan berpindah tahap; approver berikutnya menerima notifikasi masuk inbox.
		next, err := p.approverDrafts(ctx, event, reference, *event.NextStage)
		if err != nil {
			return nil, err
		}
		return append(drafts, next...), nil

	case domain.EventLeaveDelegated:
		return p.handoffDrafts(ctx, event, reference, domain.DelegationNotification)

	case domain.EventLeaveAutoEscalated, domain.EventOvertimeAutoEscalated:
		return p.handoffDrafts(ctx, event, reference, domain.EscalationNotification)

	default:
		return nil, nil
	}
}

// approverDrafts membuat notifikasi untuk approver aktif pada satu tahap.
func (p *NotificationPublisher) approverDrafts(
	ctx context.Context,
	event domain.ApprovalEvent,
	reference domain.NotificationReference,
	stage domain.ApprovalStage,
) ([]domain.NotificationDraft, error) {
	recipients, err := p.store.ApproverUserIDs(ctx, stage, event.RequesterUserID)
	if err != nil {
		return nil, fmt.Errorf("resolve approver recipients: %w", err)
	}
	drafts := []domain.NotificationDraft{}
	for _, recipient := range recipients {
		if skipSelfNotify(event, recipient) {
			continue
		}
		drafts = append(drafts, domain.SubmissionNotification(
			recipient, reference, event.RequestID, stage, event.OccurredAt,
		))
	}
	return drafts, nil
}

// handoffDrafts membuat notifikasi delegasi atau eskalasi: seluruh HR aktif sebagai penerima
// keputusan, ditambah pemohon sebagai informasi perpindahan tahap.
func (p *NotificationPublisher) handoffDrafts(
	ctx context.Context,
	event domain.ApprovalEvent,
	reference domain.NotificationReference,
	build func(
		uuid.UUID, domain.NotificationReference, uuid.UUID, time.Time,
	) domain.NotificationDraft,
) ([]domain.NotificationDraft, error) {
	recipients, err := p.store.ApproverUserIDs(ctx, domain.StageHR, event.RequesterUserID)
	if err != nil {
		return nil, fmt.Errorf("resolve handoff recipients: %w", err)
	}

	drafts := []domain.NotificationDraft{}
	for _, recipient := range recipients {
		if skipSelfNotify(event, recipient) {
			continue
		}
		drafts = append(drafts, build(recipient, reference, event.RequestID, event.OccurredAt))
	}
	return append(drafts, build(
		event.RequesterUserID, reference, event.RequestID, event.OccurredAt,
	)), nil
}

// skipSelfNotify mencegah approver menerima notifikasi atas aksinya sendiri dan mencegah
// pemohon menerima notifikasi "pengajuan masuk" untuk pengajuannya sendiri.
func skipSelfNotify(event domain.ApprovalEvent, recipient uuid.UUID) bool {
	if recipient == event.RequesterUserID {
		return true
	}
	return event.ActorUserID != nil && recipient == *event.ActorUserID
}
