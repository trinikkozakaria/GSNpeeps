package domain

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountLocked      = errors.New("account locked")
	ErrInactiveAccount    = errors.New("inactive account")
	ErrInvalidToken       = errors.New("invalid token")
	ErrSessionInvalid     = errors.New("session invalid")
	ErrForbidden          = errors.New("forbidden")
	ErrRateLimited        = errors.New("rate limited")
	ErrPasswordMismatch   = errors.New("password confirmation mismatch")
	ErrPasswordUnchanged  = errors.New("new password matches current password")
	ErrNotFound           = errors.New("resource not found")
	ErrConflict           = errors.New("resource conflict")
	ErrInvalidRequest     = errors.New("invalid request")
)
