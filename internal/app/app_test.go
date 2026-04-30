package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/commands"
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

func TestBackgroundMailSyncTickSchedulesSyncAndNextTick(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	updated, cmd := model.Update(mailAutoSyncTickMsg{})
	model = updated.(Model)

	if cmd == nil {
		t.Fatal("expected mail auto sync tick to schedule commands")
	}
	if model.CurrentScreenID() != "home" {
		t.Fatalf("expected tick to leave active screen unchanged, got %q", model.CurrentScreenID())
	}
}

func TestBackgroundMailSyncReloadsActiveMailScreen(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model = model.SwitchScreenForTest("mail-unread")
	updated, cmd := model.Update(backgroundMailSyncedMsg{source: "boot"})
	model = updated.(Model)

	if cmd == nil {
		t.Fatal("expected background mail sync to reload active mail screen")
	}
	if model.CurrentScreenID() != "mail-unread" {
		t.Fatalf("expected active mail screen to remain selected, got %q", model.CurrentScreenID())
	}
}

func TestSkippedBackgroundMailSyncDoesNotReload(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model = model.SwitchScreenForTest("mail-unread")
	_, cmd := model.Update(backgroundMailSyncedMsg{source: "timer", skipped: true, err: errMailSyncAlreadyRunning})

	if cmd != nil {
		t.Fatal("expected skipped background mail sync not to reload")
	}
}

func TestGlobalMailSidebarEntryOpensUnread(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model = sendKey(t, model, tea.Key{Code: tea.KeyTab})
	model = sendKey(t, model, tea.Key{Code: tea.KeyDown})
	if model.CurrentScreenID() != "home" {
		t.Fatalf("down should only move sidebar cursor, got %q", model.CurrentScreenID())
	}
	model = sendKey(t, model, tea.Key{Code: tea.KeyEnter})

	if model.CurrentScreenID() != "mail-unread" {
		t.Fatalf("expected mail sidebar entry to open unread, got %q", model.CurrentScreenID())
	}
}

func TestMailCommandsOpenUnreadAndMailboxes(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	cmd, ok := model.commands.Find("go-mail")
	if !ok {
		t.Fatal("expected go-mail command")
	}
	updated, _ := model.Update(cmd.Run()())
	model = updated.(Model)
	if model.CurrentScreenID() != "mail-unread" {
		t.Fatalf("go-mail opened %q, want mail-unread", model.CurrentScreenID())
	}

	cmd, ok = model.commands.Find("go-mailboxes")
	if !ok {
		t.Fatal("expected go-mailboxes command")
	}
	updated, _ = model.Update(cmd.Run()())
	model = updated.(Model)
	if model.CurrentScreenID() != "mail" {
		t.Fatalf("go-mailboxes opened %q, want mail", model.CurrentScreenID())
	}
}

func TestNotesScreenRegisteredInNavigationAndCommands(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	if _, ok := model.screens["notes"]; !ok {
		t.Fatal("expected notes screen to be registered")
	}
	if got := model.screenOrder; len(got) < 6 || got[4] != "notes" {
		t.Fatalf("screenOrder = %#v", got)
	}
	for _, id := range []string{"go-notes", "notes-sync", "notes-new", "notes-edit", "notes-delete", "notes-search"} {
		if _, ok := model.commands.Find(id); !ok {
			t.Fatalf("expected command %q", id)
		}
	}
}

func TestTasksScreenRegisteredInNavigationAndCommands(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	if _, ok := model.screens["tasks"]; !ok {
		t.Fatal("expected tasks screen to be registered")
	}
	if got := model.screenOrder; len(got) < 7 || got[5] != "tasks" {
		t.Fatalf("screenOrder = %#v", got)
	}
	for _, id := range []string{"go-tasks", "tasks-sync", "tasks-projects", "tasks-new-project", "tasks-new-card", "tasks-edit-card", "tasks-delete-card", "tasks-search"} {
		if _, ok := model.commands.Find(id); !ok {
			t.Fatalf("expected command %q", id)
		}
	}
}

