package screens

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/theme"
)

func newTestSettings() Settings {
	state := SettingsState{
		ThemeName:      "Phosphor",
		SidebarVisible: true,
		Instance:       "mail.example.com",
		AuthStatus:     "Signed in",
		DataDir:        "/tmp/data",
		ConfigDir:      "/tmp/config",
		DriveSyncMode:  driveSyncFull,
		Version:        "v0.0.0",
	}
	return NewSettings(state, theme.Phosphor(), theme.BuiltIns(), SettingsActions{})
}

func keyMsg(k tea.Key) tea.KeyPressMsg { return tea.KeyPressMsg(k) }

func runUpdate(t *testing.T, s Settings, msg tea.Msg) (Settings, tea.Msg) {
	t.Helper()
	updated, cmd := s.Update(msg)
	out := updated.(Settings)
	if cmd == nil {
		return out, nil
	}
	return out, cmd()
}

func TestSettingsInitialFocusIsFirstFocusableRow(t *testing.T) {
	s := newTestSettings()
	if got := s.focusedRow().id; got != "theme" {
		t.Fatalf("initial focused row = %q, want %q", got, "theme")
	}
}

func TestSettingsCursorSkipsSectionHeaders(t *testing.T) {
	s := newTestSettings()
	// Theme -> Sidebar -> Instance: section header between sidebar and instance must not consume a step.
	s, _ = runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyDown}))
	if got := s.focusedRow().id; got != "sidebar-visible" {
		t.Fatalf("after one down, focus = %q, want %q", got, "sidebar-visible")
	}
	s, _ = runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyDown}))
	if got := s.focusedRow().id; got != "instance" {
		t.Fatalf("after two downs, focus = %q (want %q) — cursor likely landed on a section header", got, "instance")
	}
}

func TestSettingsToggleEmitsSidebarChangedMsg(t *testing.T) {
	s := newTestSettings()
	s, _ = runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyDown})) // theme -> sidebar
	_, msg := runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyEnter}))
	got, ok := msg.(SettingsSidebarChangedMsg)
	if !ok {
		t.Fatalf("expected SettingsSidebarChangedMsg, got %T", msg)
	}
	if got.Visible != false {
		t.Fatalf("toggle from visible should produce Visible=false, got %v", got.Visible)
	}
}

func TestSettingsThemeSelectEnterAndEsc(t *testing.T) {
	s := newTestSettings()
	// Enter theme select mode
	s, _ = runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyEnter}))
	if s.mode != settingsModeThemes {
		t.Fatalf("expected mode = themes after enter on theme row")
	}
	// Down -> emits preview
	s, msg := runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyDown}))
	if _, ok := msg.(SettingsThemePreviewMsg); !ok {
		t.Fatalf("expected SettingsThemePreviewMsg on down, got %T", msg)
	}
	// Enter -> emits change
	_, changedMsg := runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyEnter}))
	if _, ok := changedMsg.(SettingsThemeChangedMsg); !ok {
		t.Fatalf("expected SettingsThemeChangedMsg on enter, got %T", changedMsg)
	}

	// Re-enter theme picker, esc reverts
	s = newTestSettings()
	s, _ = runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyEnter}))
	s, _ = runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyDown}))
	_, cancelMsg := runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyEscape}))
	cancel, ok := cancelMsg.(SettingsThemeCancelMsg)
	if !ok {
		t.Fatalf("expected SettingsThemeCancelMsg on esc, got %T", cancelMsg)
	}
	if cancel.Name != "Phosphor" {
		t.Fatalf("cancel name = %q, want %q (original)", cancel.Name, "Phosphor")
	}
}

func TestSettingsDriveSyncCycles(t *testing.T) {
	s := newTestSettings()
	// Cursor: theme, sidebar, instance, status, sign-out, data-dir, cache-size, drive-sync (7 downs)
	for i := 0; i < 7; i++ {
		s, _ = runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyDown}))
	}
	if got := s.focusedRow().id; got != "drive-sync" {
		t.Fatalf("focus after 7 downs = %q, want %q", got, "drive-sync")
	}
	_, msg := runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyEnter}))
	got, ok := msg.(SettingsDriveSyncChangedMsg)
	if !ok {
		t.Fatalf("expected SettingsDriveSyncChangedMsg, got %T", msg)
	}
	if got.Mode != driveSyncMetadataOnly {
		t.Fatalf("drive sync next mode = %q, want %q", got.Mode, driveSyncMetadataOnly)
	}
}

func TestSettingsSignOutRequiresConfirmation(t *testing.T) {
	called := false
	actions := SettingsActions{
		SignOut: func() tea.Cmd {
			return func() tea.Msg { called = true; return nil }
		},
	}
	s := NewSettings(SettingsState{ThemeName: "Phosphor"}, theme.Phosphor(), theme.BuiltIns(), actions)
	// Move to sign-out row: theme, sidebar, instance, status, sign-out (4 downs)
	for i := 0; i < 4; i++ {
		s, _ = runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyDown}))
	}
	if got := s.focusedRow().id; got != "sign-out" {
		t.Fatalf("focus after 4 downs = %q, want %q", got, "sign-out")
	}
	// First enter: arms confirmation, no cmd
	s, msg := runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyEnter}))
	if msg != nil {
		t.Fatalf("expected no cmd on first enter, got %T", msg)
	}
	if s.confirming != "sign-out" {
		t.Fatalf("expected confirming flag, got %q", s.confirming)
	}
	// Second enter: runs callback
	_, _ = runUpdate(t, s, keyMsg(tea.Key{Code: tea.KeyEnter}))
	if !called {
		t.Fatal("expected SignOut callback to run after second enter")
	}
}
