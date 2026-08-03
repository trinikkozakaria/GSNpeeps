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

	// Aturan bisnis absensi. Seluruhnya dipetakan ke 422 dengan kode kontrak.
	ErrNonWorkingDay          = errors.New("non working day")
	ErrOutOfRadius            = errors.New("out of office radius")
	ErrDuplicateCheckIn       = errors.New("duplicate check-in")
	ErrCheckoutWithoutCheckIn = errors.New("checkout without check-in")
	ErrInvalidOfficeLocation  = errors.New("invalid office location")

	// Aturan berkas unggahan.
	ErrFileTooLarge     = errors.New("file too large")
	ErrUnsupportedFile  = errors.New("unsupported file type")
	ErrDocumentRequired = errors.New("supporting document required")

	// Aturan approval.
	ErrAlreadyDecided           = errors.New("request already decided")
	ErrInsufficientLeaveBalance = errors.New("insufficient leave balance")
)
