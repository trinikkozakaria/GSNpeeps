package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

type fakeTokenVerifier struct {
	identity domain.Identity
	err      error
}

func (f fakeTokenVerifier) Verify(context.Context, string) (domain.Identity, string, error) {
	return f.identity, "fingerprint", f.err
}

type fakeSessionValidator struct{ err error }

func (f fakeSessionValidator) Validate(context.Context, uuid.UUID, string) error {
	return f.err
}

func TestAuthenticateAddsTypedIdentity(t *testing.T) {
	expected := domain.Identity{
		UserID:     uuid.New(),
		EmployeeID: uuid.New(),
		Role:       domain.RoleHR,
	}
	handler := Authenticate(
		fakeTokenVerifier{identity: expected},
		fakeSessionValidator{},
	)(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		actual, ok := IdentityFromContext(request.Context())
		require.True(t, ok)
		require.Equal(t, expected, actual)
		writer.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer synthetic-token")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestAuthenticateRejectsMissingBearer(t *testing.T) {
	handler := Authenticate(fakeTokenVerifier{}, fakeSessionValidator{})(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("protected handler must not run")
		}),
	)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Contains(t, recorder.Body.String(), "INVALID_TOKEN")
}
