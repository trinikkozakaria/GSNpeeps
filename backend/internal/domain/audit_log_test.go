package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactAuditDetailHidesSensitiveValues(t *testing.T) {
	redacted := RedactAuditDetail(map[string]any{
		"email":            "karyawan@example.test",
		"password_hash":    "$argon2id$v=19$m=65536",
		"new_password":     "rahasia",
		"Authorization":    "Bearer abc.def.ghi",
		"session_id":       "session-1",
		"gaji_pokok":       9_500_000,
		"nomor_npwp":       "12.345.678.9-012.000",
		"dokumen_url":      "https://files.example.test/kontrak.pdf",
		"status_baru":      "disetujui",
		"jumlah_hari":      3,
		"catatan_approver": "Disetujui",
	})

	for _, key := range []string{
		"password_hash", "new_password", "Authorization", "session_id",
		"gaji_pokok", "nomor_npwp", "dokumen_url",
	} {
		assert.Equalf(t, AuditRedactedPlaceholder, redacted[key], "field %s harus di-redact", key)
	}

	// Field operasional tetap terbaca agar audit tetap berguna.
	assert.Equal(t, "disetujui", redacted["status_baru"])
	assert.Equal(t, 3, redacted["jumlah_hari"])
	assert.Equal(t, "Disetujui", redacted["catatan_approver"])
	assert.Equal(t, "karyawan@example.test", redacted["email"])
}

func TestRedactAuditDetailWalksNestedMaps(t *testing.T) {
	redacted := RedactAuditDetail(map[string]any{
		"sebelum": map[string]any{
			"nomor_rekening": "1234567890",
			"modul":          "akses",
		},
	})

	nested, ok := redacted["sebelum"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, AuditRedactedPlaceholder, nested["nomor_rekening"])
	assert.Equal(t, "akses", nested["modul"])
}

func TestRedactAuditDetailKeepsNilDetail(t *testing.T) {
	assert.Nil(t, RedactAuditDetail(nil))
}
