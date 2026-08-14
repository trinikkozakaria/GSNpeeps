package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
	"github.com/stretchr/testify/require"
)

type fakeAuthUsers struct {
	mu      sync.Mutex
	account domain.LoginAccount
}

func (f *fakeAuthUsers) FindForLogin(_ context.Context, email string) (domain.LoginAccount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if email != f.account.Email {
		return domain.LoginAccount{}, repository.ErrNotFound
	}
	return f.account, nil
}

func (f *fakeAuthUsers) FindForPasswordByID(_ context.Context, id uuid.UUID) (domain.LoginAccount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id != f.account.ID {
		return domain.LoginAccount{}, repository.ErrNotFound
	}
	return f.account, nil
}

func (f *fakeAuthUsers) FindIdentityByID(_ context.Context, id uuid.UUID) (domain.AuthUser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id != f.account.ID {
		return domain.AuthUser{}, repository.ErrNotFound
	}
	return domain.AuthUser{
		ID:         f.account.ID,
		EmployeeID: f.account.EmployeeID,
		Name:       f.account.Name,
		Email:      f.account.Email,
		Role:       f.account.Role,
	}, nil
}

func (f *fakeAuthUsers) RecordFailedLogin(_ context.Context, id uuid.UUID, threshold int) (int, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id != f.account.ID {
		return 0, false, repository.ErrNotFound
	}
	f.account.FailedLoginCount++
	f.account.AccountLocked = f.account.FailedLoginCount >= threshold
	return f.account.FailedLoginCount, f.account.AccountLocked, nil
}

func (f *fakeAuthUsers) RecordSuccessfulLogin(_ context.Context, id uuid.UUID, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id != f.account.ID {
		return repository.ErrNotFound
	}
	f.account.FailedLoginCount = 0
	f.account.AccountLocked = false
	return nil
}

func (f *fakeAuthUsers) UpdatePassword(_ context.Context, id uuid.UUID, hash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id != f.account.ID {
		return repository.ErrNotFound
	}
	f.account.PasswordHash = hash
	f.account.AccountLocked = false
	f.account.FailedLoginCount = 0
	return nil
}

type fakePasswords struct{}

func (fakePasswords) Hash(value string) (string, error) { return "hash:" + value, nil }
func (fakePasswords) Verify(value, hash string) (bool, error) {
	return hash == "hash:"+value, nil
}

type fakeTokens struct{}

func (fakeTokens) Issue(domain.Identity) (string, string, time.Duration, error) {
	return "synthetic-jwt", "fingerprint", 8 * time.Hour, nil
}

type fakeSessions struct {
	revoked          int
	saved            int
	revokeContextErr error
}

func (f *fakeSessions) Save(context.Context, uuid.UUID, string, time.Duration) error {
	f.saved++
	return nil
}
func (f *fakeSessions) Revoke(ctx context.Context, _ uuid.UUID) error {
	f.revoked++
	f.revokeContextErr = ctx.Err()
	return nil
}

type fakeLimiter struct{ allowed bool }

func (f fakeLimiter) Allow(context.Context, string, int, time.Duration) (bool, error) {
	return f.allowed, nil
}

type fakeAudit struct {
	entries          []domain.AuditEntry
	appendContextErr error
}

func (f *fakeAudit) Append(ctx context.Context, entry domain.AuditEntry) error {
	f.entries = append(f.entries, entry)
	f.appendContextErr = ctx.Err()
	return nil
}

func newAuthFixture(t *testing.T) (*AuthService, *fakeAuthUsers, *fakeSessions) {
	t.Helper()
	users := &fakeAuthUsers{account: domain.LoginAccount{
		ID:             uuid.New(),
		EmployeeID:     uuid.New(),
		RoleID:         uuid.New(),
		Name:           "Karyawan Sintetis",
		Email:          "user@example.test",
		PasswordHash:   "hash:valid-password",
		Role:           domain.RoleEmployee,
		EmployeeStatus: "aktif",
	}}
	sessions := &fakeSessions{}
	service, err := NewAuthService(
		users,
		fakePasswords{},
		fakeTokens{},
		sessions,
		fakeLimiter{allowed: true},
		&fakeAudit{},
		config.Auth{
			LoginFailureLimit: 5,
			RequestWindow:     time.Minute,
		},
	)
	require.NoError(t, err)
	return service, users, sessions
}

