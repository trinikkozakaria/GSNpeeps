package config

import (
	"strings"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGIN", "http://localhost:5173")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/app")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
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
	t.Setenv("NEXTCLOUD_WEBDAV_URL", "")
	t.Setenv("NEXTCLOUD_USERNAME", "")
	t.Setenv("NEXTCLOUD_APP_PASSWORD", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("Load() error = %v, want missing DATABASE_URL", err)
	}
}
