package handler

import (
	"testing"
	"time"
)

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
