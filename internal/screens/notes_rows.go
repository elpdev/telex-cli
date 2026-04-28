package screens

import (
	"io"
	"sort"
	"strings"

	"charm.land/bubbles/v2/list"
	"github.com/elpdev/telex-cli/internal/notes"
	"github.com/elpdev/telex-cli/internal/notestore"
)

func (n Notes) buildRows() []noteRow {
	if n.folder == nil {
		return nil
	}
	if n.flat {
		return n.buildFlatRows()
	}
	folders := make([]*notes.FolderTree, 0, len(n.folder.Children))
	for i := range n.folder.Children {
		folders = append(folders, &n.folder.Children[i])
	}
	sort.SliceStable(folders, func(i, j int) bool {
		return strings.ToLower(folders[i].Name) < strings.ToLower(folders[j].Name)
	})
	rows := make([]noteRow, 0, len(folders)+len(n.notes))
	for _, folder := range folders {
		rows = append(rows, noteRow{Kind: "folder", Name: folder.Name, Folder: folder})
	}
	noteRows := make([]noteRow, 0, len(n.notes))
	for i := range n.notes {
		note := &n.notes[i]
		noteRows = append(noteRows, noteRow{Kind: "note", Name: note.Meta.Title, Note: note})
	}
	sortNoteRows(noteRows, n.sortMode)
	rows = append(rows, noteRows...)
	return rows
}

func (n Notes) buildFlatRows() []noteRow {
	if n.tree == nil {
		return nil
	}
	cached, err := n.store.AllNotes()
	if err != nil || len(cached) == 0 {
		return nil
	}
	rows := make([]noteRow, 0, len(cached))
	for i := range cached {
		note := &cached[i]
		rows = append(rows, noteRow{Kind: "note", Name: flatNoteName(n.tree, note), Note: note})
	}
	sortNoteRows(rows, n.sortMode)
	return rows
}

func flatNoteName(tree *notes.FolderTree, note *notestore.CachedNote) string {
	if tree == nil || note.Meta.FolderID == 0 || note.Meta.FolderID == tree.ID {
		return note.Meta.Title
	}
	paths := notesFolderPath(tree, note.Meta.FolderID, nil)
	if len(paths) <= 1 {
		return note.Meta.Title
	}
	return strings.Join(paths[1:], " / ") + " / " + note.Meta.Title
}

func sortNoteRows(rows []noteRow, mode string) {
	switch mode {
	case "recent":
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].Note == nil || rows[j].Note == nil {
				return rows[i].Note != nil
			}
			return rows[i].Note.Meta.RemoteUpdatedAt.After(rows[j].Note.Meta.RemoteUpdatedAt)
		})
	default:
		sort.SliceStable(rows, func(i, j int) bool {
			return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
		})
	}
}

func (n Notes) visibleRows() []noteRow {
	filter := strings.ToLower(strings.TrimSpace(n.filter))
	if filter == "" {
		return n.rows
	}
	out := make([]noteRow, 0, len(n.rows))
	for _, row := range n.rows {
		if strings.Contains(strings.ToLower(row.Name), filter) {
			out = append(out, row)
			continue
		}
		if row.Note != nil && strings.Contains(strings.ToLower(row.Note.Body), filter) {
			out = append(out, row)
		}
	}
	return out
}

func (n *Notes) clampIndex() {
	if n.index >= len(n.visibleRows()) {
		n.index = maxNotesIndex(len(n.visibleRows()))
	}
	if n.index < 0 {
		n.index = 0
	}
}

func (n *Notes) ensureNoteList(rows []noteRow) {
	if len(n.rowList.Items()) == len(rows) {
		n.rowList.Select(n.clampedRowIndex(rows))
		return
	}
	n.syncNoteList()
}

func (n *Notes) syncNoteList() {
	rows := n.visibleRows()
	n.index = n.clampedRowIndex(rows)
	n.rowList = newNotesList(rows, n.index, n.rowList.Width(), n.rowList.Height())
}

func (n Notes) clampedRowIndex(rows []noteRow) int {
	if n.index < 0 || len(rows) == 0 {
		return 0
	}
	if n.index >= len(rows) {
		return len(rows) - 1
	}
	return n.index
}

func newNotesList(rows []noteRow, selected, width, height int) list.Model {
	items := make([]list.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, noteListItem{row: row})
	}
	return newSimpleList(items, noteListDelegate{}, selected, width, height)
}

type noteListDelegate struct{ simpleDelegate }

func (d noteListDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	noteItem, ok := item.(noteListItem)
	if !ok {
		return
	}
	_, _ = io.WriteString(w, formatNotesRow(noteItem.row, index == m.Index(), m.Width()))
}

func (n Notes) selectedRow() (noteRow, bool) {
	rows := n.visibleRows()
	if len(rows) == 0 || n.index < 0 || n.index >= len(rows) {
		return noteRow{}, false
	}
	return rows[n.index], true
}

func (n Notes) currentFolderID() int64 {
	if n.folder == nil {
		return 0
	}
	return n.folder.ID
}

func (n Notes) breadcrumb() string {
	if n.tree == nil || n.folder == nil {
		return ""
	}
	if n.folder.ID == n.tree.ID {
		return ""
	}
	paths := notesFolderPath(n.tree, n.folder.ID, nil)
	if len(paths) == 0 {
		return n.folder.Name
	}
	if len(paths) > 1 {
		paths = paths[1:]
	}
	return strings.Join(paths, " / ")
}

func findNotesFolder(tree *notes.FolderTree, id int64) *notes.FolderTree {
	if tree == nil {
		return nil
	}
	if id == 0 || tree.ID == id {
		return tree
	}
	for i := range tree.Children {
		if found := findNotesFolder(&tree.Children[i], id); found != nil {
			return found
		}
	}
	return nil
}

func notesFolderPath(tree *notes.FolderTree, id int64, path []string) []string {
	if tree == nil {
		return nil
	}
	path = append(path, tree.Name)
	if tree.ID == id {
		return path
	}
	for i := range tree.Children {
		if found := notesFolderPath(&tree.Children[i], id, append([]string{}, path...)); len(found) > 0 {
			return found
		}
	}
	return nil
}

func maxNotesIndex(length int) int {
	if length <= 0 {
		return 0
	}
	return length - 1
}
