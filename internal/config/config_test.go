package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestPathsUsesOverrideDirectoryForToken(t *testing.T) {
	configPath, tokenPath := Paths("/tmp/custom/telex.toml")
	if configPath != filepath.Clean("/tmp/custom/telex.toml") {
		t.Fatalf("config path = %q", configPath)
	}
	if tokenPath != filepath.Clean("/tmp/custom/token.toml") {
		t.Fatalf("token path = %q", tokenPath)
	}
}

func TestConfigValidateRequiresCredentials(t *testing.T) {
	cfg := &Config{BaseURL: "http://localhost:3000", ClientID: "id"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing secret_key error")
	}
}

func TestDriveSyncModeDefaultsToFull(t *testing.T) {
	cfg := &Config{}
	if got := cfg.DriveSyncMode(); got != DriveSyncFull {
		t.Fatalf("drive sync mode = %q, want %q", got, DriveSyncFull)
	}
}

func TestConfigValidateRejectsInvalidDriveSyncMode(t *testing.T) {
	cfg := &Config{BaseURL: "http://localhost:3000", ClientID: "id", SecretKey: "secret", Drive: DriveConfig{SyncMode: "headers_only"}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid drive.sync_mode error")
	}
}

func TestTokenCacheValid(t *testing.T) {
	tc := &TokenCache{Token: "token", ExpiresAt: time.Now().Add(2 * time.Minute)}
	if !tc.Valid() {
		t.Fatal("expected token to be valid")
	}
	tc.ExpiresAt = time.Now().Add(30 * time.Second)
	if tc.Valid() {
		t.Fatal("expected expiring token to be invalid")
	}
}
