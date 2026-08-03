package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationEventKeyIsStableAcrossRetries(t *testing.T) {
	requestID := uuid.New()
	recipient := uuid.New()

	first := SubmissionNotification(
		recipient, ReferenceLeave, requestID, StageHR,
		time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC),
	)
	// Producer yang mengulang event yang sama beberapa jam kemudian harus menghasilkan kunci
	// identik; waktu tidak boleh menjadi komponen kunci.
	retry := SubmissionNotification(
		recipient, ReferenceLeave, requestID, StageHR,
		time.Date(2026, 8, 3, 17, 45, 0, 0, time.UTC),
	)

	assert.Equal(t, first.EventKey, retry.EventKey)
}

func TestNotificationEventKeyDistinguishesStageAndRecipient(t *testing.T) {
	requestID := uuid.New()
	supervisor := uuid.New()
	humanResources := uuid.New()
	occurredAt := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)

	supervisorStage := SubmissionNotification(
		supervisor, ReferenceLeave, requestID, StageSupervisor, occurredAt,
	)
	hrStage := SubmissionNotification(
		humanResources, ReferenceLeave, requestID, StageHR, occurredAt,
	)
	// Approver berbeda pada tahap berbeda adalah dua kemunculan yang sah.
	assert.NotEqual(t, supervisorStage.EventKey, hrStage.EventKey)

	sameStageOtherRecipient := SubmissionNotification(
		humanResources, ReferenceLeave, requestID, StageSupervisor, occurredAt,
	)
	assert.NotEqual(t, supervisorStage.EventKey, sameStageOtherRecipient.EventKey)
}

// Persetujuan tahap Atasan dan tahap HR atas pengajuan yang sama harus dapat memberi tahu
// pemohon dua kali; kunci karenanya membawa tahap pemutus.
func TestDecisionEventKeyDistinguishesDecidingStage(t *testing.T) {
	requestID := uuid.New()
	requester := uuid.New()
	occurredAt := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)

	bySupervisor := DecisionNotification(
		requester, ReferenceLeave, requestID, StageSupervisor, StatusWaitingHR, occurredAt,
	)
	byHR := DecisionNotification(
		requester, ReferenceLeave, requestID, StageHR, StatusApproved, occurredAt,
	)

	assert.NotEqual(t, bySupervisor.EventKey, byHR.EventKey)
	assert.Contains(t, bySupervisor.Message, "diteruskan ke tahap berikutnya")
	assert.Contains(t, byHR.Message, "telah disetujui")
}

func TestRejectionNotificationUsesRejectType(t *testing.T) {
	draft := DecisionNotification(
		uuid.New(), ReferenceOvertime, uuid.New(), StageHR, StatusRejected,
		time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC),
	)

	assert.Equal(t, NotificationDecisionRejected, draft.Type)
	assert.Contains(t, draft.Message, "lembur")
	// Catatan penolakan tidak disalin ke notifikasi; pesan hanya mengarahkan ke detail.
	assert.Contains(t, draft.Message, "detail pengajuan")
}

func TestContractExpiringEventKeyIdentifiesContractCycle(t *testing.T) {
	contractID := uuid.New()
	employeeID := uuid.New()
	recipient := uuid.New()
	occurredAt := time.Date(2026, 8, 3, 1, 0, 0, 0, time.UTC)

	first := ContractExpiringNotification(
		recipient, employeeID, contractID, "2026-09-02", occurredAt,
	)
	repeatRun := ContractExpiringNotification(
		recipient, employeeID, contractID, "2026-09-02", occurredAt.Add(24*time.Hour),
	)
	nextCycle := ContractExpiringNotification(
		recipient, employeeID, contractID, "2027-09-02", occurredAt,
	)

	assert.Equal(t, first.EventKey, repeatRun.EventKey)
	assert.NotEqual(t, first.EventKey, nextCycle.EventKey)

	// Pengingat menunjuk karyawan, bukan pengajuan, dan tidak memuat nama atau nomor kontrak.
	require.NotNil(t, first.ReferenceType)
	assert.Equal(t, ReferenceEmployee, *first.ReferenceType)
	require.NotNil(t, first.ReferenceID)
	assert.Equal(t, employeeID, *first.ReferenceID)
	assert.NotContains(t, first.Message, contractID.String())
}

// Deep link selalu dibentuk dari pasangan reference_type/reference_id internal; tidak ada
// jalur yang menyimpan URL.
func TestRequestNotificationsReferenceInternalResourcesOnly(t *testing.T) {
	requestID := uuid.New()
	occurredAt := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		draft NotificationDraft
		want  NotificationReference
	}{
		{"ketidakhadiran", SubmissionNotification(
			uuid.New(), ReferenceLeave, requestID, StageHR, occurredAt,
		), ReferenceLeave},
		{"lembur", SubmissionNotification(
			uuid.New(), ReferenceOvertime, requestID, StageHR, occurredAt,
		), ReferenceOvertime},
		{"delegasi", DelegationNotification(
			uuid.New(), ReferenceLeave, requestID, occurredAt,
		), ReferenceLeave},
		{"eskalasi", EscalationNotification(
			uuid.New(), ReferenceLeave, requestID, occurredAt,
		), ReferenceLeave},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			require.NotNil(t, testCase.draft.ReferenceType)
			assert.Equal(t, testCase.want, *testCase.draft.ReferenceType)
			require.NotNil(t, testCase.draft.ReferenceID)
			assert.Equal(t, requestID, *testCase.draft.ReferenceID)
			assert.True(t, testCase.draft.Type.Valid())
		})
	}
}

func TestSubmissionNotificationTypeFollowsResource(t *testing.T) {
	occurredAt := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)

	leave := SubmissionNotification(uuid.New(), ReferenceLeave, uuid.New(), StageHR, occurredAt)
	overtime := SubmissionNotification(
		uuid.New(), ReferenceOvertime, uuid.New(), StageHR, occurredAt,
	)

	assert.Equal(t, NotificationLeaveSubmitted, leave.Type)
	assert.Equal(t, NotificationOvertimeSubmitted, overtime.Type)
}
