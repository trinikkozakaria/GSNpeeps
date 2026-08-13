package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func change(module, action string, allowed bool) PermissionChange {
	return PermissionChange{
		RoleID:    uuid.New(),
		Module:    module,
		Action:    action,
		IsAllowed: allowed,
	}
}

func TestValidatePermissionChangeRejectsUnknownCapability(t *testing.T) {
	cases := []PermissionChange{
		change("hiring", ActionRead, true),
		change(ModuleDashboard, ActionDelete, true),
		change(ModuleAudit, ActionUpdate, false),
	}

	for _, candidate := range cases {
		assert.ErrorIsf(t,
			ValidatePermissionChange(RoleHR, candidate), ErrInvalidRequest,
			"%s.%s harus ditolak", candidate.Module, candidate.Action,
		)
	}
}

// Top Management read-only: memberi mutation AKSES kepadanya melanggar kontrak.
func TestValidatePermissionChangeKeepsTopManagementReadOnly(t *testing.T) {
	require.ErrorIs(t,
		ValidatePermissionChange(RoleTopManagement, change(ModuleAccess, ActionUpdate, true)),
		ErrPermissionInvariant,
	)
	require.ErrorIs(t,
		ValidatePermissionChange(RoleTopManagement, change(ModuleEmployee, ActionDelete, true)),
		ErrPermissionInvariant,
	)
	require.ErrorIs(t,
		ValidatePermissionChange(RoleTopManagement, change(ModuleEmployee, ActionExport, true)),
		ErrPermissionInvariant,
	)

	// Baca, approval pengajuan HR, dan pengelolaan notifikasi sendiri tetap sah.
	assert.NoError(t,
		ValidatePermissionChange(RoleTopManagement, change(ModuleEmployee, ActionRead, true)))
	assert.NoError(t,
		ValidatePermissionChange(RoleTopManagement, change(ModuleLeave, ActionApprove, true)))
	assert.NoError(t,
		ValidatePermissionChange(RoleTopManagement, change(ModuleNotification, ActionDelete, true)))
}

func TestValidatePermissionChangeKeepsAccessModulesAwayFromStaff(t *testing.T) {
	for _, role := range []RoleName{RoleEmployee, RoleSupervisor} {
		assert.ErrorIs(t,
			ValidatePermissionChange(role, change(ModuleAccess, ActionRead, true)),
			ErrPermissionInvariant,
		)
		assert.ErrorIs(t,
			ValidatePermissionChange(role, change(ModuleAudit, ActionRead, true)),
			ErrPermissionInvariant,
		)
	}
}

// HR adalah satu-satunya role yang dapat memulihkan matriks; mencabut aksesnya sendiri akan
// mengunci modul AKSES secara permanen.
func TestValidatePermissionChangePreventsHumanResourcesLockout(t *testing.T) {
	assert.ErrorIs(t,
		ValidatePermissionChange(RoleHR, change(ModuleAccess, ActionUpdate, false)),
		ErrPermissionInvariant,
	)
	assert.ErrorIs(t,
		ValidatePermissionChange(RoleHR, change(ModuleAccess, ActionRead, false)),
		ErrPermissionInvariant,
	)

	// Pencabutan di modul lain tetap merupakan keputusan HR yang sah.
	assert.NoError(t,
		ValidatePermissionChange(RoleHR, change(ModuleAudit, ActionRead, false)))
}

func TestValidatePermissionChangeRejectsUnknownRole(t *testing.T) {
	assert.ErrorIs(t,
		ValidatePermissionChange(RoleName("auditor"), change(ModuleAudit, ActionRead, true)),
		ErrNotFound,
	)
}

func TestRoleDescriptionCoversEveryRole(t *testing.T) {
	for _, role := range []RoleName{RoleEmployee, RoleSupervisor, RoleHR, RoleTopManagement} {
		assert.NotEmptyf(t, RoleDescription(role), "deskripsi role %s wajib ada", role)
	}
}
