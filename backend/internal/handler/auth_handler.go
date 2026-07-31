package handler

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
)

type AuthService interface {
	Login(context.Context, dto.LoginRequest, service.RequestMeta) (dto.LoginData, error)
	Me(context.Context, domain.Identity) (domain.AuthUser, error)
	Logout(context.Context, domain.Identity, service.RequestMeta) error
	ChangePassword(
		context.Context,
		domain.Identity,
		dto.ChangePasswordRequest,
		service.RequestMeta,
	) (dto.PasswordChangedData, error)
	ResetPassword(
		context.Context,
		dto.SelfResetPasswordRequest,
		service.RequestMeta,
	) (dto.PasswordChangedData, error)
}

type Validator interface {
	Struct(any) map[string]string
}

type AuthHandler struct {
	service    AuthService
	validator  Validator
	trustProxy bool
}

func NewAuthHandler(service AuthService, validator Validator, trustProxy bool) *AuthHandler {
	return &AuthHandler{service: service, validator: validator, trustProxy: trustProxy}
}

func (h *AuthHandler) Login(writer http.ResponseWriter, request *http.Request) {
	var input dto.LoginRequest
	if decodeJSON(request, &input) != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	if fields := h.validator.Struct(input); len(fields) > 0 {
		response.RequestValidationError(writer, fields)
		return
	}
	result, err := h.service.Login(request.Context(), input, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	// NOTE: sesuaikan result.Token / result.ExpiresIn dengan nama field
	// yang sebenarnya ada di struct dto.LoginData milikmu.
	setAuthCookie(writer, result.Token, result.ExpiresIn)
	response.Success(writer, http.StatusOK, result, "Login berhasil")
}

func (h *AuthHandler) Logout(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	if err := h.service.Logout(request.Context(), identity, h.requestMeta(request)); err != nil {
		response.FromError(writer, err)
		return
	}
	clearAuthCookie(writer)
	response.EmptySuccess(writer, "Logout berhasil")
}

func (h *AuthHandler) Me(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	result, err := h.service.Me(request.Context(), identity)
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, result, "OK")
}

func (h *AuthHandler) ChangePassword(writer http.ResponseWriter, request *http.Request) {
	identity, ok := middleware.IdentityFromContext(request.Context())
	if !ok {
		response.FromError(writer, domain.ErrInvalidToken)
		return
	}
	var input dto.ChangePasswordRequest
	if decodeJSON(request, &input) != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	if fields := h.validator.Struct(input); len(fields) > 0 {
		response.ValidationError(writer, fields)
		return
	}
	result, err := h.service.ChangePassword(request.Context(), identity, input, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, result, "Password berhasil diganti")
}

func (h *AuthHandler) ResetPassword(writer http.ResponseWriter, request *http.Request) {
	var input dto.SelfResetPasswordRequest
	if decodeJSON(request, &input) != nil {
		response.Error(writer, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid")
		return
	}
	if fields := h.validator.Struct(input); len(fields) > 0 {
		response.ValidationError(writer, fields)
		return
	}
	result, err := h.service.ResetPassword(request.Context(), input, h.requestMeta(request))
	if err != nil {
		response.FromError(writer, err)
		return
	}
	response.Success(writer, http.StatusOK, result, "Password berhasil direset")
}

func (h *AuthHandler) requestMeta(request *http.Request) service.RequestMeta {
	return service.RequestMeta{
		IPAddress: clientIP(request, h.trustProxy),
		RequestID: middleware.RequestIDFromContext(request.Context()),
	}
}

// setAuthCookie writes the JWT as an httpOnly cookie. Frontend and backend
// run on different origins, so SameSite=None is required for the browser
// to send this cookie on cross-origin requests -- and browsers reject
// SameSite=None cookies that aren't also Secure, so Secure is always true
// here (this requires HTTPS on both sides, including in development).
func setAuthCookie(writer http.ResponseWriter, token string, expiresInSeconds int) {
	http.SetCookie(writer, &http.Cookie{
		Name:     middleware.AccessTokenCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   expiresInSeconds,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearAuthCookie(writer http.ResponseWriter) {
	http.SetCookie(writer, &http.Cookie{
		Name:     middleware.AccessTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
}

func clientIP(request *http.Request, trustProxy bool) string {
	candidate := ""
	if trustProxy {
		candidate = strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-For"), ",")[0])
	}
	if net.ParseIP(candidate) != nil {
		return candidate
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil && net.ParseIP(host) != nil {
		return host
	}
	if net.ParseIP(request.RemoteAddr) != nil {
		return request.RemoteAddr
	}
	return ""
}
