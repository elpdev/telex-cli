package screens

import (
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

func TestTasksEditorCommandUsesNotesOverride(t *testing.T) {
	t.Setenv("TELEX_NOTES_EDITOR", "typora")
	t.Setenv("VISUAL", "code")
	t.Setenv("EDITOR", "nvim")
	if got := tasksEditorCommand(); got != "typora" {
		t.Fatalf("editor = %q", got)
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
