package minio

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
)

func testConfig(serverURL string) config.MinIO {
	return config.MinIO{
		Endpoint:  strings.TrimPrefix(serverURL, "http://"),
		AccessKey: "service",
		SecretKey: "not-a-real-secret",
		Bucket:    "gsnpeeps",
		UseSSL:    false,
		Region:    "us-east-1",
		Timeout:   5 * time.Second,
	}
}

// trimmedPath drops the trailing slash minio-go appends to bucket-only path-style requests
// (e.g. "/gsnpeeps/") so handlers can compare against a plain "/gsnpeeps".
func trimmedPath(r *http.Request) string {
	return strings.TrimSuffix(r.URL.Path, "/")
}

// bucketExistsHandler membangun handler HEAD /gsnpeeps yang selalu melaporkan bucket sudah
// ada, supaya test yang tidak menguji ensureBucket bisa fokus ke operasi objeknya sendiri.
func bucketExistsHandler(t *testing.T, objectHandler http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead && trimmedPath(r) == "/gsnpeeps" {
			w.WriteHeader(http.StatusOK)
			return
		}
		objectHandler(w, r)
	}
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := New(context.Background(), testConfig(server.URL))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return client
}

func TestNewCreatesBucketWhenMissing(t *testing.T) {
	var headCalls, putCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodHead && trimmedPath(r) == "/gsnpeeps":
			headCalls++
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodPut && trimmedPath(r) == "/gsnpeeps":
			putCalls++
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if _, err := New(context.Background(), testConfig(server.URL)); err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if headCalls == 0 {
		t.Fatal("New() did not check bucket existence")
	}
	if putCalls != 1 {
		t.Fatalf("New() MakeBucket calls = %d, want 1", putCalls)
	}
}

func TestNewToleratesExistingBucket(t *testing.T) {
	var putCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodHead && trimmedPath(r) == "/gsnpeeps":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && trimmedPath(r) == "/gsnpeeps":
			putCalls++
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if _, err := New(context.Background(), testConfig(server.URL)); err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if putCalls != 0 {
		t.Fatalf("New() should not create a bucket that already exists, MakeBucket calls = %d", putCalls)
	}
}

func TestNewFailsWhenBucketCannotBeCreated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodHead && trimmedPath(r) == "/gsnpeeps":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodPut && trimmedPath(r) == "/gsnpeeps":
			w.WriteHeader(http.StatusForbidden)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if _, err := New(context.Background(), testConfig(server.URL)); err == nil {
		t.Fatal("New() error = nil, want bucket creation error")
	}
}

func TestUploadThenDownloadRoundTrip(t *testing.T) {
	const content = "jpeg-bytes"
	const objectPath = "/gsnpeeps/employee-photos/photo.jpg"
	var putCalls, headCalls, getCalls int
	client := newTestClient(t, bucketExistsHandler(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && trimmedPath(r) == objectPath:
			putCalls++
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodHead && trimmedPath(r) == objectPath:
			// Download() calls Stat() before returning the reader, which minio-go serves via
			// StatObject (HEAD); the actual bytes are only fetched on the caller's first Read
			// (GET), so both methods must be handled for the same object path.
			headCalls++
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			w.Header().Set("Content-Length", "10")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && trimmedPath(r) == objectPath:
			getCalls++
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(content))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	location, err := client.Upload(context.Background(), "employee-photos/photo.jpg", strings.NewReader(content), "image/jpeg")
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if location != "employee-photos/photo.jpg" {
		t.Fatalf("Upload() location = %q, want relative object path without a bucket/root-folder prefix", location)
	}
	if putCalls != 1 {
		t.Fatalf("PUT calls = %d, want 1", putCalls)
	}

	body, contentType, err := client.Download(context.Background(), location)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	defer body.Close()
	if contentType != "image/jpeg" {
		t.Fatalf("Download() contentType = %q, want image/jpeg", contentType)
	}
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read downloaded body error = %v", err)
	}
	if string(got) != content {
		t.Fatalf("downloaded body = %q, want %q", got, content)
	}
	if headCalls != 1 {
		t.Fatalf("HEAD calls = %d, want 1", headCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GET calls = %d, want 1", getCalls)
	}
}

func TestDownloadPropagatesNotFound(t *testing.T) {
	client := newTestClient(t, bucketExistsHandler(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	if _, _, err := client.Download(context.Background(), "employee-photos/missing.jpg"); err == nil {
		t.Fatal("Download() error = nil, want not-found error")
	}
}

func TestDeleteRemovesObject(t *testing.T) {
	var deleteCalls int
	client := newTestClient(t, bucketExistsHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && trimmedPath(r) == "/gsnpeeps/employee-documents/emp-1/doc.pdf" {
			deleteCalls++
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
	}))

	if err := client.Delete(context.Background(), "employee-documents/emp-1/doc.pdf"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleteCalls != 1 {
		t.Fatalf("DELETE calls = %d, want 1", deleteCalls)
	}
}

func TestUploadRejectsPathTraversal(t *testing.T) {
	client := newTestClient(t, bucketExistsHandler(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request %s %s, traversal must be rejected before any HTTP call", r.Method, r.URL.Path)
	}))

	if _, err := client.Upload(context.Background(), "../secret", strings.NewReader("x"), "text/plain"); err == nil {
		t.Fatal("Upload() error = nil, want path traversal rejected")
	}
}

func TestDeleteRejectsPathTraversal(t *testing.T) {
	client := newTestClient(t, bucketExistsHandler(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request %s %s, traversal must be rejected before any HTTP call", r.Method, r.URL.Path)
	}))

	if err := client.Delete(context.Background(), "../secret"); err == nil {
		t.Fatal("Delete() error = nil, want path traversal rejected")
	}
}

func TestHealthReportsBucketStatus(t *testing.T) {
	healthy := newTestClient(t, bucketExistsHandler(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	if err := healthy.Health(context.Background()); err != nil {
		t.Fatalf("Health() error = %v, want nil", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	// Klien dibangun langsung (bukan lewat New) supaya bootstrap ensureBucket tidak ikut
	// gagal duluan; test ini hanya menguji Health() terhadap bucket yang sudah hilang setelah
	// klien dibuat.
	rawClient, err := miniogo.New(strings.TrimPrefix(server.URL, "http://"), &miniogo.Options{
		Creds:  credentials.NewStaticV4("service", "not-a-real-secret", ""),
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("miniogo.New() error = %v", err)
	}
	unhealthy := &Client{client: rawClient, bucket: "gsnpeeps"}
	if err := unhealthy.Health(context.Background()); err == nil {
		t.Fatal("Health() error = nil, want bucket-not-found error")
	}
}
