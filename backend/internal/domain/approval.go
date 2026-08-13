package domain

import (
	"time"

	"github.com/google/uuid"
)

// RequestStatus adalah enum machine-readable yang sama di database, query, dan response
// sesuai keputusan D-005 dan D-021.
type RequestStatus string

const (
	StatusWaitingSupervisor    RequestStatus = "menunggu_atasan"
	StatusWaitingHR            RequestStatus = "menunggu_hr"
	StatusWaitingTopManagement RequestStatus = "menunggu_top_management"
	StatusApproved             RequestStatus = "disetujui"
	StatusRejected             RequestStatus = "ditolak"
	StatusCancelled            RequestStatus = "dibatalkan"
)

func (s RequestStatus) Valid() bool {
	switch s {
	case StatusWaitingSupervisor, StatusWaitingHR, StatusWaitingTopManagement,
		StatusApproved, StatusRejected, StatusCancelled:
		return true
	default:
		return false
	}
}

// Pending menyatakan status masih menunggu keputusan pada salah satu tahap.
func (s RequestStatus) Pending() bool {
	return s == StatusWaitingSupervisor || s == StatusWaitingHR || s == StatusWaitingTopManagement
}

// ApprovalStage adalah tahap approval yang tersimpan pada riwayat.
type ApprovalStage string

const (
	StageSupervisor    ApprovalStage = "atasan"
	StageHR            ApprovalStage = "hr"
	StageTopManagement ApprovalStage = "top_management"
)

// StageForStatus memetakan status menunggu ke tahap yang sedang aktif.
func StageForStatus(status RequestStatus) (ApprovalStage, bool) {
	switch status {
	case StatusWaitingSupervisor:
		return StageSupervisor, true
	case StatusWaitingHR:
		return StageHR, true
	case StatusWaitingTopManagement:
		return StageTopManagement, true
	default:
		return "", false
	}
}

// ApprovalDecision adalah nilai keputusan yang tersimpan di database.
type ApprovalDecision string

const (
	DecisionApprove      ApprovalDecision = "approve"
	DecisionReject       ApprovalDecision = "reject"
	DecisionDelegate     ApprovalDecision = "delegate"
	DecisionAutoEscalate ApprovalDecision = "auto_escalate"
)

// wireDecision memetakan nilai database ke enum response ApprovalHistory (D-025).
var wireDecision = map[ApprovalDecision]string{
	DecisionApprove:      "disetujui",
	DecisionReject:       "ditolak",
	DecisionDelegate:     "didelegasikan",
	DecisionAutoEscalate: "auto_eskalasi",
}

// ApprovalHistory memetakan schema ApprovalHistory pada OpenAPI. `approver_id` dan
// `approver_nama` kosong ketika keputusan dipicu sistem.
type ApprovalHistory struct {
	Stage        ApprovalStage `json:"tahap"`
	ApproverID   *uuid.UUID    `json:"approver_id"`
	ApproverName *string       `json:"approver_nama"`
	Decision     string        `json:"keputusan"`
	Note         *string       `json:"catatan"`
	DecidedAt    time.Time     `json:"decided_at"`
}

// NewApprovalHistory membentuk entri riwayat dengan nilai keputusan versi response.
func NewApprovalHistory(
	stage ApprovalStage,
	approverID *uuid.UUID,
	approverName *string,
	decision ApprovalDecision,
	note *string,
	decidedAt time.Time,
) ApprovalHistory {
	return ApprovalHistory{
		Stage:        stage,
		ApproverID:   approverID,
		ApproverName: approverName,
		Decision:     wireDecision[decision],
		Note:         note,
		DecidedAt:    decidedAt,
	}
}

// RequestStateData adalah payload response untuk create dan decision.
type RequestStateData struct {
	ID     uuid.UUID     `json:"id"`
	Status RequestStatus `json:"status"`
}

// DecisionInput adalah keputusan yang dikirim approver.
type DecisionInput struct {
	Approve bool
	Note    *string
}

// InitialStatusForRole menentukan status awal pengajuan berdasarkan role pemohon dan
// keberadaan atasan langsung, sesuai alur approval CLAUDE.md bagian 9:
//
//	Karyawan dengan atasan -> menunggu_atasan
//	Karyawan tanpa atasan  -> menunggu_hr
//	Atasan                 -> menunggu_hr
//	HR                     -> menunggu_top_management
//
// Top Management tidak mengajukan; kontrak tidak menyediakan jalur approval untuknya.
func InitialStatusForRole(role RoleName, hasSupervisor bool) (RequestStatus, bool) {
	switch role {
	case RoleEmployee:
		if hasSupervisor {
			return StatusWaitingSupervisor, true
		}
		return StatusWaitingHR, true
	case RoleSupervisor:
		return StatusWaitingHR, true
	case RoleHR:
		return StatusWaitingTopManagement, true
	default:
		return "", false
	}
}

// NextStatusAfterApprove menentukan status setelah persetujuan pada tahap aktif. Persetujuan
// Atasan memindahkan pengajuan ke HR; persetujuan HR dan Top Management bersifat final.
func NextStatusAfterApprove(current RequestStatus) (RequestStatus, bool) {
	switch current {
	case StatusWaitingSupervisor:
		return StatusWaitingHR, true
	case StatusWaitingHR, StatusWaitingTopManagement:
		return StatusApproved, true
	default:
		return "", false
	}
}

// CanDecide menyatakan apakah role boleh memutus pengajuan pada status aktif tertentu.
// Top Management hanya memutus pengajuan milik HR pada tahap Top Management.
func CanDecide(role RoleName, status RequestStatus) bool {
	switch status {
	case StatusWaitingSupervisor:
		return role == RoleSupervisor
	case StatusWaitingHR:
		return role == RoleHR
	case StatusWaitingTopManagement:
		return role == RoleTopManagement
	default:
		return false
	}
}

// RequestLock adalah state pengajuan yang dibaca di bawah row lock sebelum keputusan
// diterapkan. Nilai kuota dan tahun dipakai untuk pengurangan saldo pada final approval.
type RequestLock struct {
	RequestID            uuid.UUID
	RequesterUserID      uuid.UUID
	RequesterEmployeeID  uuid.UUID
	SupervisorEmployeeID *uuid.UUID
	Status               RequestStatus
	TotalDays            int
	AnnualQuota          int
	LeaveCategory        string
	LeaveTypeID          uuid.UUID
	Year                 int
}

// EscalationCandidate adalah pengajuan yang melewati SLA tahap Atasan.
type EscalationCandidate struct {
	RequestID       uuid.UUID
	RequesterUserID uuid.UUID
}

// EscalationThreshold adalah SLA 2x24 jam dari tahap Atasan ke HR. Tidak ada escalation
// dari HR ke Top Management.
const EscalationThreshold = 48 * time.Hour
