package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// NotificationType adalah katalog event notifikasi yang disetujui. Katalog bersifat tertutup:
// nilai yang sama juga menjadi CHECK constraint pada tabel notifications.
type NotificationType string

const (
	NotificationLeaveSubmitted    NotificationType = "ketidakhadiran_baru"
	NotificationOvertimeSubmitted NotificationType = "lembur_baru"
	NotificationDecisionApproved  NotificationType = "keputusan_approve"
	NotificationDecisionRejected  NotificationType = "keputusan_reject"
	NotificationAutoEscalated     NotificationType = "auto_escalate"
	NotificationDelegated         NotificationType = "delegasi"
	NotificationContractExpiring  NotificationType = "kontrak_akan_habis"
)

func (t NotificationType) Valid() bool {
	switch t {
	case NotificationLeaveSubmitted, NotificationOvertimeSubmitted,
		NotificationDecisionApproved, NotificationDecisionRejected,
		NotificationAutoEscalated, NotificationDelegated, NotificationContractExpiring:
		return true
	default:
		return false
	}
}

// NotificationReference membatasi tujuan deep link ke resource internal yang dikenal.
// Backend tidak pernah menyimpan URL bebas; frontend membentuk tautan dari pasangan
// reference_type dan reference_id sehingga tujuan eksternal tidak mungkin muncul.
type NotificationReference string

const (
	ReferenceLeave    NotificationReference = "ketidakhadiran"
	ReferenceOvertime NotificationReference = "lembur"
	ReferenceEmployee NotificationReference = "karyawan"
)

