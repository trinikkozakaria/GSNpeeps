package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
)

type AuthUserRepository interface {
	FindForLogin(context.Context, string) (domain.LoginAccount, error)
	FindForPasswordByID(context.Context, uuid.UUID) (domain.LoginAccount, error)
	FindIdentityByID(context.Context, uuid.UUID) (domain.AuthUser, error)
	RecordFailedLogin(context.Context, uuid.UUID, int) (int, bool, error)
	RecordSuccessfulLogin(context.Context, uuid.UUID, time.Time) error
	UpdatePassword(context.Context, uuid.UUID, string) error
}

type PasswordHasher interface {
	Hash(string) (string, error)
	Verify(string, string) (bool, error)
}

type TokenIssuer interface {
	Issue(domain.Identity) (string, string, time.Duration, error)
}

type SessionStore interface {
	Save(context.Context, uuid.UUID, string, time.Duration) error
	Revoke(context.Context, uuid.UUID) error
}

type AuthRateLimiter interface {
	Allow(context.Context, string, int, time.Duration) (bool, error)
}

type AuditWriter interface {
	Append(context.Context, domain.AuditEntry) error
}

type RequestMeta struct {
	IPAddress string
	RequestID string
}

type AuthService struct {
	users          AuthUserRepository
	passwords      PasswordHasher
	tokens         TokenIssuer
	sessions       SessionStore
	limiter        AuthRateLimiter
	audit          AuditWriter
	failureLimit   int
	loginRateLimit int
	rateWindow     time.Duration
	dummyHash      string
	now            func() time.Time
}

func NewAuthService(
	users AuthUserRepository,
	passwords PasswordHasher,
	tokens TokenIssuer,
	sessions SessionStore,
	limiter AuthRateLimiter,
	audit AuditWriter,
	cfg config.Auth,
) (*AuthService, error) {
	dummyHash, err := passwords.Hash("gsnpeeps-dummy-verification-value")
	if err != nil {
		return nil, fmt.Errorf("initialize auth verifier: %w", err)
	}
	return &AuthService{
		users:          users,
		passwords:      passwords,
		tokens:         tokens,
		sessions:       sessions,
		limiter:        limiter,
		audit:          audit,
		failureLimit:   cfg.LoginFailureLimit,
		loginRateLimit: max(cfg.LoginFailureLimit*2, 10),
		rateWindow:     cfg.RequestWindow,
		dummyHash:      dummyHash,
		now:            time.Now,
	}, nil
}

