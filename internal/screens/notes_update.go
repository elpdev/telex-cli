package screens

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
)

func (n Notes) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case notesLoadedMsg:
		n.loading = false
		n.err = msg.err
		if msg.err == nil {
			n.tree = msg.tree
			n.folder = msg.folder
			n.notes = msg.notes
			n.rows = n.buildRows()
			n.clampIndex()
			n.syncNoteList()
		}
		return n, nil
	case notesSyncedMsg:
		n.syncing = false
		n.err = msg.err
		if msg.err == nil {
			n.status = fmt.Sprintf("Synced %d folder(s), %d note(s)", msg.result.Folders, msg.result.Notes)
			n.tree = msg.loaded.tree
			n.folder = msg.loaded.folder
			n.notes = msg.loaded.notes
			n.rows = n.buildRows()
			n.clampIndex()
			n.syncNoteList()
		} else {
			n.status = ""
		}
		return n, nil
	case noteActionFinishedMsg:
		n.loading = false
		n.err = msg.err
		if msg.err != nil {
			n.status = fmt.Sprintf("Notes action failed: %v", msg.err)
			return n, nil
		}
		n.status = msg.status
		if msg.loaded.tree != nil {
			n.tree = msg.loaded.tree
			n.folder = msg.loaded.folder
			n.notes = msg.loaded.notes
			n.rows = n.buildRows()
			n.clampIndex()
			n.syncNoteList()
		}
		return n, nil
	case NotesActionMsg:
		return n.handleAction(msg.Action)
	case tea.KeyPressMsg:
		return n.handleKey(msg)
	}
	return n, nil
}