// Notification adalah proyeksi response sesuai schema Notification pada OpenAPI.
type Notification struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	Type          string     `json:"tipe"`
	Title         string     `json:"judul"`
	Message       string     `json:"pesan"`
	ReferenceID   *uuid.UUID `json:"reference_id"`
	ReferenceType *string    `json:"reference_type"`
	IsRead        bool       `json:"is_read"`
	ReadAt        *time.Time `json:"read_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// NotificationPage adalah hasil list yang sudah dipaginasi.
type NotificationPage struct {
	Items []Notification
	Page  int
	Limit int
	Total int
}

// ExpiringContract adalah kontrak aktif yang masuk siklus pengingat H-30.
type ExpiringContract struct {
	ContractID uuid.UUID
	EmployeeID uuid.UUID
	// UserID adalah akun karyawan pemilik kontrak. Nilai ini dipakai untuk mengecualikan
	// subjek dari daftar penerima HR.
	UserID  uuid.UUID
	EndDate string
}

// NotificationDraft adalah notifikasi yang siap ditulis. Draft dibentuk server dari katalog
// event; tidak ada jalur yang menerima judul, pesan, maupun tujuan dari input pengguna.
type NotificationDraft struct {
	RecipientUserID uuid.UUID
	Type            NotificationType
	Title           string
	Message         string
	ReferenceID     *uuid.UUID
	ReferenceType   *NotificationReference
	EventKey        string
	CreatedAt       time.Time
}

// NotificationEventKey membentuk kunci idempotensi yang deterministik.
//
// Pola: <tipe>:<reference_id>:<cycle>:<recipient_user_id>
//
// `cycle` membedakan kemunculan yang sah dari event bertipe sama pada resource yang sama:
// tahap approval untuk event pengajuan, dan tanggal berakhir kontrak untuk pengingat H-30.
// Kunci sengaja tidak memuat judul, pesan, maupun timestamp agar retry producer menghasilkan
// kunci yang identik dan tertolak UNIQUE constraint.
func NotificationEventKey(
	notificationType NotificationType,
	referenceID uuid.UUID,
	cycle string,
	recipientUserID uuid.UUID,
) string {
	return strings.Join([]string{
		string(notificationType),
		referenceID.String(),
		cycle,
		recipientUserID.String(),
	}, ":")
}

// requestKindLabel memberi label Bahasa Indonesia untuk jenis pengajuan pada pesan.
func requestKindLabel(reference NotificationReference) string {
	if reference == ReferenceOvertime {
		return "lembur"
	}
	return "ketidakhadiran"
}

// SubmissionNotification memberi tahu approver pada tahap aktif bahwa ada pengajuan masuk.
func SubmissionNotification(
	recipientUserID uuid.UUID,
	reference NotificationReference,
	requestID uuid.UUID,
	stage ApprovalStage,
	occurredAt time.Time,
) NotificationDraft {
	notificationType := NotificationLeaveSubmitted
	if reference == ReferenceOvertime {
		notificationType = NotificationOvertimeSubmitted
	}
	kind := requestKindLabel(reference)
	return newRequestDraft(
		recipientUserID, notificationType, reference, requestID, string(stage), occurredAt,
		"Pengajuan "+kind+" baru",
		"Ada pengajuan "+kind+" yang menunggu keputusan Anda.",
	)
}

// DecisionNotification memberi tahu pemohon hasil keputusan pada satu tahap.
func DecisionNotification(
	recipientUserID uuid.UUID,
	reference NotificationReference,
	requestID uuid.UUID,
	stage ApprovalStage,
	status RequestStatus,
	occurredAt time.Time,
) NotificationDraft {
	kind := requestKindLabel(reference)
	if status == StatusRejected {
		return newRequestDraft(
			recipientUserID, NotificationDecisionRejected, reference, requestID,
			string(stage), occurredAt,
			"Pengajuan "+kind+" ditolak",
			"Pengajuan "+kind+" Anda ditolak. Catatan approver tersedia pada detail pengajuan.",
		)
	}
	message := "Pengajuan " + kind + " Anda telah disetujui."
	if status.Pending() {
		// Persetujuan tahap awal belum final; pemohon perlu tahu pengajuan berpindah tahap.
		message = "Pengajuan " + kind + " Anda disetujui pada tahap " +
			stageLabel(stage) + " dan diteruskan ke tahap berikutnya."
	}
	return newRequestDraft(
		recipientUserID, NotificationDecisionApproved, reference, requestID,
		string(stage), occurredAt,
		"Pengajuan "+kind+" disetujui", message,
	)
}

// DelegationNotification memberi tahu HR dan pemohon bahwa Atasan mendelegasikan keputusan.
func DelegationNotification(
	recipientUserID uuid.UUID,
	reference NotificationReference,
	requestID uuid.UUID,
	occurredAt time.Time,
) NotificationDraft {
	kind := requestKindLabel(reference)
	return newRequestDraft(
		recipientUserID, NotificationDelegated, reference, requestID,
		string(StageSupervisor), occurredAt,
		"Keputusan "+kind+" didelegasikan ke HR",
		"Atasan mendelegasikan keputusan pengajuan "+kind+" ini kepada HR.",
	)
}

// EscalationNotification memberi tahu HR dan pemohon bahwa SLA tahap Atasan terlampaui.
func EscalationNotification(
	recipientUserID uuid.UUID,
	reference NotificationReference,
	requestID uuid.UUID,
	occurredAt time.Time,
) NotificationDraft {
	kind := requestKindLabel(reference)
	return newRequestDraft(
		recipientUserID, NotificationAutoEscalated, reference, requestID,
		string(StageSupervisor), occurredAt,
		"Pengajuan "+kind+" dieskalasi otomatis",
		"Pengajuan "+kind+" melewati SLA 2x24 jam pada tahap Atasan dan diteruskan ke HR.",
	)
}

// ContractExpiringNotification memberi tahu pengingat kontrak H-30.
//
// Pesan tidak memuat nama, NIP, atau nomor kontrak. Penerima membuka detail karyawan melalui
// `reference_id` sehingga data personal tetap tunduk pada otorisasi endpoint karyawan.
func ContractExpiringNotification(
	recipientUserID uuid.UUID,
	employeeID uuid.UUID,
	contractID uuid.UUID,
	endDate string,
	occurredAt time.Time,
) NotificationDraft {
	reference := ReferenceEmployee
	return NotificationDraft{
		RecipientUserID: recipientUserID,
		Type:            NotificationContractExpiring,
		Title:           "Kontrak karyawan akan berakhir",
		Message: "Ada kontrak karyawan yang akan berakhir pada " + endDate +
			". Buka detail karyawan untuk menindaklanjuti.",
		ReferenceID:   &employeeID,
		ReferenceType: &reference,
		// Siklus memakai kontrak dan tanggal berakhirnya sehingga satu masa kontrak hanya
		// menghasilkan satu pengingat per penerima, berapa kali pun job dijalankan.
		EventKey: NotificationEventKey(
			NotificationContractExpiring, contractID, endDate, recipientUserID,
		),
		CreatedAt: occurredAt,
	}
}

func newRequestDraft(
	recipientUserID uuid.UUID,
	notificationType NotificationType,
	reference NotificationReference,
	requestID uuid.UUID,
	cycle string,
	occurredAt time.Time,
	title string,
	message string,
) NotificationDraft {
	referenceType := reference
	return NotificationDraft{
		RecipientUserID: recipientUserID,
		Type:            notificationType,
		Title:           title,
		Message:         message,
		ReferenceID:     &requestID,
		ReferenceType:   &referenceType,
		EventKey: NotificationEventKey(
			notificationType, requestID, cycle, recipientUserID,
		),
		CreatedAt: occurredAt,
	}
}

func stageLabel(stage ApprovalStage) string {
	switch stage {
	case StageSupervisor:
		return "Atasan"
	case StageHR:
		return "HR"
	case StageTopManagement:
		return "Top Management"
	default:
		return string(stage)
	}
}
