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