func TestContactsScreenRegisteredInNavigationAndCommands(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	if _, ok := model.screens["contacts"]; !ok {
		t.Fatal("expected contacts screen to be registered")
	}
	if got := model.screenOrder; len(got) < 5 || got[3] != "contacts" {
		t.Fatalf("screenOrder = %#v", got)
	}
	for _, id := range []string{"go-contacts", "contacts-sync", "contacts-search", "contacts-delete", "contacts-edit-note", "contacts-refresh-note", "contacts-communications"} {
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

func TestCalendarPaletteCommandAvailability(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})

	agenda := commands.Context{ActiveScreen: "calendar", Selection: &commands.Selection{Kind: "calendar-event", Subject: "Planning", HasItems: true}}
	assertCommandIDs(t, model.commands.Filter("calendar ", agenda), []string{"calendar-new", "calendar-edit", "calendar-previous-range", "calendar-next-range", "calendar-filter-agenda", "calendar-delete"}, true)
	assertCommandIDs(t, model.commands.Filter("calendar ", agenda), []string{"calendar-invitation-accept", "calendars-import-ics", "calendars-edit"}, false)

	invite := commands.Context{ActiveScreen: "calendar", Selection: &commands.Selection{Kind: "calendar-event", Subject: "Planning", HasItems: true, HasInvitation: true}}
	assertCommandIDs(t, model.commands.Filter("calendar invitation", invite), []string{"calendar-invitation-show", "calendar-invitation-sync", "calendar-invitation-accept", "calendar-invitation-tentative", "calendar-invitation-decline", "calendar-invitation-needs-action"}, true)

	calendarManagement := commands.Context{ActiveScreen: "calendar", Selection: &commands.Selection{Kind: "calendar", Subject: "Work", HasItems: true}}
	assertCommandIDs(t, model.commands.Filter("calendar ", calendarManagement), []string{"calendar-view-agenda", "calendars-new", "calendars-edit", "calendars-import-ics", "calendars-delete"}, true)
	assertCommandIDs(t, model.commands.Filter("calendar ", calendarManagement), []string{"calendar-new", "calendar-edit", "calendar-previous-range", "calendar-invitation-accept"}, false)

	otherScreen := commands.Context{ActiveScreen: "home"}
	assertCommandIDs(t, model.commands.Filter("calendar ", otherScreen), []string{"go-calendar", "calendar-sync"}, true)
	assertCommandIDs(t, model.commands.Filter("calendar ", otherScreen), []string{"calendar-new", "calendar-edit", "calendars-new", "calendars-import-ics", "calendar-invitation-accept"}, false)
}

func TestCalendarPaletteDescriptionsIncludeSelectionSubject(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	ctx := commands.Context{ActiveScreen: "calendar", Selection: &commands.Selection{Kind: "calendar-event", Subject: "Planning", HasItems: true, HasInvitation: true}}

	for _, id := range []string{"calendar-edit", "calendar-delete", "calendar-invitation-accept"} {
		cmd, ok := model.commands.Find(id)
		if !ok {
			t.Fatalf("expected command %q", id)
		}
		if got := cmd.DescriptionFor(ctx); !strings.Contains(got, "Planning") {
			t.Fatalf("%s description = %q, want selected subject", id, got)
		}
	}

	calendarCtx := commands.Context{ActiveScreen: "calendar", Selection: &commands.Selection{Kind: "calendar", Subject: "Work", HasItems: true}}
	cmd, ok := model.commands.Find("calendars-import-ics")
	if !ok {
		t.Fatal("expected calendars-import-ics command")
	}
	if got := cmd.DescriptionFor(calendarCtx); !strings.Contains(got, "Work") {
		t.Fatalf("calendars-import-ics description = %q, want selected calendar", got)
	}
}

func assertCommandIDs(t *testing.T, list []commands.Command, ids []string, want bool) {
	t.Helper()
	found := make(map[string]bool, len(list))
	for _, cmd := range list {
		found[cmd.ID] = true
	}
	for _, id := range ids {
		if found[id] != want {
			t.Fatalf("command %q present = %v, want %v in %#v", id, found[id], want, commandIDs(list))
		}
	}
}

func commandIDs(list []commands.Command) []string {
	ids := make([]string, 0, len(list))
	for _, cmd := range list {
		ids = append(ids, cmd.ID)
	}
	return ids
}

func assertContainsScreenID(t *testing.T, ids []string, want string) {
	t.Helper()
	for _, id := range ids {
		if id == want {
			return
		}
	}
	t.Fatalf("screen id %q not found in %#v", want, ids)
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

func TestAggregateMailScreensRegisteredWithMailSidebar(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	for _, id := range []string{"mail-unread", "mail-inbox", "mail-sent", "mail-drafts", "mail-outbox", "mail-junk", "mail-archive", "mail-trash"} {
		if _, ok := model.screens[id]; !ok {
			t.Fatalf("expected %s screen", id)
		}
		if _, ok := model.commands.Find("go-" + id); !ok {
			t.Fatalf("expected go-%s command", id)
		}
		for _, navID := range model.screenOrder {
			if navID == id {
				t.Fatalf("%s should be hidden from global sidebar: %#v", id, model.screenOrder)
			}
		}
	}

	model = model.SwitchScreenForTest("mail-unread")
	ids := model.sidebarScreenIDs()
	assertContainsScreenID(t, ids, "home")
	assertContainsScreenID(t, ids, "mail-unread")
	assertContainsScreenID(t, ids, "mail-trash")
	assertContainsScreenID(t, ids, "mail-admin")
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
