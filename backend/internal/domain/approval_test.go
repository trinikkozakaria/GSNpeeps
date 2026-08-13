package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Empat jalur routing pemohon sesuai alur approval CLAUDE.md bagian 9.
func TestInitialStatusForRoleCoversEveryRequesterPath(t *testing.T) {
	cases := []struct {
		name          string
		role          RoleName
		hasSupervisor bool
		expected      RequestStatus
		allowed       bool
	}{
		{"karyawan dengan atasan", RoleEmployee, true, StatusWaitingSupervisor, true},
		{"karyawan tanpa atasan", RoleEmployee, false, StatusWaitingHR, true},
		{"atasan", RoleSupervisor, true, StatusWaitingHR, true},
		{"atasan tanpa atasan", RoleSupervisor, false, StatusWaitingHR, true},
		{"hr", RoleHR, false, StatusWaitingTopManagement, true},
		{"top management tidak mengajukan", RoleTopManagement, false, "", false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			status, ok := InitialStatusForRole(testCase.role, testCase.hasSupervisor)

			assert.Equal(t, testCase.allowed, ok)
			assert.Equal(t, testCase.expected, status)
		})
	}
}

// Persetujuan Atasan memindahkan ke HR; persetujuan HR dan Top Management bersifat final.
func TestNextStatusAfterApprove(t *testing.T) {
	cases := []struct {
		current  RequestStatus
		expected RequestStatus
		ok       bool
	}{
		{StatusWaitingSupervisor, StatusWaitingHR, true},
		{StatusWaitingHR, StatusApproved, true},
		{StatusWaitingTopManagement, StatusApproved, true},
		{StatusApproved, "", false},
		{StatusRejected, "", false},
	}

	for _, testCase := range cases {
		next, ok := NextStatusAfterApprove(testCase.current)

		assert.Equalf(t, testCase.ok, ok, "status %s", testCase.current)
		assert.Equalf(t, testCase.expected, next, "status %s", testCase.current)
	}
}

// Hanya approver tahap aktif yang boleh memutus.
func TestCanDecideMatchesActiveStage(t *testing.T) {
	assert.True(t, CanDecide(RoleSupervisor, StatusWaitingSupervisor))
	assert.False(t, CanDecide(RoleHR, StatusWaitingSupervisor))
	assert.False(t, CanDecide(RoleTopManagement, StatusWaitingSupervisor))

	assert.True(t, CanDecide(RoleHR, StatusWaitingHR))
	assert.False(t, CanDecide(RoleSupervisor, StatusWaitingHR))
	assert.False(t, CanDecide(RoleEmployee, StatusWaitingHR))

	assert.True(t, CanDecide(RoleTopManagement, StatusWaitingTopManagement))
	assert.False(t, CanDecide(RoleHR, StatusWaitingTopManagement))

	// Pengajuan yang sudah final tidak dapat diputus siapa pun.
	for _, role := range []RoleName{RoleEmployee, RoleSupervisor, RoleHR, RoleTopManagement} {
		assert.Falsef(t, CanDecide(role, StatusApproved), "role %s", role)
		assert.Falsef(t, CanDecide(role, StatusRejected), "role %s", role)
	}
}

func TestPendingCoversWaitingStatusesOnly(t *testing.T) {
	assert.True(t, StatusWaitingSupervisor.Pending())
	assert.True(t, StatusWaitingHR.Pending())
	assert.True(t, StatusWaitingTopManagement.Pending())
	assert.False(t, StatusApproved.Pending())
	assert.False(t, StatusRejected.Pending())
	assert.False(t, StatusCancelled.Pending())
}

// Nilai keputusan database dipetakan ke enum response kontrak (D-025).
func TestApprovalHistoryMapsDatabaseDecisionToWireValue(t *testing.T) {
	moment := time.Date(2026, time.August, 3, 10, 0, 0, 0, time.UTC)
	approver := uuid.New()
	name := "Atasan Sintetis"

	assert.Equal(t, "disetujui",
		NewApprovalHistory(StageSupervisor, &approver, &name, DecisionApprove, nil, moment).Decision)
	assert.Equal(t, "ditolak",
		NewApprovalHistory(StageHR, &approver, &name, DecisionReject, nil, moment).Decision)
	assert.Equal(t, "didelegasikan",
		NewApprovalHistory(StageSupervisor, &approver, &name, DecisionDelegate, nil, moment).Decision)

	// Auto-escalation dipicu sistem sehingga tidak memiliki approver.
	system := NewApprovalHistory(StageSupervisor, nil, nil, DecisionAutoEscalate, nil, moment)
	assert.Equal(t, "auto_eskalasi", system.Decision)
	assert.Nil(t, system.ApproverID)
	assert.Nil(t, system.ApproverName)
}

func TestStageForStatus(t *testing.T) {
	stage, ok := StageForStatus(StatusWaitingSupervisor)
	require.True(t, ok)
	assert.Equal(t, StageSupervisor, stage)

	stage, ok = StageForStatus(StatusWaitingTopManagement)
	require.True(t, ok)
	assert.Equal(t, StageTopManagement, stage)

	_, ok = StageForStatus(StatusApproved)
	assert.False(t, ok)
}

func TestTotalLeaveDaysIsInclusive(t *testing.T) {
	start := time.Date(2026, time.August, 3, 0, 0, 0, 0, Jakarta())
	assert.Equal(t, 1, TotalLeaveDays(start, start))
	assert.Equal(t, 5, TotalLeaveDays(start, start.AddDate(0, 0, 4)))
}

// SLA eskalasi adalah 2x24 jam dan hanya berlaku dari Atasan ke HR.
func TestEscalationThresholdIsTwoDays(t *testing.T) {
	assert.Equal(t, 48*time.Hour, EscalationThreshold)
}