func (s *AuthService) Login(
	ctx context.Context,
	request dto.LoginRequest,
	meta RequestMeta,
) (dto.LoginData, error) {
	email := normalizeEmail(request.Email)
	if err := s.checkPublicRate(ctx, email, meta.IPAddress); err != nil {
		return dto.LoginData{}, err
	}

	account, err := s.users.FindForLogin(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		_, _ = s.passwords.Verify(request.Password, s.dummyHash)
		return dto.LoginData{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return dto.LoginData{}, fmt.Errorf("login lookup: %w", err)
	}
	if account.AccountLocked {
		return dto.LoginData{}, domain.ErrAccountLocked
	}
	if !account.Active() {
		return dto.LoginData{}, domain.ErrInactiveAccount
	}

	matches, err := s.passwords.Verify(request.Password, account.PasswordHash)
	if err != nil {
		return dto.LoginData{}, fmt.Errorf("verify login password: %w", err)
	}
	if !matches {
		_, locked, err := s.users.RecordFailedLogin(ctx, account.ID, s.failureLimit)
		if err != nil {
			return dto.LoginData{}, fmt.Errorf("record login failure: %w", err)
		}
		if locked {
			if err := s.sessions.Revoke(ctx, account.ID); err != nil {
				return dto.LoginData{}, fmt.Errorf("revoke locked session: %w", err)
			}
			_ = s.appendAudit(ctx, account.ID, "ACCOUNT_LOCK", meta, map[string]any{
				"reason": "consecutive_authentication_failures",
			})
			return dto.LoginData{}, domain.ErrAccountLocked
		}
		return dto.LoginData{}, domain.ErrInvalidCredentials
	}

	now := s.now().UTC()
	if err := s.users.RecordSuccessfulLogin(ctx, account.ID, now); err != nil {
		return dto.LoginData{}, fmt.Errorf("complete login: %w", err)
	}
	identity := domain.Identity{UserID: account.ID, EmployeeID: account.EmployeeID, Role: account.Role}
	rawToken, fingerprint, ttl, err := s.tokens.Issue(identity)
	if err != nil {
		return dto.LoginData{}, err
	}
	if err := s.sessions.Save(ctx, account.ID, fingerprint, ttl); err != nil {
		return dto.LoginData{}, err
	}
	if err := s.appendAudit(ctx, account.ID, "LOGIN", meta, nil); err != nil {
		_ = s.sessions.Revoke(ctx, account.ID)
		return dto.LoginData{}, err
	}

	return dto.LoginData{
		Token:     rawToken,
		TokenType: "Bearer",
		ExpiresIn: int(ttl.Seconds()),
		User: domain.AuthUser{
			ID:         account.ID,
			EmployeeID: account.EmployeeID,
			Name:       account.Name,
			Email:      account.Email,
			Role:       account.Role,
			PhotoURL:   account.PhotoURL,
		},
	}, nil
}

func (s *AuthService) Me(ctx context.Context, identity domain.Identity) (domain.AuthUser, error) {
	user, err := s.users.FindIdentityByID(ctx, identity.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.AuthUser{}, domain.ErrInactiveAccount
	}
	if err != nil {
		return domain.AuthUser{}, fmt.Errorf("get current identity: %w", err)
	}
	if user.EmployeeID != identity.EmployeeID || user.Role != identity.Role {
		return domain.AuthUser{}, domain.ErrSessionInvalid
	}
	return user, nil
}

func (s *AuthService) Logout(ctx context.Context, identity domain.Identity, meta RequestMeta) error {
	if err := s.sessions.Revoke(ctx, identity.UserID); err != nil {
		return err
	}
	return s.appendAudit(ctx, identity.UserID, "LOGOUT", meta, nil)
}

func (s *AuthService) ChangePassword(
	ctx context.Context,
	identity domain.Identity,
	request dto.ChangePasswordRequest,
	meta RequestMeta,
) (dto.PasswordChangedData, error) {
	if request.NewPassword != request.NewPasswordConfirmation {
		return dto.PasswordChangedData{}, domain.ErrPasswordMismatch
	}
	account, err := s.users.FindForPasswordByID(ctx, identity.UserID)
	if err != nil {
		return dto.PasswordChangedData{}, fmt.Errorf("load password account: %w", err)
	}
	return s.replacePassword(ctx, account, request.CurrentPassword, request.NewPassword, meta)
}

func (s *AuthService) ResetPassword(
	ctx context.Context,
	request dto.SelfResetPasswordRequest,
	meta RequestMeta,
) (dto.PasswordChangedData, error) {
	email := normalizeEmail(request.Email)
	if err := s.checkPublicRate(ctx, email, meta.IPAddress); err != nil {
		return dto.PasswordChangedData{}, err
	}
	if request.NewPassword != request.NewPasswordConfirmation {
		return dto.PasswordChangedData{}, domain.ErrPasswordMismatch
	}
	account, err := s.users.FindForLogin(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		_, _ = s.passwords.Verify(request.CurrentPassword, s.dummyHash)
		return dto.PasswordChangedData{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return dto.PasswordChangedData{}, fmt.Errorf("reset password lookup: %w", err)
	}
	if !account.Active() {
		return dto.PasswordChangedData{}, domain.ErrInvalidCredentials
	}
	return s.replacePassword(ctx, account, request.CurrentPassword, request.NewPassword, meta)
}

func (s *AuthService) replacePassword(
	ctx context.Context,
	account domain.LoginAccount,
	currentPassword string,
	newPassword string,
	meta RequestMeta,
) (dto.PasswordChangedData, error) {
	matches, err := s.passwords.Verify(currentPassword, account.PasswordHash)
	if err != nil {
		return dto.PasswordChangedData{}, fmt.Errorf("verify current password: %w", err)
	}
	if !matches {
		_, locked, err := s.users.RecordFailedLogin(ctx, account.ID, s.failureLimit)
		if err != nil {
			return dto.PasswordChangedData{}, fmt.Errorf("record password verification failure: %w", err)
		}
		if locked {
			if err := s.sessions.Revoke(ctx, account.ID); err != nil {
				return dto.PasswordChangedData{}, err
			}
			return dto.PasswordChangedData{}, domain.ErrAccountLocked
		}
		return dto.PasswordChangedData{}, domain.ErrInvalidCredentials
	}

	same, err := s.passwords.Verify(newPassword, account.PasswordHash)
	if err != nil {
		return dto.PasswordChangedData{}, fmt.Errorf("compare new password: %w", err)
	}
	if same {
		return dto.PasswordChangedData{}, domain.ErrPasswordUnchanged
	}
	encoded, err := s.passwords.Hash(newPassword)
	if err != nil {
		return dto.PasswordChangedData{}, fmt.Errorf("hash new password: %w", err)
	}
	if err := s.users.UpdatePassword(ctx, account.ID, encoded); err != nil {
		return dto.PasswordChangedData{}, fmt.Errorf("save new password: %w", err)
	}
	if err := s.sessions.Revoke(ctx, account.ID); err != nil {
		return dto.PasswordChangedData{}, err
	}
	if err := s.appendAudit(ctx, account.ID, "UPDATE", meta, map[string]any{
		"field":            "password",
		"sessions_revoked": true,
	}); err != nil {
		return dto.PasswordChangedData{}, err
	}
	return dto.PasswordChangedData{
		PasswordChanged: true,
		AccountLocked:   false,
		SessionsRevoked: true,
	}, nil
}

func (s *AuthService) checkPublicRate(ctx context.Context, email, ipAddress string) error {
	for _, key := range []string{
		"auth:account:" + hashKey(email),
		"auth:ip:" + hashKey(ipAddress),
	} {
		allowed, err := s.limiter.Allow(ctx, key, s.loginRateLimit, s.rateWindow)
		if err != nil {
			return fmt.Errorf("check auth rate limit: %w", err)
		}
		if !allowed {
			return domain.ErrRateLimited
		}
	}
	return nil
}

func (s *AuthService) appendAudit(
	ctx context.Context,
	userID uuid.UUID,
	action string,
	meta RequestMeta,
	detail map[string]any,
) error {
	if detail == nil {
		detail = map[string]any{}
	}
	if meta.RequestID != "" {
		detail["request_id"] = meta.RequestID
	}
	return s.audit.Append(ctx, domain.AuditEntry{
		UserID:    &userID,
		Action:    action,
		Module:    "auth",
		DataID:    &userID,
		Detail:    detail,
		IPAddress: meta.IPAddress,
		CreatedAt: s.now().UTC(),
	})
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func hashKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
