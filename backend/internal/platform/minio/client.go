// Package minio adalah adapter object storage S3-compatible (MinIO) yang memenuhi
// filestore.Store. Ini adalah driver default sejak keputusan 2026-08-20; lihat
// docs/architecture/minio-integration.md.
package minio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/objectpath"
)

type Client struct {
	client *miniogo.Client
	bucket string
}

// New membuat klien MinIO dan memastikan bucket target ada. Berbeda dari WebDAV yang harus
// MKCOL setiap direktori antara per request, object storage S3-compatible hanya butuh bucket
// itu sendiri dibuat sekali di awal; "/" pada object key hanyalah bagian dari nama key, bukan
// koleksi/direktori nyata yang perlu dibuat terpisah.
func New(ctx context.Context, cfg config.MinIO) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("invalid MINIO_ENDPOINT")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, errors.New("invalid MINIO_ACCESS_KEY or MINIO_SECRET_KEY")
	}
	if cfg.Bucket == "" {
		return nil, errors.New("invalid MINIO_BUCKET")
	}

	rawClient, err := miniogo.New(cfg.Endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	client := &Client{client: rawClient, bucket: cfg.Bucket}

	bootstrapCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	if err := client.ensureBucket(bootstrapCtx, cfg.Region); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) ensureBucket(ctx context.Context, region string) error {
	exists, err := c.client.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("check minio bucket: %w", err)
	}
	if exists {
		return nil
	}
	if err := c.client.MakeBucket(ctx, c.bucket, miniogo.MakeBucketOptions{Region: region}); err != nil {
		// Toleransi race antar instance yang bootstrap bersamaan: bucket bisa saja sudah
		// dibuat instance lain di antara BucketExists dan MakeBucket di atas.
		if exists, existsErr := c.client.BucketExists(ctx, c.bucket); existsErr == nil && exists {
			return nil
		}
		return fmt.Errorf("create minio bucket: %w", err)
	}
	return nil
}

func (c *Client) Upload(ctx context.Context, objectPath string, body io.Reader, contentType string) (string, error) {
	cleaned, err := objectpath.SafePath(objectPath)
	if err != nil {
		return "", err
	}
	if _, err := c.client.PutObject(ctx, c.bucket, cleaned, body, readerSize(body), miniogo.PutObjectOptions{
		ContentType: contentType,
	}); err != nil {
		return "", fmt.Errorf("upload minio object: %w", err)
	}
	return cleaned, nil
}

// readerSize melaporkan panjang body bila diketahui di muka (mis. bytes.Reader dari
// []byte yang sudah divalidasi handler) supaya PutObject memakai PUT tunggal, bukan
// multipart streaming yang dipakai saat ukuran tidak diketahui (-1). Dokumen dan foto
// selalu jauh di bawah 5 MB, sehingga jalur PUT tunggal ini yang selalu dipakai.
func readerSize(body io.Reader) int64 {
	switch typed := body.(type) {
	case *bytes.Reader:
		return int64(typed.Len())
	case *bytes.Buffer:
		return int64(typed.Len())
	case *strings.Reader:
		return int64(typed.Len())
	default:
		return -1
	}
}

// Download reads a stored locator returned by Upload through the MinIO client.
func (c *Client) Download(ctx context.Context, storedPath string) (io.ReadCloser, string, error) {
	cleaned, err := objectpath.SafePath(storedPath)
	if err != nil {
		return nil, "", err
	}
	object, err := c.client.GetObject(ctx, c.bucket, cleaned, miniogo.GetObjectOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("download minio object: %w", err)
	}
	// GetObject tidak melakukan request HTTP nyata sampai dibaca; Stat memaksa request agar
	// objek yang tidak ada terdeteksi di sini, bukan pada Read pertama oleh caller.
	info, err := object.Stat()
	if err != nil {
		object.Close()
		return nil, "", fmt.Errorf("download minio object: %w", err)
	}
	return object, info.ContentType, nil
}

func (c *Client) Delete(ctx context.Context, objectPath string) error {
	cleaned, err := objectpath.SafePath(objectPath)
	if err != nil {
		return err
	}
	if err := c.client.RemoveObject(ctx, c.bucket, cleaned, miniogo.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete minio object: %w", err)
	}
	return nil
}

func (c *Client) Health(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("check minio health: %w", err)
	}
	if !exists {
		return fmt.Errorf("check minio health: bucket %q not found", c.bucket)
	}
	return nil
}
