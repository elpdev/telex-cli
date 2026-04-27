package tasks

import (
	"strings"
	"testing"
)

func TestAddCardToColumnAddsCardUnderTodo(t *testing.T) {
	updated := AddCardToColumn("# Project\n\n## Todo\n\n## Doing\n", "Todo", "cards/Homepage.md")
	if !strings.Contains(updated, "- [[cards/Homepage.md]]") || !strings.Contains(updated, "## Todo") {
		t.Fatalf("updated = %q", updated)
	}
}

func TestAddCardToColumnAvoidsDuplicate(t *testing.T) {
	board := "# Project\n\n## Todo\n- [[cards/Homepage.md]]\n"
	if got := AddCardToColumn(board, "Todo", "cards/Homepage.md"); got != board {
		t.Fatalf("updated = %q", got)
	}
}

func TestReplaceCardPathUpdatesBoardLink(t *testing.T) {
	board := "# Project\n\n## Todo\n- [[cards/Old.md]]\n"
	updated := ReplaceCardPath(board, "cards/Old.md", "cards/New.md")
	if strings.Contains(updated, "cards/Old.md") || !strings.Contains(updated, "cards/New.md") {
		t.Fatalf("updated = %q", updated)
	}
}

func TestMoveCardToColumnMovesBetweenColumns(t *testing.T) {
	board := "# Project\n\n## Todo\n- [[cards/Homepage.md]]\n\n## Doing\n\n## Done\n"
	updated := MoveCardToColumn(board, "cards/Homepage.md", "Doing")
	todoIdx := strings.Index(updated, "## Todo")
	doingIdx := strings.Index(updated, "## Doing")
	doneIdx := strings.Index(updated, "## Done")
	cardIdx := strings.Index(updated, "- [[cards/Homepage.md]]")
	if todoIdx < 0 || doingIdx < 0 || doneIdx < 0 || cardIdx < 0 {
		t.Fatalf("missing section in %q", updated)
	}
	if cardIdx < doingIdx || cardIdx > doneIdx {
		t.Fatalf("card not under Doing: %q", updated)
	}
	if strings.Count(updated, "- [[cards/Homepage.md]]") != 1 {
		t.Fatalf("expected one link, got %q", updated)
	}
}

func TestMoveCardToColumnNoOpWhenAlreadyThere(t *testing.T) {
	board := "# Project\n\n## Todo\n- [[cards/Homepage.md]]\n\n## Doing\n"
	if got := MoveCardToColumn(board, "cards/Homepage.md", "Todo"); got != board {
		t.Fatalf("updated = %q", got)
	}
}

func TestMoveCardToColumnCreatesMissingColumn(t *testing.T) {
	board := "# Project\n\n## Todo\n- [[cards/Homepage.md]]\n"
	updated := MoveCardToColumn(board, "cards/Homepage.md", "Doing")
	if !strings.Contains(updated, "## Doing") || !strings.Contains(updated, "- [[cards/Homepage.md]]") {
		t.Fatalf("updated = %q", updated)
	}
	if strings.Count(updated, "- [[cards/Homepage.md]]") != 1 {
		t.Fatalf("expected one link, got %q", updated)
	}
}

func TestMoveCardToColumnFromUnlinked(t *testing.T) {
	board := "# Project\n\n## Todo\n\n## Doing\n"
	updated := MoveCardToColumn(board, "cards/Homepage.md", "Doing")
	doingIdx := strings.Index(updated, "## Doing")
	cardIdx := strings.Index(updated, "- [[cards/Homepage.md]]")
	if doingIdx < 0 || cardIdx < 0 || cardIdx < doingIdx {
		t.Fatalf("card not placed under Doing: %q", updated)
	}
}

func TestRemoveCardFromColumnsStripsAllInstances(t *testing.T) {
	board := "# Project\n\n## Todo\n- [[cards/X.md]]\n\n## Doing\n- [[cards/X.md]]\n"
	updated := RemoveCardFromColumns(board, "cards/X.md")
	if strings.Contains(updated, "cards/X.md") {
		t.Fatalf("expected card removed, got %q", updated)
	}
	if !strings.Contains(updated, "## Todo") || !strings.Contains(updated, "## Doing") {
		t.Fatalf("columns missing: %q", updated)
	}
}
