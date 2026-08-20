package webdav

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/objectpath"
)

type Client struct {
	baseURL    *url.URL
	username   string
	password   string
	rootFolder string
	httpClient *http.Client
}

func New(cfg config.Nextcloud) (*Client, error) {
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, errors.New("invalid NEXTCLOUD_WEBDAV_URL")
	}
	root, err := objectpath.SafePath(cfg.RootFolder)
	if err != nil {
		return nil, fmt.Errorf("invalid NEXTCLOUD_ROOT_FOLDER: %w", err)
	}
	return &Client{
		baseURL:    baseURL,
		username:   cfg.Username,
		password:   cfg.AppPassword,
		rootFolder: root,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
			// Redirect WebDAV ke halaman setup/login harus dianggap gagal. Jika diikuti,
			// halaman HTML dapat salah dianggap sebagai unggahan atau unduhan berhasil.
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (c *Client) Upload(ctx context.Context, objectPath string, body io.Reader, contentType string) (string, error) {
	target, err := c.objectURL(objectPath)
	if err != nil {
		return "", err
	}
	if err := c.ensureCollections(ctx, objectPath); err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, target, body)
	if err != nil {
		return "", fmt.Errorf("create WebDAV upload request: %w", err)
	}
	request.SetBasicAuth(c.username, c.password)
	request.Header.Set("Content-Type", contentType)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("upload WebDAV object: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("upload WebDAV object: unexpected status %d", response.StatusCode)
	}
	return path.Join(c.rootFolder, objectPath), nil
}

// ensureCollections membuat rootFolder dan setiap direktori antara sebelum PUT, karena WebDAV
// menolak menulis objek bila koleksi induknya belum ada (Nextcloud mengembalikan 404/409, bukan
// membuatnya otomatis seperti filesystem lokal).
func (c *Client) ensureCollections(ctx context.Context, objectPath string) error {
	object, err := objectpath.SafePath(objectPath)
	if err != nil {
		return err
	}
	segments := strings.Split(c.rootFolder, "/")
	if dir := path.Dir(object); dir != "." {
		segments = append(segments, strings.Split(dir, "/")...)
	}
	current := ""
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		current = path.Join(current, segment)
		if err := c.mkcol(ctx, current); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) mkcol(ctx context.Context, relativeFromBase string) error {
	target := *c.baseURL
	target.Path = path.Join(target.Path, relativeFromBase)
	request, err := http.NewRequestWithContext(ctx, "MKCOL", target.String(), nil)
	if err != nil {
		return fmt.Errorf("create WebDAV collection request: %w", err)
	}
	request.SetBasicAuth(c.username, c.password)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("create WebDAV collection: %w", err)
	}
	defer response.Body.Close()
	// 201 Created bila koleksi baru dibuat, 405 Method Not Allowed bila sudah ada
	// sebelumnya (mis. upload berikutnya untuk karyawan yang sama); keduanya bukan
	// kegagalan. Traversal berjalan top-down sehingga parent selalu sudah dibuat lebih
	// dahulu sebelum child dicoba.
	if response.StatusCode == http.StatusCreated || response.StatusCode == http.StatusMethodNotAllowed {
		return nil
	}
	return fmt.Errorf("create WebDAV collection: unexpected status %d", response.StatusCode)
}

func (c *Client) Delete(ctx context.Context, objectPath string) error {
	target, err := c.objectURL(objectPath)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, target, nil)
	if err != nil {
		return fmt.Errorf("create WebDAV delete request: %w", err)
	}
	request.SetBasicAuth(c.username, c.password)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("delete WebDAV object: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("delete WebDAV object: unexpected status %d", response.StatusCode)
	}
	return nil
}

// Download reads a stored locator returned by Upload through the authenticated WebDAV client.
func (c *Client) Download(ctx context.Context, storedPath string) (io.ReadCloser, string, error) {
	cleaned, err := objectpath.SafePath(storedPath)
	if err != nil {
		return nil, "", err
	}
	target := *c.baseURL
	target.Path = path.Join(target.Path, cleaned)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, "", err
	}
	request.SetBasicAuth(c.username, c.password)
	result, err := c.httpClient.Do(request)
	if err != nil {
		return nil, "", fmt.Errorf("download WebDAV object: %w", err)
	}
	if result.StatusCode < 200 || result.StatusCode >= 300 {
		result.Body.Close()
		return nil, "", fmt.Errorf("download WebDAV object: status %d", result.StatusCode)
	}
	return result.Body, result.Header.Get("Content-Type"), nil
}

func (c *Client) Health(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, "PROPFIND", c.baseURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create WebDAV health request: %w", err)
	}
	request.SetBasicAuth(c.username, c.password)
	request.Header.Set("Depth", "0")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("check WebDAV health: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return fmt.Errorf("check WebDAV health: unexpected status %d", response.StatusCode)
	}
	return nil
}

func (c *Client) objectURL(objectPath string) (string, error) {
	object, err := objectpath.SafePath(objectPath)
	if err != nil {
		return "", err
	}
	target := *c.baseURL
	target.Path = path.Join(target.Path, c.rootFolder, object)
	return target.String(), nil
}
