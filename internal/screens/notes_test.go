package screens

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/notes"
	"github.com/elpdev/telex-cli/internal/notestore"
)

func TestNotesScreenLoadsCachedNotesAndOpensDetail(t *testing.T) {
	store := testNotesStore(t)
	screen := NewNotes(store, nil)
	updated, _ := screen.Update(screen.load(0))
	screen = updated.(Notes)
	view := screen.View(80, 20)
	if !strings.Contains(view, "Cached Note") {
		t.Fatalf("view = %q", view)
	}
	screen.index = 1
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Notes)
	view = screen.View(80, 20)
	plain := stripNotesANSI(view)
	if !strings.Contains(plain, "Cached Note") || !strings.Contains(plain, "Updated") || strings.Contains(plain, "ID: 9") || strings.Contains(plain, "# Cached") {
		t.Fatalf("detail view = %q", view)
	}
	if !strings.Contains(plain, "[esc] back") {
		t.Fatalf("detail view missing footer hint = %q", plain)
	}
}

var notesANSIRE = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func stripNotesANSI(value string) string {
	return notesANSIRE.ReplaceAllString(value, "")
}

func TestNotesScreenNavigatesFolderTree(t *testing.T) {
	store := testNotesStore(t)
	screen := NewNotes(store, nil)
	updated, _ := screen.Update(screen.load(0))
	screen = updated.(Notes)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Notes)
	updated, _ = screen.Update(cmd())
	screen = updated.(Notes)
	view := screen.View(80, 20)
	if !strings.Contains(view, "Project Note") || strings.Contains(view, "Cached Note") {
		t.Fatalf("folder view = %q", view)
	}
	updated, cmd = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	screen = updated.(Notes)
	updated, _ = screen.Update(cmd())
	screen = updated.(Notes)
	if !strings.Contains(screen.View(80, 20), "Cached Note") {
		t.Fatalf("root view = %q", screen.View(80, 20))
	}
}

func TestNotesScreenFilter(t *testing.T) {
	store := testNotesStore(t)
	screen := NewNotes(store, nil)
	updated, _ := screen.Update(screen.load(0))
	screen = updated.(Notes)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "/", Code: '/'}))
	screen = updated.(Notes)
	for _, r := range "cached" {
		updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: string(r), Code: r}))
		screen = updated.(Notes)
	}
	view := screen.View(80, 20)
	if !strings.Contains(view, "Cached Note") || strings.Contains(view, "Projects") {
		t.Fatalf("filtered view = %q", view)
	}
}

func TestNotesScreenUsesListNavigation(t *testing.T) {
	store := testNotesStore(t)
	screen := NewNotes(store, nil)
	updated, _ := screen.Update(screen.load(0))
	screen = updated.(Notes)

	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnd}))
	if cmd != nil {
		t.Fatal("expected no command")
	}
	screen = updated.(Notes)
	if screen.index != len(screen.visibleRows())-1 {
		t.Fatalf("index = %d, want last row", screen.index)
	}
	row, ok := screen.selectedRow()
	if !ok || row.Note == nil || row.Note.Meta.Title != "Cached Note" {
		t.Fatalf("selected row = %#v ok = %v", row, ok)
	}

	view := stripNotesANSI(screen.View(80, 20))
	if !strings.Contains(view, ">   Cached Note") {
		t.Fatalf("view missing selected cached note:\n%s", view)
	}
}

func TestNotesScreenSyncRefreshesList(t *testing.T) {
	store := notestore.New(t.TempDir())
	rootID := int64(1)
	if err := store.StoreTree(&notes.FolderTree{FolderSummary: notes.FolderSummary{ID: rootID, Name: "Notes"}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	screen := NewNotes(store, func(ctx context.Context) (NotesSyncResult, error) {
		if err := store.StoreNote(notes.Note{ID: 9, FolderID: &rootID, Title: "Synced Note", Body: "body"}, time.Now()); err != nil {
			t.Fatal(err)
		}
		return NotesSyncResult{Folders: 1, Notes: 1}, nil
	})
	updated, _ := screen.Update(screen.load(0))
	screen = updated.(Notes)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "S", Code: 'S'}))
	screen = updated.(Notes)
	updated, _ = screen.Update(cmd())
	screen = updated.(Notes)
	view := screen.View(80, 20)
	if !strings.Contains(view, "Synced 1 folder(s), 1 note(s)") || !strings.Contains(view, "Synced Note") {
		t.Fatalf("view = %q", view)
	}
}

