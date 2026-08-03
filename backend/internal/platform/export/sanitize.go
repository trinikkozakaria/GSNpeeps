// Package export membuat berkas XLSX dan PDF untuk operasi export yang disetujui kontrak.
// Implementasi hanya memakai standard library sehingga tidak menambah dependency di luar
// baseline stack yang disetujui.
package export

import "strings"

// formulaTriggers adalah karakter pembuka yang membuat spreadsheet memperlakukan sel
// sebagai formula. Nilai apa pun yang diawali karakter ini berasal dari data pengguna dan
// harus dinetralkan sebelum ditulis.
const formulaTriggers = "=+-@\t\r"

// SanitizeCell menetralkan formula injection dengan memberi prefix apostrof pada nilai yang
// dapat dieksekusi spreadsheet, lalu membuang control character yang merusak XML.
func SanitizeCell(value string) string {
	if value == "" {
		return value
	}
	// Trigger diperiksa pada nilai asli karena tab dan carriage return juga membuat
	// spreadsheet mengevaluasi sel, sedangkan stripControl akan mengubahnya jadi spasi.
	dangerous := strings.ContainsRune(formulaTriggers, rune(value[0]))
	cleaned := stripControl(value)
	if cleaned == "" {
		return cleaned
	}
	if dangerous {
		return "'" + cleaned
	}
	return cleaned
}

func stripControl(value string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			return ' '
		case r < 0x20 || r == 0x7f:
			return -1
		default:
			return r
		}
	}, value)
}

// SanitizeFileName membuat nama berkas aman untuk header Content-Disposition: hanya
// karakter ASCII yang dapat diprediksi dan tanpa separator path.
func SanitizeFileName(value string) string {
	mapped := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_' || r == '.':
			return r
		default:
			return '-'
		}
	}, value)
	trimmed := strings.Trim(mapped, "-.")
	if trimmed == "" {
		return "export"
	}
	return trimmed
}
