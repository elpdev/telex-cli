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

func TestNotesScreenRegisteredInNavigationAndCommands(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	if _, ok := model.screens["notes"]; !ok {
		t.Fatal("expected notes screen to be registered")
	}
	if got := model.screenOrder; len(got) < 5 || got[3] != "notes" {
		t.Fatalf("screenOrder = %#v", got)
	}
	for _, id := range []string{"go-notes", "notes-sync", "notes-new", "notes-edit", "notes-delete", "notes-search"} {
		if _, ok := model.commands.Find(id); !ok {
			t.Fatalf("expected command %q", id)
		}
	}
}

func TestCalendarScreenRegisteredInNavigationAndCommands(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	if _, ok := model.screens["calendar"]; !ok {
		t.Fatal("expected calendar screen to be registered")
	}
	if got := model.screenOrder; len(got) < 4 || got[2] != "calendar" {
		t.Fatalf("screenOrder = %#v", got)
	}
	for _, id := range []string{"go-calendar", "calendar-sync", "calendar-view-agenda", "calendar-view-calendars", "calendar-new", "calendar-edit", "calendar-today", "calendar-previous-range", "calendar-next-range", "calendar-filter-agenda", "calendar-clear-agenda-filters", "calendar-delete", "calendar-invitation-show", "calendar-invitation-sync", "calendar-invitation-accept", "calendar-invitation-tentative", "calendar-invitation-decline", "calendar-invitation-needs-action", "calendars-new", "calendars-edit", "calendars-import-ics", "calendars-delete"} {
		if _, ok := model.commands.Find(id); !ok {
			t.Fatalf("expected command %q", id)
		}
	}
}

func TestMailAdminScreenRegisteredInNavigationAndCommands(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	if _, ok := model.screens["mail-admin"]; !ok {
		t.Fatal("expected mail admin screen to be registered")
	}
	for _, id := range model.screenOrder {
		if id == "mail-admin" {
			t.Fatalf("mail-admin should be hidden from primary sidebar: %#v", model.screenOrder)
		}
	}
	for _, id := range []string{"go-mail-admin", "mail-admin-refresh", "domains-new", "domains-validate", "inboxes-new", "inboxes-pipeline"} {
		if _, ok := model.commands.Find(id); !ok {
			t.Fatalf("expected command %q", id)
		}
	}
}

func TestTabCyclesFocusOutsideConversationView(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model = sendKey(t, model, tea.Key{Code: tea.KeyTab})
	if model.focus != FocusSidebar {
		t.Fatalf("focus = %v, want sidebar", model.focus)
	}
	model = sendKey(t, model, tea.Key{Code: tea.KeyTab})
	if model.focus != FocusMain {
		t.Fatalf("focus = %v, want main", model.focus)
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
