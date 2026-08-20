// Package filestore adalah composition-root port untuk object storage. Service dan handler
// tetap memakai interface sempit milik mereka sendiri (mis. service.DocumentStore,
// worker.PhotoDeleter); Store di sini hanya dipakai cmd/api dan cmd/worker untuk memilih dan
// mengembalikan satu adapter konkret yang memenuhi seluruh interface sempit tersebut sekaligus.
package filestore

import (
	"context"
	"fmt"
	"io"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/platform/minio"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/platform/webdav"
)

// Store adalah union method yang dibutuhkan seluruh konsumen object storage di aplikasi.
type Store interface {
	Upload(ctx context.Context, objectPath string, body io.Reader, contentType string) (string, error)
	Download(ctx context.Context, storedPath string) (io.ReadCloser, string, error)
	Delete(ctx context.Context, objectPath string) error
	Health(ctx context.Context) error
}

// New memilih adapter object storage aktif berdasarkan cfg.Storage.Driver. MinIO adalah
// default; Nextcloud/WebDAV dipertahankan untuk deployment yang belum bermigrasi.
func New(ctx context.Context, cfg config.Config) (Store, error) {
	switch cfg.Storage.Driver {
	case config.StorageDriverMinIO:
		client, err := minio.New(ctx, cfg.MinIO)
		if err != nil {
			return nil, fmt.Errorf("minio adapter: %w", err)
		}
		return client, nil
	case config.StorageDriverNextcloud:
		client, err := webdav.New(cfg.Nextcloud)
		if err != nil {
			return nil, fmt.Errorf("nextcloud adapter: %w", err)
		}
		return client, nil
	default:
		return nil, fmt.Errorf("unsupported STORAGE_DRIVER %q", cfg.Storage.Driver)
	}
}
