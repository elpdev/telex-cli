package screens

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/tasks"
	"github.com/elpdev/telex-cli/internal/taskstore"
)

func TestTasksScreenLoadsCachedBoardAndOpensCard(t *testing.T) {
	store := testTasksStore(t)
	screen := NewTasks(store, nil)
	updated, _ := screen.Update(screen.load(4))
	screen = updated.(Tasks)
	view := screen.View(80, 20)
	if !strings.Contains(view, "Todo") || !strings.Contains(view, "Homepage") {
		t.Fatalf("view = %q", view)
	}
	screen.index = 1
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Tasks)
	if !strings.Contains(screen.View(80, 20), "Homepage") || !strings.Contains(screen.View(80, 20), "[esc] back") {
		t.Fatalf("detail view = %q", screen.View(80, 20))
	}
}

func TestTasksScreenShowsUnlinkedCards(t *testing.T) {
	store := testTasksStore(t)
	syncedAt := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	if err := store.StoreCard(4, tasks.Card{TaskFile: tasks.TaskFile{ID: 10, FolderID: 7, Title: "Unplanned", Filename: "Unplanned.md", UpdatedAt: syncedAt}, Body: "# Unplanned"}, syncedAt); err != nil {
		t.Fatal(err)
	}
	screen := NewTasks(store, nil)
	updated, _ := screen.Update(screen.load(4))
	screen = updated.(Tasks)
	view := screen.View(80, 20)
	if !strings.Contains(view, "Unlinked") || !strings.Contains(view, "Unplanned") {
		t.Fatalf("view = %q", view)
	}
}

func TestTasksScreenMoveCardNextInvokesMoveCard(t *testing.T) {
	store := testTasksStore(t)
	syncedAt := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	if err := store.StoreBoard(4, tasks.Board{TaskFile: tasks.TaskFile{ID: 5, Title: "Board", Filename: "board.md", UpdatedAt: syncedAt}, Body: "# Website\n\n## Todo\n- [[cards/Homepage.md]]\n\n## Doing\n\n## Done\n"}, syncedAt); err != nil {
		t.Fatal(err)
	}
	var calledFilename, calledColumn string
	move := func(_ context.Context, _ int64, filename, column string) error {
		calledFilename = filename
		calledColumn = column
		return nil
	}
	screen := NewTasks(store, nil).WithActions(nil, nil, nil, nil, move)
	updated, _ := screen.Update(screen.load(4))
	screen = updated.(Tasks)
	screen.index = 1
	updated2, cmd := screen.Update(TasksActionMsg{Action: "move-card-next"})
	screen = updated2.(Tasks)
	if cmd == nil {
		t.Fatalf("expected a command, got nil; status=%q", screen.status)
	}
	cmd()
	if calledFilename != "Homepage.md" || calledColumn != "Doing" {
		t.Fatalf("move called with filename=%q column=%q", calledFilename, calledColumn)
	}
}

func TestTasksScreenMoveCardToPicker(t *testing.T) {
	store := testTasksStore(t)
	syncedAt := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	if err := store.StoreBoard(4, tasks.Board{TaskFile: tasks.TaskFile{ID: 5, Title: "Board", Filename: "board.md", UpdatedAt: syncedAt}, Body: "# Website\n\n## Todo\n- [[cards/Homepage.md]]\n\n## Doing\n\n## Done\n"}, syncedAt); err != nil {
		t.Fatal(err)
	}
	var calledColumn string
	move := func(_ context.Context, _ int64, _, column string) error {
		calledColumn = column
		return nil
	}
	screen := NewTasks(store, nil).WithActions(nil, nil, nil, nil, move)
	updated, _ := screen.Update(screen.load(4))
	screen = updated.(Tasks)
	screen.index = 1
	updated, _ = screen.Update(TasksActionMsg{Action: "move-card-to"})
	screen = updated.(Tasks)
	if !screen.picking {
		t.Fatalf("expected picker open")
	}
	if !strings.Contains(screen.View(80, 20), "Move to column") {
		t.Fatalf("picker prompt missing from view")
	}
	for _, ch := range "doi" {
		updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: rune(ch), Text: string(ch)}))
		screen = updated.(Tasks)
	}
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Tasks)
	if cmd == nil {
		t.Fatalf("enter did not trigger move command; status=%q", screen.status)
	}
	cmd()
	if calledColumn != "Doing" {
		t.Fatalf("expected 'Doing', got %q", calledColumn)
	}
}

func TestTasksScreenHeaderShowsProjectAndHints(t *testing.T) {
	store := testTasksStore(t)
	screen := NewTasks(store, nil)
	updated, _ := screen.Update(screen.load(4))
	screen = updated.(Tasks)
	view := screen.View(80, 20)
	if !strings.Contains(view, "Website") {
		t.Fatalf("expected project name 'Website' in header, got %q", view)
	}
	if !strings.Contains(view, "esc/p") {
		t.Fatalf("expected back hint, got %q", view)
	}
}

func TestTasksEditorCommandUsesNotesOverride(t *testing.T) {
	t.Setenv("TELEX_NOTES_EDITOR", "typora")
	t.Setenv("VISUAL", "code")
	t.Setenv("EDITOR", "nvim")
	if got := tasksEditorCommand(); got != "typora" {
		t.Fatalf("editor = %q", got)
	}
}

func TestParseTaskCardTemplateFrontmatter(t *testing.T) {
	document := "---\ntitle: Homepage\ncolumn: Doing\n---\n\n# Body"
	input := parseTaskCardTemplate(document)
	if input.Card.Title != "Homepage" || input.Card.Body != document || input.Column != "Doing" {
		t.Fatalf("input = %#v", input)
	}
}

func TestParseTaskCardTemplateLegacy(t *testing.T) {
	document := "Title: Homepage\n\n# Body"
	input := parseTaskCardTemplate(document)
	if input.Card.Title != "Homepage" || input.Card.Body != document || input.Column != "" {
		t.Fatalf("input = %#v", input)
	}
}

func TestParseTaskProjectDocumentFrontmatter(t *testing.T) {
	document := "---\nname: Website\n---\n"
	input := parseTaskProjectDocument(document)
	if input.Name != "Website" || input.Body != document {
		t.Fatalf("input = %#v", input)
	}
}

func testTasksStore(t *testing.T) taskstore.Store {
	t.Helper()
	store := taskstore.New(t.TempDir())
	syncedAt := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	project := tasks.Project{ProjectSummary: tasks.ProjectSummary{ID: 4, Name: "Website", UpdatedAt: syncedAt}}
	if err := store.StoreProject(project, syncedAt); err != nil {
		t.Fatal(err)
	}
	if err := store.StoreCard(4, tasks.Card{TaskFile: tasks.TaskFile{ID: 9, FolderID: 7, Title: "Homepage", Filename: "Homepage.md", UpdatedAt: syncedAt}, Body: "# Homepage"}, syncedAt); err != nil {
		t.Fatal(err)
	}
	if err := store.StoreBoard(4, tasks.Board{TaskFile: tasks.TaskFile{ID: 5, Title: "Board", Filename: "board.md", UpdatedAt: syncedAt}, Body: "# Website\n\n## Todo\n- [[cards/Homepage.md]]\n"}, syncedAt); err != nil {
		t.Fatal(err)
	}
	return store
}
