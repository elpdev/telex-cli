package screens

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (n Notes) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if n.confirm != "" {
		return n.handleConfirmKey(msg)
	}
	if n.editing {
		return n.handleFilterKey(msg)
	}
	if n.detail != nil {
		if key.Matches(msg, n.keys.Back) {
			n.detail = nil
			n.detailScroll = 0
			return n, nil
		}
		if key.Matches(msg, n.keys.Edit) {
			return n, n.editCachedCmd(*n.detail)
		}
		switch {
		case key.Matches(msg, n.keys.Up):
			if n.detailScroll > 0 {
				n.detailScroll--
			}
		case key.Matches(msg, n.keys.Down):
			n.detailScroll++
		case msg.String() == "pgup":
			n.detailScroll -= 10
			if n.detailScroll < 0 {
				n.detailScroll = 0
			}
		case msg.String() == "pgdown":
			n.detailScroll += 10
		}
		return n, nil
	}
	rows := n.visibleRows()
	if key.Matches(msg, n.keys.Open) {
		if len(rows) == 0 {
			return n, nil
		}
		row := rows[n.index]
		if row.Folder != nil {
			n.index = 0
			n.syncNoteList()
			return n, n.loadCmd(row.Folder.ID)
		}
		if row.Note != nil {
			note := *row.Note
			n.detail = &note
			n.detailScroll = 0
		}
		return n, nil
	}
	switch {
	case key.Matches(msg, n.keys.Back):
		if n.folder != nil && n.folder.ParentID != nil {
			n.index = 0
			n.syncNoteList()
			return n, n.loadCmd(*n.folder.ParentID)
		}
	case key.Matches(msg, n.keys.Refresh):
		return n, n.loadCmd(n.currentFolderID())
	case key.Matches(msg, n.keys.Sync):
		if n.sync == nil || n.syncing {
			return n, nil
		}
		n.syncing = true
		n.status = ""
		return n, n.syncCmd()
	case key.Matches(msg, n.keys.Search):
		n.editing = true
		n.filter = ""
		n.index = 0
		n.syncNoteList()
	case key.Matches(msg, n.keys.New):
		return n, n.createCmd()
	case key.Matches(msg, n.keys.Edit):
		return n, n.editCmd()
	case key.Matches(msg, n.keys.Delete):
		if row, ok := n.selectedRow(); ok && row.Note != nil {
			n.confirm = "Delete " + row.Note.Meta.Title + "?"
		}
	case key.Matches(msg, n.keys.Order):
		return n.handleAction("toggle-sort")
	case key.Matches(msg, n.keys.Flat):
		return n.handleAction("toggle-flat")
	default:
		n.ensureNoteList(rows)
		updated, cmd := n.rowList.Update(msg)
		n.rowList = updated
		n.index = n.rowList.GlobalIndex()
		n.clampIndex()
		return n, cmd
	}
	return n, nil
}

func (n Notes) handleFilterKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		n.editing = false
		n.filter = ""
		n.index = 0
	case "enter":
		n.editing = false
	case "backspace":
		if len(n.filter) > 0 {
			n.filter = n.filter[:len(n.filter)-1]
		}
		n.index = 0
	default:
		if msg.Text != "" {
			n.filter += msg.Text
			n.index = 0
		}
	}
	n.syncNoteList()
	return n, nil
}
