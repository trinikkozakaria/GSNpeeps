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

func TestSafePathRejectsTraversal(t *testing.T) {
	for _, value := range []string{"../secret", "employees/../../secret", `employees\..\secret`, "/absolute"} {
		if _, err := safePath(value); err == nil {
			t.Fatalf("safePath(%q) accepted traversal", value)
		}
	}
}

func TestSafePathAcceptsNestedRelativePath(t *testing.T) {
	got, err := safePath("employees/123/contract.pdf")
	if err != nil {
		t.Fatalf("safePath() error = %v", err)
	}
	if got != "employees/123/contract.pdf" {
		t.Fatalf("safePath() = %q", got)
	}
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
