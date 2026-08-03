package domain

import (
	"time"

	"github.com/google/uuid"
)

// ApprovalEventType adalah jenis domain event yang dipancarkan modul approval. Persistence
// dan pengiriman notifikasi dikerjakan pada epic BE.5; modul ini hanya menyatakan intent.
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
	// NextStage adalah tahap yang menjadi tujuan notifikasi approver berikutnya; kosong
	// ketika pengajuan sudah final.
	NextStage  *ApprovalStage `json:"next_stage"`
	OccurredAt time.Time      `json:"occurred_at"`
}

// ApprovalEventPublisher adalah boundary publikasi event. Implementasi sementara hanya
// mencatat event; BE.5 menggantinya dengan penulisan notifikasi idempotent tanpa mengubah
// pemanggil.
//
// Publish dipanggil di dalam transaction yang sama dengan perubahan status agar kegagalan
// publikasi ikut membatalkan keputusan dan tidak menghasilkan side effect yang hilang diam-diam.
type ApprovalEventPublisher interface {
	Publish(events ...ApprovalEvent) error
}

// NextStageForStatus mengembalikan tahap approver berikutnya untuk status menunggu.
func NextStageForStatus(status RequestStatus) *ApprovalStage {
	stage, ok := StageForStatus(status)
	if !ok {
		return nil
	}
	return &stage
}
