package screens

import (
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/tasks"
	"github.com/elpdev/telex-cli/internal/taskstore"
)

func (t Tasks) buildRows() []taskRow {
	if t.project == nil {
		rows := make([]taskRow, 0, len(t.projects))
		for i := range t.projects {
			project := &t.projects[i]
			rows = append(rows, taskRow{Kind: "project", Name: project.Meta.Name, Project: project})
		}
		return rows
	}
	byID := map[int64]*taskstore.CachedCard{}
	for i := range t.cards {
		byID[t.cards[i].Meta.RemoteID] = &t.cards[i]
	}
	linked := map[int64]bool{}
	rows := []taskRow{}
	if t.board != nil {
		for i := range t.board.Columns {
			column := &t.board.Columns[i]
			rows = append(rows, taskRow{Kind: "column", Name: column.Name, Column: column})
			for _, link := range column.Cards {
				if link.Card != nil {
					if card := byID[link.Card.ID]; card != nil {
						linked[card.Meta.RemoteID] = true
						rows = append(rows, taskRow{Kind: "card", Name: card.Meta.Title, Card: card, Column: column})
						continue
					}
				}
				rows = append(rows, taskRow{Kind: "missing", Name: link.Title, Missing: true, Column: column})
			}
		}
	}
	unlinked := t.unlinkedCards(linked)
	if len(unlinked) > 0 {
		rows = append(rows, taskRow{Kind: "column", Name: "Unlinked", Column: &tasks.BoardColumn{Name: "Unlinked"}})
		for i := range unlinked {
			card := &unlinked[i]
			rows = append(rows, taskRow{Kind: "card", Name: card.Meta.Title, Card: card})
		}
	}
	if len(rows) == 0 {
		for i := range t.cards {
			card := &t.cards[i]
			rows = append(rows, taskRow{Kind: "card", Name: card.Meta.Title, Card: card})
		}
	}
	return rows
}

func (t Tasks) unlinkedCards(linked map[int64]bool) []taskstore.CachedCard {
	out := []taskstore.CachedCard{}
	for _, card := range t.cards {
		if !linked[card.Meta.RemoteID] {
			out = append(out, card)
		}
	}
	return out
}

func (t Tasks) visibleRows() []taskRow {
	filter := strings.ToLower(strings.TrimSpace(t.filter))
	if filter == "" {
		return t.rows
	}
	out := make([]taskRow, 0, len(t.rows))
	for _, row := range t.rows {
		if strings.Contains(strings.ToLower(row.Name), filter) || row.Card != nil && strings.Contains(strings.ToLower(row.Card.Body), filter) {
			out = append(out, row)
		}
	}
	return out
}

func (t *Tasks) clampIndex() {
	if t.index >= len(t.visibleRows()) {
		t.index = maxNotesIndex(len(t.visibleRows()))
	}
	if t.index < 0 {
		t.index = 0
	}
}

func (t *Tasks) ensureList(rows []taskRow) {
	if len(t.rowList.Items()) == len(rows) {
		t.rowList.Select(t.clampedIndex(rows))
		return
	}
	t.syncList()
}

func (t *Tasks) syncList() {
	rows := t.visibleRows()
	t.index = t.clampedIndex(rows)
	t.rowList = newTaskList(rows, t.index, t.rowList.Width(), t.rowList.Height())
}

func (t Tasks) clampedIndex(rows []taskRow) int {
	if t.index < 0 || len(rows) == 0 {
		return 0
	}
	if t.index >= len(rows) {
		return len(rows) - 1
	}
	return t.index
}

func newTaskList(rows []taskRow, selected, width, height int) list.Model {
	items := make([]list.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, taskListItem{row: row})
	}
	return newSimpleList(items, taskListDelegate{}, selected, width, height)
}

func (d taskListDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	taskItem, ok := item.(taskListItem)
	if !ok {
		return
	}
	_, _ = io.WriteString(w, formatTaskRow(taskItem.row, index == m.Index(), m.Width()))
}

func formatTaskRow(row taskRow, selected bool, width int) string {
	cursor := listCursor(selected)
	glyph := "  "
	switch row.Kind {
	case "project":
		glyph = "▸ "
	case "column":
		glyph = "# "
	case "card":
		glyph = "- "
	case "missing":
		glyph = "! "
	}
	line := cursor + glyph + truncate(row.Name, max(0, width-4))
	if selected {
		line = lipgloss.NewStyle().Bold(true).Render(line)
	}
	return line
}

func (t Tasks) selectedRow() (taskRow, bool) {
	rows := t.visibleRows()
	if len(rows) == 0 || t.index < 0 || t.index >= len(rows) {
		return taskRow{}, false
	}
	return rows[t.index], true
}

func (t Tasks) currentProjectID() int64 {
	if t.project == nil {
		return 0
	}
	return t.project.Meta.RemoteID
}
