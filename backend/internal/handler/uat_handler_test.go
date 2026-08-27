package handler

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

type memoryMultipartFile struct{ *bytes.Reader }

func (memoryMultipartFile) Close() error { return nil }

func TestCorrectedAttendanceTimeConvertsJakartaClockToUTC(t *testing.T) {
	t.Parallel()

	got, err := correctedAttendanceTime("2026-08-14", "09:15")
	if err != nil {
		t.Fatalf("correctedAttendanceTime() error = %v", err)
	}
	want := time.Date(2026, time.August, 14, 2, 15, 0, 0, time.UTC)
	if !got.Equal(want) || got.Location() != time.UTC {
		t.Fatalf("correctedAttendanceTime() = %v (%v), want %v (UTC)", got, got.Location(), want)
	}
}

func TestCorrectedAttendanceTimeRejectsInvalidClock(t *testing.T) {
	t.Parallel()

	if _, err := correctedAttendanceTime("2026-08-14", "25:00"); err == nil {
		t.Fatal("correctedAttendanceTime() error = nil, want invalid clock error")
	}
}

func TestReadFeedAttachmentAcceptsPDF(t *testing.T) {
	t.Parallel()
	content := []byte("%PDF-1.7\nsynthetic")
	header := &multipart.FileHeader{Filename: "kebijakan.pdf", Size: int64(len(content)), Header: make(textproto.MIMEHeader)}
	header.Header.Set("Content-Type", "application/pdf")
	recorder := httptest.NewRecorder()

	upload, ok := readFeedAttachment(recorder, memoryMultipartFile{bytes.NewReader(content)}, header)
	if !ok {
		t.Fatalf("readFeedAttachment() rejected valid PDF: %s", recorder.Body.String())
	}
	if upload.MediaType != "application/pdf" || upload.Extension != ".pdf" || upload.FileName != "kebijakan.pdf" {
		t.Fatalf("readFeedAttachment() = %#v", upload)
	}
}

func TestReadFeedAttachmentRejectsMismatchedSignature(t *testing.T) {
	t.Parallel()
	content := []byte("not-a-png")
	header := &multipart.FileHeader{Filename: "poster.png", Size: int64(len(content)), Header: make(textproto.MIMEHeader)}
	header.Header.Set("Content-Type", "image/png")
	recorder := httptest.NewRecorder()

	if _, ok := readFeedAttachment(recorder, memoryMultipartFile{bytes.NewReader(content)}, header); ok {
		t.Fatal("readFeedAttachment() accepted invalid PNG signature")
	}
	if recorder.Code != 415 {
		t.Fatalf("status = %d, want 415", recorder.Code)
	}
}

func TestStaticFileOwnerParsesNamespacedPaths(t *testing.T) {
	t.Parallel()
	emp := "11111111-1111-1111-1111-111111111111"
	cases := []struct {
		path     string
		wantKind string
		wantID   string
		wantOK   bool
	}{
		{"employee-photos/" + emp + "/a.jpg", "employee", emp, true},
		{"employee-documents/" + emp + "/a.pdf", "employee", emp, true},
		{"attendance-photos/" + emp + "/2026-08-01/x.jpg", "user", emp, true},
		{"leave-documents/" + emp + "/x.pdf", "user", emp, true},
		{"overtime-documents/" + emp + "/x.pdf", "user", emp, true},
		{"GSNpeeps/employee-photos/" + emp + "/a.jpg", "employee", emp, true},
		{"company-feed/22222222-2222-2222-2222-222222222222/a.png", "feed", "", true},
		{"random/thing.txt", "", "", false},
		{"employee-photos/not-a-uuid/a.jpg", "", "", false},
	}
	for _, tc := range cases {
		id, kind, ok := staticFileOwner(tc.path)
		if ok != tc.wantOK || kind != tc.wantKind {
			t.Fatalf("staticFileOwner(%q) = (%v, %q, %v), want (_, %q, %v)", tc.path, id, kind, ok, tc.wantKind, tc.wantOK)
		}
		if tc.wantID != "" && id.String() != tc.wantID {
			t.Fatalf("staticFileOwner(%q) id = %s, want %s", tc.path, id, tc.wantID)
		}
	}
}

func TestMediaAllowedShortCircuits(t *testing.T) {
	t.Parallel()
	h := &UATHandler{}
	emp := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	other := "22222222-2222-2222-2222-222222222222"
	identity := func(role domain.RoleName) domain.Identity {
		return domain.Identity{UserID: emp, EmployeeID: emp, Role: role}
	}

	if !h.mediaAllowed(context.Background(), identity(domain.RoleHR), "employee-documents/"+other+"/a.pdf") {
		t.Fatal("HR should read any static file")
	}
	if !h.mediaAllowed(context.Background(), identity(domain.RoleTopManagement), "employee-documents/"+other+"/a.pdf") {
		t.Fatal("Top Management should read any static file")
	}
	if !h.mediaAllowed(context.Background(), identity(domain.RoleEmployee), "company-feed/"+other+"/a.png") {
		t.Fatal("any role should read company-feed attachments")
	}
	if !h.mediaAllowed(context.Background(), identity(domain.RoleEmployee), "employee-photos/"+emp.String()+"/a.jpg") {
		t.Fatal("karyawan should read own file")
	}
	if h.mediaAllowed(context.Background(), identity(domain.RoleEmployee), "random/thing.txt") {
		t.Fatal("unknown namespace must be denied for non-HR")
	}
}
