package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/elpdev/telex-cli/internal/config"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client

	cfg       *config.Config
	tokenPath string
	mu        sync.Mutex
	token     string
	tokenExp  time.Time
}

func NewClient(cfg *config.Config, tokenPath string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		cfg:        cfg,
		tokenPath:  tokenPath,
	}
}

func (c *Client) Authenticate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.authenticate(ctx)
}

func (c *Client) Get(ctx context.Context, path string, params url.Values) ([]byte, int, error) {
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	return c.do(ctx, http.MethodGet, path, nil)
}

func (c *Client) Post(ctx context.Context, path string, body any) ([]byte, int, error) {
	return c.do(ctx, http.MethodPost, path, body)
}

func (c *Client) Patch(ctx context.Context, path string, body any) ([]byte, int, error) {
	return c.do(ctx, http.MethodPatch, path, body)
}

func (c *Client) Delete(ctx context.Context, path string) (int, error) {
	_, status, err := c.do(ctx, http.MethodDelete, path, nil)
	return status, err
}

func (c *Client) PostMultipartFile(ctx context.Context, path, fieldName, filePath string) ([]byte, int, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return nil, 0, err
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, err
	}
	defer closeSilently(file)
	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return nil, 0, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, 0, err
	}
	if err := writer.Close(); err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, &body)
	if err != nil {
		return nil, 0, fmt.Errorf("creating multipart request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request POST %s: %w", path, err)
	}
	defer closeSilently(resp.Body)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response body: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return respBody, resp.StatusCode, parseError(resp.StatusCode, respBody)
	}
	return respBody, resp.StatusCode, nil
}

func (c *Client) Download(ctx context.Context, rawURL string) ([]byte, string, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return nil, "", err
	}
	downloadURL, sameOrigin, err := c.resolveDownloadURL(rawURL)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("creating download request: %w", err)
	}
	if sameOrigin {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("downloading %s: %w", rawURL, err)
	}
	defer closeSilently(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.Header.Get("Content-Type"), fmt.Errorf("reading download response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, resp.Header.Get("Content-Type"), parseError(resp.StatusCode, body)
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func (c *Client) resolveDownloadURL(rawURL string) (string, bool, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", false, err
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", false, err
	}
	if !parsed.IsAbs() {
		return base.ResolveReference(parsed).String(), true, nil
	}
	return parsed.String(), parsed.Scheme == base.Scheme && parsed.Host == base.Host, nil
}

func (c *Client) ensureAuth(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Until(c.tokenExp) > 60*time.Second {
		return nil
	}
	if tc, err := config.LoadTokenFrom(c.tokenPath); err == nil && tc.Valid() {
		c.token = tc.Token
		c.tokenExp = tc.ExpiresAt
		return nil
	}
	return c.authenticate(ctx)
}

func (c *Client) authenticate(ctx context.Context) error {
	payload, err := json.Marshal(map[string]string{"client_id": c.cfg.ClientID, "secret_key": c.cfg.SecretKey})
	if err != nil {
		return fmt.Errorf("marshal auth payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/auth/token", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating auth request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("authenticating: %w", err)
	}
	defer closeSilently(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading auth response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return parseError(resp.StatusCode, body)
	}
	var result struct {
		Token     string `json:"token"`
		ExpiresIn int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing auth response: %w", err)
	}
	if result.Token == "" {
		return fmt.Errorf("parsing auth response: missing token")
	}
	c.token = result.Token
	c.tokenExp = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	if err := config.SaveTokenTo(c.tokenPath, &config.TokenCache{Token: c.token, ExpiresAt: c.tokenExp}); err != nil {
		return fmt.Errorf("saving token cache: %w", err)
	}
	return nil
}

func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return nil, 0, err
	}
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request payload: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer closeSilently(resp.Body)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response body: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return respBody, resp.StatusCode, parseError(resp.StatusCode, respBody)
	}
	return respBody, resp.StatusCode, nil
}

func closeSilently(closer io.Closer) { _ = closer.Close() }
