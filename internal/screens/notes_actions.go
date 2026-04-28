package screens

import tea "charm.land/bubbletea/v2"

func (n Notes) Selection() NotesSelection {
	row, ok := n.selectedRow()
	if !ok || row.Note == nil {
		return NotesSelection{Kind: "note", HasItem: false}
	}
	return NotesSelection{Kind: "note", Subject: row.Note.Meta.Title, HasItem: true}
}

type NotesSelection struct {
	Kind    string
	Subject string
	HasItem bool
}

func (n Notes) handleAction(action string) (Screen, tea.Cmd) {
	if n.confirm != "" || n.editing {
		return n, nil
	}
	switch action {
	case "sync":
		if n.sync == nil || n.syncing {
			return n, nil
		}
		n.syncing = true
		n.status = ""
		return n, n.syncCmd()
	case "new":
		return n, n.createCmd()
	case "edit":
		return n, n.editCmd()
	case "delete":
		if row, ok := n.selectedRow(); ok && row.Note != nil {
			n.confirm = "Delete " + row.Note.Meta.Title + "?"
		}
	case "search":
		n.editing = true
		n.filter = ""
		n.index = 0
		n.syncNoteList()
	case "toggle-sort":
		n.sortMode = nextSortMode(n.sortMode)
		n.rows = n.buildRows()
		n.index = 0
		n.syncNoteList()
		n.status = "Sort: " + sortModeLabel(n.sortMode)
	case "toggle-flat":
		n.flat = !n.flat
		n.rows = n.buildRows()
		n.index = 0
		n.syncNoteList()
		if n.flat {
			n.status = "Flat view: all notes"
		} else {
			n.status = "Folder view"
		}
	}
	return n, nil
}

func nextSortMode(mode string) string {
	if mode == "recent" {
		return "name"
	}
	return "recent"
}

func sortModeLabel(mode string) string {
	if mode == "recent" {
		return "Recent"
	}
	return "A-Z"
}

func (n Notes) handleConfirmKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		n.confirm = ""
		return n, n.deleteCmd()
	case "n", "N", "esc":
		n.confirm = ""
		n.status = "Cancelled"
	}
	return n, nil
}