func TestLogoutFinalizesSessionAndAuditAfterClientCancellation(t *testing.T) {
	users := &fakeAuthUsers{account: domain.LoginAccount{ID: uuid.New(), EmployeeID: uuid.New()}}
	sessions := &fakeSessions{}
	audit := &fakeAudit{}
	auth, err := NewAuthService(
		users, fakePasswords{}, fakeTokens{}, sessions, fakeLimiter{allowed: true}, audit,
		config.Auth{LoginFailureLimit: 5, RequestWindow: time.Minute},
	)
	require.NoError(t, err)

	requestCtx, cancel := context.WithCancel(context.Background())
	cancel()
	err = auth.Logout(requestCtx, domain.Identity{UserID: users.account.ID}, RequestMeta{IPAddress: "127.0.0.1"})

	require.NoError(t, err)
	require.Equal(t, 1, sessions.revoked)
	require.NoError(t, sessions.revokeContextErr)
	require.Len(t, audit.entries, 1)
	require.Equal(t, "LOGOUT", audit.entries[0].Action)
	require.NoError(t, audit.appendContextErr)
}

func TestLoginSuccess(t *testing.T) {
	auth, users, sessions := newAuthFixture(t)
	result, err := auth.Login(context.Background(), dto.LoginRequest{
		Email:    " USER@EXAMPLE.TEST ",
		Password: "valid-password",
	}, RequestMeta{IPAddress: "127.0.0.1"})

	require.NoError(t, err)
	require.Equal(t, "synthetic-jwt", result.Token)
	require.Equal(t, 28800, result.ExpiresIn)
	require.Equal(t, domain.RoleEmployee, result.User.Role)
	require.Equal(t, 1, sessions.saved)
	require.Equal(t, 0, users.account.FailedLoginCount)
}

func TestFifthFailedLoginLocksAndRevokesSession(t *testing.T) {
	auth, users, sessions := newAuthFixture(t)
	for attempt := 1; attempt <= 4; attempt++ {
		_, err := auth.Login(context.Background(), dto.LoginRequest{
			Email: "user@example.test", Password: "wrong-password",
		}, RequestMeta{IPAddress: "127.0.0.1"})
		require.ErrorIs(t, err, domain.ErrInvalidCredentials)
	}
	_, err := auth.Login(context.Background(), dto.LoginRequest{
		Email: "user@example.test", Password: "wrong-password",
	}, RequestMeta{IPAddress: "127.0.0.1"})

	require.ErrorIs(t, err, domain.ErrAccountLocked)
	require.True(t, users.account.AccountLocked)
	require.Equal(t, 5, users.account.FailedLoginCount)
	require.Equal(t, 1, sessions.revoked)
}

func TestSelfResetUnlocksAndRevokesSession(t *testing.T) {
	auth, users, sessions := newAuthFixture(t)
	users.account.AccountLocked = true
	users.account.FailedLoginCount = 5

	result, err := auth.ResetPassword(context.Background(), dto.SelfResetPasswordRequest{
		Email:                   "user@example.test",
		CurrentPassword:         "valid-password",
		NewPassword:             "new-valid-password",
		NewPasswordConfirmation: "new-valid-password",
	}, RequestMeta{IPAddress: "127.0.0.1"})

	require.NoError(t, err)
	require.True(t, result.PasswordChanged)
	require.False(t, users.account.AccountLocked)
	require.Equal(t, "hash:new-valid-password", users.account.PasswordHash)
	require.Equal(t, 1, sessions.revoked)
}

func TestRateLimitFailsClosed(t *testing.T) {
	auth, _, _ := newAuthFixture(t)
	auth.limiter = fakeLimiter{allowed: false}
	_, err := auth.Login(context.Background(), dto.LoginRequest{
		Email: "user@example.test", Password: "valid-password",
	}, RequestMeta{IPAddress: "127.0.0.1"})
	require.True(t, errors.Is(err, domain.ErrRateLimited))
}
