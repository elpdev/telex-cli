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
