package token

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

const issuer = "gsnpeeps"

type Claims struct {
	UserID     uuid.UUID       `json:"user_id"`
	EmployeeID uuid.UUID       `json:"employee_id"`
	Role       domain.RoleName `json:"role"`
	jwt.RegisteredClaims
}

type Manager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func New(cfg config.JWT) *Manager {
	return &Manager{secret: []byte(cfg.Secret), ttl: cfg.TTL, now: time.Now}
}

func (m *Manager) Issue(identity domain.Identity) (string, string, time.Duration, error) {
	now := m.now().UTC()
	tokenID := uuid.NewString()
	claims := Claims{
		UserID:     identity.UserID,
		EmployeeID: identity.EmployeeID,
		Role:       identity.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   identity.UserID.String(),
			ID:        tokenID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", "", 0, fmt.Errorf("sign JWT: %w", err)
	}
	return signed, fingerprint(tokenID), m.ttl, nil
}

func (m *Manager) Verify(_ context.Context, raw string) (domain.Identity, string, error) {
	parsed, err := jwt.ParseWithClaims(
		raw,
		&Claims{},
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("unexpected JWT algorithm")
			}
			return m.secret, nil
		},
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || !parsed.Valid {
		return domain.Identity{}, "", domain.ErrInvalidToken
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || claims.UserID == uuid.Nil || claims.EmployeeID == uuid.Nil ||
		!claims.Role.Valid() || claims.ID == "" {
		return domain.Identity{}, "", domain.ErrInvalidToken
	}
	return domain.Identity{
		UserID:     claims.UserID,
		EmployeeID: claims.EmployeeID,
		Role:       claims.Role,
	}, fingerprint(claims.ID), nil
}

func fingerprint(tokenID string) string {
	sum := sha256.Sum256([]byte(tokenID))
	return hex.EncodeToString(sum[:])
}
