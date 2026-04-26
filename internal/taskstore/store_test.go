package taskstore

import (
	"strings"
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/tasks"
)

func TestTaskStoreCachesProjectBoardAndCards(t *testing.T) {
	store := New(t.TempDir())
	syncedAt := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	project := tasks.Project{ProjectSummary: tasks.ProjectSummary{ID: 4, Name: "Website", UpdatedAt: syncedAt}}
	if err := store.StoreProject(project, syncedAt); err != nil {
		t.Fatal(err)
	}
	card := tasks.Card{TaskFile: tasks.TaskFile{ID: 9, FolderID: 7, Title: "Homepage", Filename: "Homepage.md", UpdatedAt: syncedAt}, Body: "# Homepage"}
	if err := store.StoreCard(4, card, syncedAt); err != nil {
		t.Fatal(err)
	}
	board := tasks.Board{TaskFile: tasks.TaskFile{ID: 5, Title: "Board", Filename: "board.md", UpdatedAt: syncedAt}, Body: "# Website\n\n## Todo\n- [[cards/Homepage.md]]\n"}
	if err := store.StoreBoard(4, board, syncedAt); err != nil {
		t.Fatal(err)
	}

	projects, err := store.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].Meta.Name != "Website" {
		t.Fatalf("projects = %#v", projects)
	}
	cards, err := store.ListCards(4)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || cards[0].Body != "# Homepage" {
		t.Fatalf("cards = %#v", cards)
	}
	cachedBoard, err := store.ReadBoard(4)
	if err != nil {
		t.Fatal(err)
	}
	if len(cachedBoard.Columns) != 1 || cachedBoard.Columns[0].Cards[0].Missing || cachedBoard.Columns[0].Cards[0].Title != "Homepage" {
		t.Fatalf("board columns = %#v body=%q", cachedBoard.Columns, cachedBoard.Body)
	}
	if !strings.Contains(cachedBoard.Path, "board.md") {
		t.Fatalf("board path = %q", cachedBoard.Path)
	}
}
