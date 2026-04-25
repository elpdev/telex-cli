package config

import (
	"path/filepath"
	"testing"
)

func TestLoadPrefsReturnsZeroValueWhenMissing(t *testing.T) {
	prefs, err := LoadPrefs(filepath.Join(t.TempDir(), "prefs.toml"))
	if err != nil {
		t.Fatalf("LoadPrefs returned error: %v", err)
	}
	if prefs == nil {
		t.Fatal("LoadPrefs returned nil prefs")
	}
	if prefs.Theme != "" || prefs.SidebarVisible != nil {
		t.Fatalf("expected zero-value prefs, got %+v", prefs)
	}
}

func TestUIPrefsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prefs.toml")
	visible := false
	original := &UIPrefs{Theme: "Miami", SidebarVisible: &visible}
	if err := original.SaveTo(path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}
	loaded, err := LoadPrefs(path)
	if err != nil {
		t.Fatalf("LoadPrefs: %v", err)
	}
	if loaded.Theme != "Miami" {
		t.Fatalf("theme = %q, want Miami", loaded.Theme)
	}
	if loaded.SidebarVisible == nil || *loaded.SidebarVisible != false {
		t.Fatalf("sidebar visible pointer = %v, want pointer to false", loaded.SidebarVisible)
	}
}

func TestPrefsPathForOverride(t *testing.T) {
	got := PrefsPathFor("/tmp/custom/config.toml")
	want := filepath.Clean("/tmp/custom/prefs.toml")
	if got != want {
		t.Fatalf("PrefsPathFor = %q, want %q", got, want)
	}
}