func TestNotesScreenCreateAndEditInvokeCallbacks(t *testing.T) {
	t.Setenv("EDITOR", "false")
	t.Setenv("TELEX_NOTES_EDITOR", testEditorScript(t, "Title: Edited\n\nEdited body"))
	store := testNotesStore(t)
	rootID := int64(1)
	var created notes.NoteInput
	var updated notes.NoteInput
	screen := NewNotes(store, nil).WithActions(func(ctx context.Context, input notes.NoteInput) (*notes.Note, error) {
		created = input
		note := notes.Note{ID: 11, FolderID: &rootID, Title: input.Title, Body: input.Body}
		if err := store.StoreNote(note, time.Now()); err != nil {
			t.Fatal(err)
		}
		return &note, nil
	}, func(ctx context.Context, id int64, input notes.NoteInput) (*notes.Note, error) {
		updated = input
		note := notes.Note{ID: id, FolderID: input.FolderID, Title: input.Title, Body: input.Body}
		if err := store.StoreNote(note, time.Now()); err != nil {
			t.Fatal(err)
		}
		return &note, nil
	}, nil)
	loaded, _ := screen.Update(screen.load(0))
	screen = loaded.(Notes)
	loaded, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "n", Code: 'n'}))
	screen = loaded.(Notes)
	loaded, _ = screen.Update(cmd())
	screen = loaded.(Notes)
	if created.Title != "Edited" || created.Body != "Edited body" || created.FolderID == nil || *created.FolderID != rootID {
		t.Fatalf("created = %#v", created)
	}
	screen.index = 1
	loaded, cmd = screen.Update(tea.KeyPressMsg(tea.Key{Text: "e", Code: 'e'}))
	screen = loaded.(Notes)
	loaded, _ = screen.Update(cmd())
	if updated.Title != "Edited" || updated.Body != "Edited body" {
		t.Fatalf("updated = %#v", updated)
	}
}

func TestNotesScreenEditSkipsUnchangedTemplate(t *testing.T) {
	t.Setenv("TELEX_NOTES_EDITOR", "true")
	store := testNotesStore(t)
	updates := 0
	screen := NewNotes(store, nil).WithActions(nil, func(ctx context.Context, id int64, input notes.NoteInput) (*notes.Note, error) {
		updates++
		return &notes.Note{ID: id, FolderID: input.FolderID, Title: input.Title, Body: input.Body}, nil
	}, nil)
	loaded, _ := screen.Update(screen.load(0))
	screen = loaded.(Notes)
	screen.index = 1
	loaded, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "e", Code: 'e'}))
	screen = loaded.(Notes)
	loaded, _ = screen.Update(cmd())
	screen = loaded.(Notes)
	if updates != 0 {
		t.Fatalf("updates = %d, want 0", updates)
	}
	if screen.status != "No changes to save" {
		t.Fatalf("status = %q", screen.status)
	}
}

func TestNotesEditorCommandPrefersNotesOverride(t *testing.T) {
	t.Setenv("TELEX_NOTES_EDITOR", "typora")
	t.Setenv("VISUAL", "code")
	t.Setenv("EDITOR", "nvim")
	if got := notesEditorCommand(); got != "typora" {
		t.Fatalf("editor = %q", got)
	}
}

func TestNotesScreenDeleteRequiresConfirmation(t *testing.T) {
	store := testNotesStore(t)
	var deleted int64
	screen := NewNotes(store, nil).WithActions(nil, nil, func(ctx context.Context, id int64) error {
		deleted = id
		return store.DeleteNote(id)
	})
	updated, _ := screen.Update(screen.load(0))
	screen = updated.(Notes)
	screen.index = 1
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
	screen = updated.(Notes)
	if screen.confirm == "" {
		t.Fatal("expected confirmation")
	}
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "y", Code: 'y'}))
	screen = updated.(Notes)
	updated, _ = screen.Update(cmd())
	if deleted != 9 {
		t.Fatalf("deleted = %d", deleted)
	}
}

func TestNotesScreenSmallTerminalDoesNotPanic(t *testing.T) {
	store := testNotesStore(t)
	screen := NewNotes(store, nil)
	updated, _ := screen.Update(screen.load(0))
	screen = updated.(Notes)
	_ = screen.View(10, 3)
}

func TestParseNoteTemplate(t *testing.T) {
	input := parseNoteTemplate("Title: Plan\n\n# Body")
	if input.Title != "Plan" || input.Body != "# Body" {
		t.Fatalf("input = %#v", input)
	}
}

func testNotesStore(t *testing.T) notestore.Store {
	t.Helper()
	store := notestore.New(t.TempDir())
	syncedAt := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	rootID := int64(1)
	tree := &notes.FolderTree{FolderSummary: notes.FolderSummary{ID: rootID, Name: "Notes"}, Children: []notes.FolderTree{{FolderSummary: notes.FolderSummary{ID: 2, ParentID: &rootID, Name: "Projects"}, NoteCount: 1}}}
	if err := store.StoreTree(tree, syncedAt); err != nil {
		t.Fatal(err)
	}
	if err := store.StoreNote(notes.Note{ID: 9, FolderID: &rootID, Title: "Cached Note", Filename: "cached.md", Body: "# Cached", UpdatedAt: syncedAt}, syncedAt); err != nil {
		t.Fatal(err)
	}
	projectID := int64(2)
	if err := store.StoreNote(notes.Note{ID: 10, FolderID: &projectID, Title: "Project Note", Filename: "project.md", Body: "# Project", UpdatedAt: syncedAt}, syncedAt); err != nil {
		t.Fatal(err)
	}
	return store
}

func testEditorScript(t *testing.T, content string) string {
	t.Helper()
	path := t.TempDir() + "/editor.sh"
	script := "#!/bin/sh\nprintf '%s' '" + strings.ReplaceAll(content, "'", "'\\''") + "' > \"$1\"\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
