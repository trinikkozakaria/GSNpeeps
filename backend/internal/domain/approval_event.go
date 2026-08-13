package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ApprovalEventType adalah jenis domain event yang dipancarkan modul approval. Consumer
// pada BE.5 menerjemahkannya menjadi notifikasi idempotent.
type ApprovalEventType string

const (
	EventLeaveSubmitted          ApprovalEventType = "LeaveSubmitted"
	EventLeaveDecisionChanged    ApprovalEventType = "LeaveDecisionChanged"
	EventLeaveDelegated          ApprovalEventType = "LeaveDelegated"
	EventLeaveAutoEscalated      ApprovalEventType = "LeaveAutoEscalated"
	EventOvertimeSubmitted       ApprovalEventType = "OvertimeSubmitted"
	EventOvertimeDecisionChanged ApprovalEventType = "OvertimeDecisionChanged"
	EventOvertimeAutoEscalated   ApprovalEventType = "OvertimeAutoEscalated"
)

// ApprovalEvent memuat identifier dan aktor minimum. Event sengaja tidak membawa alasan,
// dokumen, koordinat, atau data personal lain agar tidak menyebarkan PII ke kanal
// notifikasi. Konsumen mengambil detail yang diizinkan melalui repository saat dibutuhkan.
type ApprovalEvent struct {
	Type ApprovalEventType `json:"type"`
	// RequestID adalah leave_request_id atau overtime_request_id.
	RequestID uuid.UUID `json:"request_id"`
	// RequesterUserID adalah pemilik pengajuan yang harus diberi tahu perubahan status.
	RequesterUserID uuid.UUID `json:"requester_user_id"`
	// ActorUserID kosong ketika perubahan dipicu sistem melalui auto-escalation.
	ActorUserID *uuid.UUID `json:"actor_user_id"`
	// Status adalah status pengajuan setelah perubahan.
	Status RequestStatus `json:"status"`
	// Stage adalah tahap tempat event terjadi: tahap awal untuk pengajuan baru, dan tahap
	// pemutus untuk keputusan, delegasi, maupun eskalasi. Nilai ini menjadi komponen siklus
	// pada event key sehingga persetujuan Atasan dan persetujuan HR atas pengajuan yang sama
	// menghasilkan notifikasi berbeda, sementara retry tahap yang sama tetap terdeduplikasi.
	Stage ApprovalStage `json:"stage"`
	// NextStage adalah tahap yang menjadi tujuan notifikasi approver berikutnya; kosong
	// ketika pengajuan sudah final.
	NextStage  *ApprovalStage `json:"next_stage"`
	OccurredAt time.Time      `json:"occurred_at"`
}

// Reference memetakan jenis event ke resource tujuan deep link notifikasi.
func (e ApprovalEvent) Reference() NotificationReference {
	switch e.Type {
	case EventOvertimeSubmitted, EventOvertimeDecisionChanged, EventOvertimeAutoEscalated:
		return ReferenceOvertime
	default:
		return ReferenceLeave
	}
}

// ApprovalEventPublisher adalah boundary publikasi event.
//
// Publish menerima context transaction pemanggil dan dieksekusi di dalam transaction yang
// sama dengan perubahan status. Kegagalan publikasi membatalkan keputusan, sehingga tidak ada
// event yang hilang diam-diam maupun notifikasi yang menggambarkan status yang tidak jadi
// tersimpan (D-033).
type ApprovalEventPublisher interface {
	Publish(ctx context.Context, events ...ApprovalEvent) error
}

// NextStageForStatus mengembalikan tahap approver berikutnya untuk status menunggu.
func NextStageForStatus(status RequestStatus) *ApprovalStage {
	stage, ok := StageForStatus(status)
	if !ok {
		return nil
	}
	return &stage
}
