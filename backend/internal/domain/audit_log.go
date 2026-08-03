package domain

import "strings"

// auditSensitiveKeys adalah fragmen nama field yang tidak boleh keluar melalui pembacaan
// Audit Log. Pencocokan memakai substring case-insensitive sehingga variasi seperti
// `password_hash`, `new_password`, dan `authorization_header` ikut tertangkap.
var auditSensitiveKeys = []string{
	"password",
	"token",
	"hash",
	"secret",
	"authorization",
	"session",
	"credential",
	"app_password",
	"gaji",
	"salary",
	"npwp",
	"nik",
	"ktp",
	"rekening",
	"bank",
	"foto",
	"dokumen_url",
	"file_url",
}

// AuditRedactedPlaceholder menggantikan nilai sensitif pada response Audit Log.
const AuditRedactedPlaceholder = "[REDACTED]"

// RedactAuditDetail mengganti nilai field sensitif sebelum detail audit dikirim ke HR atau
// Top Management. Penulisan audit sudah dikurasi per modul; fungsi ini adalah jaring pengaman
// terakhir sehingga field baru yang lalai tidak langsung terekspos pada pembacaan.
//
// Redaksi berjalan rekursif untuk map bersarang. Nilai non-map dibiarkan apa adanya agar
// audit tetap berguna.
func RedactAuditDetail(detail map[string]any) map[string]any {
	if detail == nil {
		return nil
	}
	redacted := make(map[string]any, len(detail))
	for key, value := range detail {
		if isSensitiveAuditKey(key) {
			redacted[key] = AuditRedactedPlaceholder
			continue
		}
		if nested, ok := value.(map[string]any); ok {
			redacted[key] = RedactAuditDetail(nested)
			continue
		}
		redacted[key] = value
	}
	return redacted
}

func isSensitiveAuditKey(key string) bool {
	lowered := strings.ToLower(key)
	for _, sensitive := range auditSensitiveKeys {
		if strings.Contains(lowered, sensitive) {
			return true
		}
	}
	return false
}
