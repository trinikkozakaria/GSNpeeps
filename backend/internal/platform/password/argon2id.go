package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"golang.org/x/crypto/argon2"
)

type Hasher struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

func New(cfg config.Auth) *Hasher {
	return &Hasher{
		memory:      cfg.ArgonMemoryKiB,
		iterations:  cfg.ArgonIterations,
		parallelism: cfg.ArgonParallelism,
		saltLength:  16,
		keyLength:   32,
	}
}

func (h *Hasher) Hash(value string) (string, error) {
	salt := make([]byte, h.saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey([]byte(value), salt, h.iterations, h.memory, h.parallelism, h.keyLength)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		h.memory,
		h.iterations,
		h.parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func (h *Hasher) Verify(value, encoded string) (bool, error) {
	var memory, iterations uint32
	var parallelism uint8
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false, errors.New("invalid password hash format")
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, errors.New("invalid password hash parameters")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, errors.New("invalid password hash salt")
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, errors.New("invalid password hash value")
	}
	actual := argon2.IDKey([]byte(value), salt, iterations, memory, parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}
