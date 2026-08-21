package handler

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"
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
