package middleware

import (
	"net/http"
	"strings"
)

// BodyLimit membatasi ukuran body request. Request multipart memakai batas unggahan yang
// lebih besar karena endpoint dokumen dan absensi menerima berkas sampai 5 MB; batas presisi
// per berkas tetap ditegakkan oleh handler terkait.
func BodyLimit(maxBytes, maxUploadBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.Body != nil {
				limit := maxBytes
				if isMultipart(request) && maxUploadBytes > limit {
					limit = maxUploadBytes
				}
				request.Body = http.MaxBytesReader(writer, request.Body, limit)
			}
			next.ServeHTTP(writer, request)
		})
	}
}

func isMultipart(request *http.Request) bool {
	contentType := strings.ToLower(request.Header.Get("Content-Type"))
	return strings.HasPrefix(strings.TrimSpace(contentType), "multipart/form-data")
}
