package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetReaderStub memakai employeeReaderStub bersama untuk 15 method lain dan hanya
// meng-override resolusi user id serta penyimpanan password baru.
type resetReaderStub struct {
	*employeeReaderStub
	resolvedUserID  uuid.UUID
	resolveErr      error
	setPasswordUser uuid.UUID
	setPasswordHash string
	setPasswordErr  error
	setPasswordHits int
}

func (s *resetReaderStub) ResolveUserID(context.Context, uuid.UUID) (uuid.UUID, error) {
	if s.resolveErr != nil {
		return uuid.Nil, s.resolveErr
	}
	return s.resolvedUserID, nil
}

func (s *resetReaderStub) SetPassword(_ context.Context, userID uuid.UUID, hash string) error {
	s.setPasswordHits++
	s.setPasswordUser = userID
	s.setPasswordHash = hash
	return s.setPasswordErr
}

type recordingAudit struct{ entries []domain.AuditEntry }

func (a *recordingAudit) Append(_ context.Context, entry domain.AuditEntry) error {
	a.entries = append(a.entries, entry)
	return nil
}

type recordingSessionRevoker struct{ revoked []uuid.UUID }

func (s *recordingSessionRevoker) Revoke(_ context.Context, id uuid.UUID) error {
	s.revoked = append(s.revoked, id)
	return nil
}

type recordingHasher struct{ received string }

func (h *recordingHasher) Hash(plain string) (string, error) {
	h.received = plain
	return "argon2:" + plain, nil
}

func (h *recordingHasher) Verify(string, string) (bool, error) { return false, nil }

func newResetFixture(t *testing.T) (*EmployeeService, *resetReaderStub, *recordingAudit, *recordingSessionRevoker, *recordingHasher) {
	t.Helper()
	reader := &resetReaderStub{employeeReaderStub: &employeeReaderStub{}, resolvedUserID: uuid.New()}
	audit := &recordingAudit{}
	sessions := &recordingSessionRevoker{}
	hasher := &recordingHasher{}
	service := NewEmployeeService(reader, transactionStub{}, audit, sessions, hasher, &documentStoreStub{})
	service.now = func() time.Time { return time.Date(2026, time.September, 2, 10, 0, 0, 0, domain.Jakarta()) }
	return service, reader, audit, sessions, hasher
}

func hrIdentity() domain.Identity {
	return domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR}
}

func validResetRequest() dto.ResetEmployeePasswordRequest {
	return dto.ResetEmployeePasswordRequest{
		NewPassword:             "GantiSandi2026!",
		NewPasswordConfirmation: "GantiSandi2026!",
	}
}

func TestResetEmployeePasswordHashesUnlocksAndRevokesAllSessions(t *testing.T) {
	service, reader, _, sessions, hasher := newResetFixture(t)
	identity := hrIdentity()
	target := uuid.New()

	err := service.ResetEmployeePassword(context.Background(), identity, target, validResetRequest(), RequestMeta{RequestID: "req-1", IPAddress: "10.0.0.1"})
	require.NoError(t, err)

	assert.Equal(t, "GantiSandi2026!", hasher.received, "password plain diteruskan ke hasher")
	assert.Equal(t, 1, reader.setPasswordHits)
	assert.Equal(t, reader.resolvedUserID, reader.setPasswordUser, "SetPassword memakai user id hasil resolusi employee id")
	assert.Equal(t, "argon2:GantiSandi2026!", reader.setPasswordHash)
	require.Equal(t, []uuid.UUID{reader.resolvedUserID}, sessions.revoked, "seluruh sesi user target dicabut (revoke-all)")
}

func TestResetEmployeePasswordWritesAuditWithoutPasswordValue(t *testing.T) {
	service, _, audit, _, _ := newResetFixture(t)
	identity := hrIdentity()

	err := service.ResetEmployeePassword(context.Background(), identity, uuid.New(), validResetRequest(), RequestMeta{RequestID: "req-2"})
	require.NoError(t, err)

	require.Len(t, audit.entries, 1)
	entry := audit.entries[0]
	assert.Equal(t, "PASSWORD_RESET", entry.Action)
	assert.LessOrEqual(t, len(entry.Action), 30, "aksi muat pada audit_logs.aksi VARCHAR(30)")
	assert.Equal(t, "karyawan", entry.Module)
	require.NotNil(t, entry.UserID)
	assert.Equal(t, identity.UserID, *entry.UserID)
	for key, value := range entry.Detail {
		if str, ok := value.(string); ok {
			assert.NotContains(t, str, "GantiSandi2026!", "detail audit %q tidak memuat password", key)
		}
	}
	assert.Equal(t, "hr", entry.Detail["by"])
	assert.Equal(t, true, entry.Detail["sessions_revoked"])
}

func TestResetEmployeePasswordRejectsNonHR(t *testing.T) {
	for _, role := range []domain.RoleName{domain.RoleEmployee, domain.RoleSupervisor, domain.RoleTopManagement} {
		service, reader, _, sessions, hasher := newResetFixture(t)
		identity := domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: role}

		err := service.ResetEmployeePassword(context.Background(), identity, uuid.New(), validResetRequest(), RequestMeta{})
		require.ErrorIs(t, err, domain.ErrForbidden, "role %s ditolak", role)
		assert.Zero(t, reader.setPasswordHits, "tidak ada mutasi untuk role %s", role)
		assert.Empty(t, sessions.revoked)
		assert.Empty(t, hasher.received)
	}
}

func TestResetEmployeePasswordRejectsConfirmationMismatch(t *testing.T) {
	service, reader, _, sessions, _ := newResetFixture(t)
	request := validResetRequest()
	request.NewPasswordConfirmation = "BedaSandi2026!"

	err := service.ResetEmployeePassword(context.Background(), hrIdentity(), uuid.New(), request, RequestMeta{})
	require.ErrorIs(t, err, domain.ErrPasswordMismatch)
	assert.Zero(t, reader.setPasswordHits)
	assert.Empty(t, sessions.revoked)
}

func TestResetEmployeePasswordRefusesSelfReset(t *testing.T) {
	service, reader, _, sessions, _ := newResetFixture(t)
	identity := hrIdentity()

	err := service.ResetEmployeePassword(context.Background(), identity, identity.EmployeeID, validResetRequest(), RequestMeta{})
	require.ErrorIs(t, err, domain.ErrInvalidRequest, "HR tidak dapat mereset akunnya sendiri lewat endpoint ini")
	assert.Zero(t, reader.setPasswordHits)
	assert.Empty(t, sessions.revoked)
}

func TestResetEmployeePasswordMapsUnknownEmployeeToNotFound(t *testing.T) {
	service, reader, _, sessions, _ := newResetFixture(t)
	reader.resolveErr = repository.ErrNotFound

	err := service.ResetEmployeePassword(context.Background(), hrIdentity(), uuid.New(), validResetRequest(), RequestMeta{})
	require.ErrorIs(t, err, domain.ErrNotFound)
	assert.Empty(t, sessions.revoked, "tidak mencabut sesi bila employee tidak ditemukan")
}
