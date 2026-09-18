package config

import (
	"strings"
	"testing"
)

func TestLoadValidConfigDefaultsToMinIO(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGIN", "http://localhost:5173")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/app")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("MINIO_ENDPOINT", "minio:9000")
	t.Setenv("MINIO_ACCESS_KEY", "service")
	t.Setenv("MINIO_SECRET_KEY", "not-a-real-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Storage.Driver != StorageDriverMinIO {
		t.Fatalf("Storage.Driver = %q, want %q", cfg.Storage.Driver, StorageDriverMinIO)
	}
	if cfg.MinIO.Bucket != "gsnpeeps" {
		t.Fatalf("MinIO.Bucket = %q, want default gsnpeeps", cfg.MinIO.Bucket)
	}
}

func TestLoadValidConfigNextcloudDriver(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGIN", "http://localhost:5173")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/app")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("STORAGE_DRIVER", "nextcloud")
	t.Setenv("NEXTCLOUD_WEBDAV_URL", "http://nextcloud/remote.php/dav/files/service")
	t.Setenv("NEXTCLOUD_USERNAME", "service")
	t.Setenv("NEXTCLOUD_APP_PASSWORD", "not-a-real-secret")

	if _, err := Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadReportsMissingNamesWithoutValues(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGIN", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("MINIO_ENDPOINT", "")
	t.Setenv("MINIO_ACCESS_KEY", "")
	t.Setenv("MINIO_SECRET_KEY", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("Load() error = %v, want missing DATABASE_URL", err)
	}
}

func TestLoadDefaultsAttendancePolicy(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGIN", "http://localhost:5173")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/app")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("MINIO_ENDPOINT", "minio:9000")
	t.Setenv("MINIO_ACCESS_KEY", "service")
	t.Setenv("MINIO_SECRET_KEY", "not-a-real-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Attendance.WFORadiusMeters != 500 {
		t.Fatalf("Attendance.WFORadiusMeters = %v, want default 500", cfg.Attendance.WFORadiusMeters)
	}
	if cfg.Attendance.WorkStartHour != 9 || cfg.Attendance.WorkStartMinute != 0 {
		t.Fatalf("Attendance work start = %d:%d, want 9:0", cfg.Attendance.WorkStartHour, cfg.Attendance.WorkStartMinute)
	}
	if cfg.Attendance.WorkEndHour != 18 || cfg.Attendance.WorkEndMinute != 0 {
		t.Fatalf("Attendance work end = %d:%d, want 18:0", cfg.Attendance.WorkEndHour, cfg.Attendance.WorkEndMinute)
	}
}

func TestLoadOverridesAttendancePolicyFromEnv(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGIN", "http://localhost:5173")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/app")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("MINIO_ENDPOINT", "minio:9000")
	t.Setenv("MINIO_ACCESS_KEY", "service")
	t.Setenv("MINIO_SECRET_KEY", "not-a-real-secret")
	t.Setenv("ATTENDANCE_WFO_RADIUS_METERS", "750.5")
	t.Setenv("ATTENDANCE_WORK_START_HOUR", "8")
	t.Setenv("ATTENDANCE_WORK_START_MINUTE", "30")
	t.Setenv("ATTENDANCE_WORK_END_HOUR", "17")
	t.Setenv("ATTENDANCE_WORK_END_MINUTE", "45")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Attendance.WFORadiusMeters != 750.5 {
		t.Fatalf("Attendance.WFORadiusMeters = %v, want 750.5", cfg.Attendance.WFORadiusMeters)
	}
	if cfg.Attendance.WorkStartHour != 8 || cfg.Attendance.WorkStartMinute != 30 {
		t.Fatalf("Attendance work start = %d:%d, want 8:30", cfg.Attendance.WorkStartHour, cfg.Attendance.WorkStartMinute)
	}
	if cfg.Attendance.WorkEndHour != 17 || cfg.Attendance.WorkEndMinute != 45 {
		t.Fatalf("Attendance work end = %d:%d, want 17:45", cfg.Attendance.WorkEndHour, cfg.Attendance.WorkEndMinute)
	}
}

func TestLoadRejectsInvalidAttendancePolicy(t *testing.T) {
	base := func() {
		t.Setenv("CORS_ALLOWED_ORIGIN", "http://localhost:5173")
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/app")
		t.Setenv("REDIS_URL", "redis://localhost:6379/0")
		t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
		t.Setenv("MINIO_ENDPOINT", "minio:9000")
		t.Setenv("MINIO_ACCESS_KEY", "service")
		t.Setenv("MINIO_SECRET_KEY", "not-a-real-secret")
	}

	t.Run("radius zero", func(t *testing.T) {
		base()
		t.Setenv("ATTENDANCE_WFO_RADIUS_METERS", "0")
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "ATTENDANCE_WFO_RADIUS_METERS") {
			t.Fatalf("Load() error = %v, want ATTENDANCE_WFO_RADIUS_METERS error", err)
		}
	})

	t.Run("end before start", func(t *testing.T) {
		base()
		t.Setenv("ATTENDANCE_WORK_START_HOUR", "18")
		t.Setenv("ATTENDANCE_WORK_END_HOUR", "9")
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "ATTENDANCE_WORK_END_HOUR") {
			t.Fatalf("Load() error = %v, want ATTENDANCE_WORK_END_HOUR error", err)
		}
	})
}

func TestLoadRejectsUnsupportedStorageDriver(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGIN", "http://localhost:5173")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/app")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("STORAGE_DRIVER", "s3")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "STORAGE_DRIVER") {
		t.Fatalf("Load() error = %v, want unsupported STORAGE_DRIVER error", err)
	}
}
