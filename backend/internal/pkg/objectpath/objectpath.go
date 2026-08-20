// Package objectpath memvalidasi path relatif objek yang dipakai bersama oleh setiap adapter
// object storage (MinIO, Nextcloud/WebDAV) sebelum path tersebut menjadi object key atau
// collection path.
package objectpath

import (
	"errors"
	"path"
	"strings"
)

// SafePath menormalkan dan memvalidasi path relatif. Ia menolak path kosong, path absolut,
// dan segmen traversal (".", "..") sehingga path yang lolos aman dipakai sebagai object
// key/collection oleh adapter storage manapun.
func SafePath(value string) (string, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if normalized == "" || strings.HasPrefix(normalized, "/") {
		return "", errors.New("path must be relative and non-empty")
	}
	for _, part := range strings.Split(normalized, "/") {
		if part == "" || part == "." || part == ".." {
			return "", errors.New("path traversal is not allowed")
		}
	}
	cleaned := path.Clean(normalized)
	if cleaned == "." || strings.HasPrefix(cleaned, "../") {
		return "", errors.New("path traversal is not allowed")
	}
	return cleaned, nil
}
