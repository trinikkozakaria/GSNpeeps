package dto

import "github.com/gsnpeeps/gsnpeeps/backend/internal/domain"

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

type LoginData struct {
	Token     string          `json:"token"`
	TokenType string          `json:"token_type"`
	ExpiresIn int             `json:"expires_in"`
	User      domain.AuthUser `json:"user"`
}

type ChangePasswordRequest struct {
	CurrentPassword         string `json:"current_password" validate:"required,min=8,max=128"`
	NewPassword             string `json:"new_password" validate:"required,min=12,max=128"`
	NewPasswordConfirmation string `json:"new_password_confirmation" validate:"required,min=12,max=128"`
}

type SelfResetPasswordRequest struct {
	Email                   string `json:"email" validate:"required,email,max=255"`
	CurrentPassword         string `json:"current_password" validate:"required,min=8,max=128"`
	NewPassword             string `json:"new_password" validate:"required,min=12,max=128"`
	NewPasswordConfirmation string `json:"new_password_confirmation" validate:"required,min=12,max=128"`
}

type PasswordChangedData struct {
	PasswordChanged bool `json:"password_changed"`
	AccountLocked   bool `json:"account_locked"`
	SessionsRevoked bool `json:"sessions_revoked"`
}
