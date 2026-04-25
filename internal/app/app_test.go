package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestSwitchScreenForTest(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model = model.SwitchScreenForTest("settings")

	if model.CurrentScreenID() != "settings" {
		t.Fatalf("expected settings screen, got %q", model.CurrentScreenID())
	}
}

func TestRouteRunsScreenInitCommand(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	updated, cmd := model.Update(routeMsg{ScreenID: "mail"})
	model = updated.(Model)

	if model.CurrentScreenID() != "mail" {
		t.Fatalf("expected mail screen, got %q", model.CurrentScreenID())
	}
	if cmd == nil {
		t.Fatal("expected route to run screen init command")
	}
}

func TestCommandPaletteThemePreviewCanReturnToRoot(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model = openThemePalette(t, model)

	model = sendKey(t, model, tea.Key{Code: tea.KeyDown})
	if model.theme.Name != "Muted Dark" {
		t.Fatalf("expected preview to switch to Muted Dark, got %q", model.theme.Name)
	}

	model = sendKey(t, model, tea.Key{Code: tea.KeyEscape})
	if model.theme.Name != "Phosphor" {
		t.Fatalf("expected esc to restore Phosphor theme, got %q", model.theme.Name)
	}
	if !model.showCommandPalette {
		t.Fatal("expected esc from theme page to keep command palette open")
	}
	if model.commandPalette.Action().Type != 0 {
		t.Fatal("expected palette action to be cleared after handling")
	}

	model = sendKey(t, model, tea.Key{Code: tea.KeyEscape})
	if model.showCommandPalette {
		t.Fatal("expected esc from root command palette to close palette")
	}
}

func TestCommandPaletteThemeSelectionConfirmsPreview(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model = openThemePalette(t, model)

	model = sendKey(t, model, tea.Key{Code: tea.KeyDown})
	model = sendKey(t, model, tea.Key{Code: tea.KeyDown})
	model = sendKey(t, model, tea.Key{Code: tea.KeyEnter})

	if model.theme.Name != "Miami" {
		t.Fatalf("expected confirmed Miami theme, got %q", model.theme.Name)
	}
	if model.showCommandPalette {
		t.Fatal("expected theme selection to close command palette")
	}
}

func openThemePalette(t *testing.T, model Model) Model {
	t.Helper()
	model = sendKey(t, model, tea.Key{Code: 'k', Mod: tea.ModCtrl})
	if !model.showCommandPalette {
		t.Fatal("expected command palette to open")
	}

	for _, r := range "theme" {
		model = sendKey(t, model, tea.Key{Text: string(r), Code: r})
	}
	model = sendKey(t, model, tea.Key{Code: tea.KeyEnter})
	return model
}

func sendKey(t *testing.T, model Model, key tea.Key) Model {
	t.Helper()
	updated, cmd := model.Update(keyPress(key))
	model = updated.(Model)
	for cmd != nil {
		updated, cmd = model.Update(cmd())
		model = updated.(Model)
	}
	return model
}

func keyPress(key tea.Key) tea.KeyPressMsg {
	return tea.KeyPressMsg(key)
}
