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
)

type Storage interface {
	Upload(context.Context, string, io.Reader, string) (string, error)
	Delete(context.Context, string) error
	Health(context.Context) error
}

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
	root, err := safePath(cfg.RootFolder)
	if err != nil {
		return nil, fmt.Errorf("invalid NEXTCLOUD_ROOT_FOLDER: %w", err)
	}
	return &Client{
		baseURL:    baseURL,
		username:   cfg.Username,
		password:   cfg.AppPassword,
		rootFolder: root,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}, nil
}

func (c *Client) Upload(ctx context.Context, objectPath string, body io.Reader, contentType string) (string, error) {
	target, err := c.objectURL(objectPath)
	if err != nil {
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
	object, err := safePath(objectPath)
	if err != nil {
		return "", err
	}
	target := *c.baseURL
	target.Path = path.Join(target.Path, c.rootFolder, object)
	return target.String(), nil
}

func safePath(value string) (string, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if normalized == "" || strings.HasPrefix(normalized, "/") {
		return "", errors.New("path must be relative and non-empty")
	}
	for _, part := range strings.Split(normalized, "/") {
		if part == "" || part == "." || part == ".." {
			return "", errors.New("path traversal is not allowed")
		}
	}
	cleaned := path.Clean(normalized)
	if cleaned == "." || strings.HasPrefix(cleaned, "../") {
		return "", errors.New("path traversal is not allowed")
	}
	return cleaned, nil
}
