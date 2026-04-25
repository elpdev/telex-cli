package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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

func testClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	tokenPath := filepath.Join(t.TempDir(), "token.toml")
	if err := config.SaveTokenTo(tokenPath, &config.TokenCache{Token: "token", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	return NewClient(&config.Config{BaseURL: baseURL, ClientID: "id", SecretKey: "secret"}, tokenPath)
}
