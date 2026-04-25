package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/config"
)

func TestDownloadResolvesRelativeURLAndUsesAuth(t *testing.T) {
	var auth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("attachment"))
	}))
	defer server.Close()
	client := testClient(t, server.URL)
	body, contentType, err := client.Download(context.Background(), "/files/1")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "attachment" || contentType != "text/plain; charset=utf-8" {
		t.Fatalf("body=%q contentType=%q", string(body), contentType)
	}
	if auth != "Bearer token" {
		t.Fatalf("auth = %q", auth)
	}
}

func TestPostMultipartFileUploadsFileWithAuth(t *testing.T) {
	var auth string
	var filename string
	var uploaded string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = file.Close() }()
		body, err := io.ReadAll(file)
		if err != nil {
			t.Fatal(err)
		}
		filename = header.Filename
		uploaded = string(body)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	filePath := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	client := testClient(t, server.URL)
	if _, _, err := client.PostMultipartFile(context.Background(), "/upload", "file", filePath); err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer token" || filename != "upload.txt" || uploaded != "hello" {
		t.Fatalf("auth=%q filename=%q uploaded=%q", auth, filename, uploaded)
	}
}

func TestPutRawUploadsWithoutAuth(t *testing.T) {
	var auth string
	var contentType string
	var uploaded string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		contentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		uploaded = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client := testClient(t, server.URL)
	if _, err := client.PutRaw(context.Background(), "/upload", map[string]string{"Content-Type": "text/plain"}, strings.NewReader("hello")); err != nil {
		t.Fatal(err)
	}
	if auth != "" || contentType != "text/plain" || uploaded != "hello" {
		t.Fatalf("auth=%q contentType=%q uploaded=%q", auth, contentType, uploaded)
	}
}

func testClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	tokenPath := filepath.Join(t.TempDir(), "token.toml")
	if err := config.SaveTokenTo(tokenPath, &config.TokenCache{Token: "token", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	return NewClient(&config.Config{BaseURL: baseURL, ClientID: "id", SecretKey: "secret"}, tokenPath)
}
