package screens

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/taskstore"
)

func (t Tasks) Selection() TasksSelection {
	row, ok := t.selectedRow()
	if !ok || row.Card == nil {
		return TasksSelection{Kind: "task-card", HasItem: false}
	}
	return TasksSelection{Kind: "task-card", Subject: row.Card.Meta.Title, HasItem: true}
}

func (t Tasks) handleAction(action string) (Screen, tea.Cmd) {
	if t.confirm != "" || t.filtering || t.picking {
		return t, nil
	}
	switch action {
	case "sync":
		if t.sync == nil || t.syncing {
			return t, nil
		}
		t.syncing = true
		t.status = ""
		return t, t.syncCmd()
	case "new-card":
		return t, t.createCardCmd()
	case "new-project":
		return t, t.createProjectCmd()
	case "edit-card":
		return t, t.editCardCmd()
	case "delete-card":
		if row, ok := t.selectedRow(); ok && row.Card != nil {
			t.confirm = "Delete " + row.Card.Meta.Title + "?"
		}
	case "move-card-next":
		return t.moveCardNeighbor(1)
	case "move-card-prev":
		return t.moveCardNeighbor(-1)
	case "move-card-to":
		row, ok := t.selectedRow()
		if !ok || row.Card == nil || t.board == nil || len(t.board.Columns) == 0 {
			t.status = "Open a card in a project to move it"
			return t, nil
		}
		t.picking = true
		t.picker = ""
		return t, nil
	case "search":
		t.filtering = true
		t.filter = ""
		t.index = 0
		t.syncList()
	case "projects":
		t.project = nil
		t.board = nil
		t.cards = nil
		t.rows = t.buildRows()
		t.index = 0
		t.syncList()
	}
	return t, nil
}

func (t Tasks) moveCardNeighbor(delta int) (Screen, tea.Cmd) {
	row, ok := t.selectedRow()
	if !ok || row.Card == nil {
		t.status = "Select a card to move"
		return t, nil
	}
	if t.board == nil || len(t.board.Columns) == 0 {
		t.status = "No columns to move between"
		return t, nil
	}
	target := t.neighborColumn(row, delta)
	if target == "" {
		t.status = "Already at edge column"
		return t, nil
	}
	return t, t.moveCardCmd(*row.Card, target)
}

func (t Tasks) neighborColumn(row taskRow, delta int) string {
	cols := t.board.Columns
	current := -1
	if row.Column != nil {
		for i := range cols {
			if strings.EqualFold(cols[i].Name, row.Column.Name) {
				current = i
				break
			}
		}
	}
	if current == -1 {
		if delta > 0 {
			return cols[0].Name
		}
		return cols[len(cols)-1].Name
	}
	next := current + delta
	if next < 0 || next >= len(cols) {
		return ""
	}
	return cols[next].Name
}

func (t Tasks) resolvePickerColumn() string {
	needle := strings.ToLower(strings.TrimSpace(t.picker))
	if needle == "" || t.board == nil {
		return ""
	}
	for _, col := range t.board.Columns {
		if strings.EqualFold(col.Name, needle) {
			return col.Name
		}
	}
	match := ""
	count := 0
	for _, col := range t.board.Columns {
		if strings.HasPrefix(strings.ToLower(col.Name), needle) {
			match = col.Name
			count++
		}
	}
	if count == 1 {
		return match
	}
	return ""
}

func (t Tasks) moveCardCmd(card taskstore.CachedCard, target string) tea.Cmd {
	if t.moveCard == nil {
		t.status = "Move card is not configured"
		return nil
	}
	projectID := card.Meta.ProjectID
	return func() tea.Msg {
		if err := t.moveCard(context.Background(), projectID, card.Meta.Filename, target); err != nil {
			return taskActionFinishedMsg{err: err}
		}
		loaded := t.load(projectID)
		return taskActionFinishedMsg{status: "Moved " + card.Meta.Title + " to " + target, loaded: loaded, err: loaded.err}
	}
}
