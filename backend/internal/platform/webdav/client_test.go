package webdav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := New(config.Nextcloud{
		BaseURL:  server.URL + "/remote.php/dav/files/service",
		Username: "service", AppPassword: "test", RootFolder: "GSNpeeps", Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return client, server
}

func TestClientRejectsRedirectToWebPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/index.php", http.StatusFound)
	}))
	defer server.Close()

	client, err := New(config.Nextcloud{
		BaseURL:  server.URL + "/remote.php/dav/files/service",
		Username: "service", AppPassword: "test", RootFolder: "GSNpeeps", Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if _, err = client.Upload(context.Background(), "employee-photos/photo.jpg", strings.NewReader("jpeg"), "image/jpeg"); err == nil || !strings.Contains(err.Error(), "status 302") {
		t.Fatalf("Upload() error = %v, want redirect status error", err)
	}
	if _, _, err = client.Download(context.Background(), "GSNpeeps/employee-photos/photo.jpg"); err == nil || !strings.Contains(err.Error(), "status 302") {
		t.Fatalf("Download() error = %v, want redirect status error", err)
	}
}

// TestUploadCreatesMissingParentCollections memastikan Upload membuat rootFolder dan setiap
// direktori antara sebelum PUT (defect: upload dokumen karyawan pertama untuk suatu employee
// gagal 404/409 karena koleksi indumnya belum pernah dibuat di Nextcloud).
func TestUploadCreatesMissingParentCollections(t *testing.T) {
	var mkcolPaths []string
	var putHappened bool
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "MKCOL":
			mkcolPaths = append(mkcolPaths, r.URL.Path)
			w.WriteHeader(http.StatusCreated)
		case http.MethodPut:
			putHappened = true
			w.WriteHeader(http.StatusCreated)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	})

	location, err := client.Upload(
		context.Background(), "employee-documents/emp-1/doc.pdf", strings.NewReader("pdf"), "application/pdf",
	)
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if location != "GSNpeeps/employee-documents/emp-1/doc.pdf" {
		t.Fatalf("Upload() location = %q", location)
	}
	if !putHappened {
		t.Fatal("Upload() did not PUT the object")
	}
	want := []string{
		"/remote.php/dav/files/service/GSNpeeps",
		"/remote.php/dav/files/service/GSNpeeps/employee-documents",
		"/remote.php/dav/files/service/GSNpeeps/employee-documents/emp-1",
	}
	if len(mkcolPaths) != len(want) {
		t.Fatalf("MKCOL calls = %v, want %v", mkcolPaths, want)
	}
	for i, path := range want {
		if mkcolPaths[i] != path {
			t.Fatalf("MKCOL[%d] = %q, want %q", i, mkcolPaths[i], path)
		}
	}
}

// TestUploadToleratesExistingCollections memastikan 405 Method Not Allowed pada MKCOL (koleksi
// sudah ada dari upload sebelumnya) tidak dianggap kegagalan.
func TestUploadToleratesExistingCollections(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "MKCOL":
			w.WriteHeader(http.StatusMethodNotAllowed)
		case http.MethodPut:
			w.WriteHeader(http.StatusCreated)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	})

	if _, err := client.Upload(
		context.Background(), "employee-documents/emp-1/doc.pdf", strings.NewReader("pdf"), "application/pdf",
	); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
}

// TestUploadFailsWhenCollectionCannotBeCreated memastikan status MKCOL selain 201/405 tetap
// menggagalkan upload alih-alih diam-diam melanjutkan ke PUT yang pasti gagal.
func TestUploadFailsWhenCollectionCannotBeCreated(t *testing.T) {
	var putAttempted bool
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "MKCOL":
			w.WriteHeader(http.StatusForbidden)
		case http.MethodPut:
			putAttempted = true
			w.WriteHeader(http.StatusCreated)
		}
	})

	_, err := client.Upload(
		context.Background(), "employee-documents/emp-1/doc.pdf", strings.NewReader("pdf"), "application/pdf",
	)
	if err == nil || !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("Upload() error = %v, want collection creation error", err)
	}
	if putAttempted {
		t.Fatal("Upload() should not PUT when a parent collection cannot be created")
	}
}
