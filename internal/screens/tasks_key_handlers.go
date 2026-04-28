package screens

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (t Tasks) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if t.confirm != "" {
		return t.handleConfirmKey(msg)
	}
	if t.filtering {
		return t.handleFilterKey(msg)
	}
	if t.picking {
		return t.handlePickerKey(msg)
	}
	if t.detail != nil {
		if key.Matches(msg, t.keys.Back) {
			t.detail = nil
			t.detailScroll = 0
			return t, nil
		}
		if key.Matches(msg, t.keys.Edit) {
			return t, t.editCachedCardCmd(*t.detail)
		}
		if key.Matches(msg, t.keys.Up) && t.detailScroll > 0 {
			t.detailScroll--
		} else if key.Matches(msg, t.keys.Down) {
			t.detailScroll++
		}
		return t, nil
	}
	rows := t.visibleRows()
	if key.Matches(msg, t.keys.Open) {
		if row, ok := t.selectedRow(); ok {
			if row.Project != nil {
				return t, t.loadCmd(row.Project.Meta.RemoteID)
			}
			if row.Card != nil {
				card := *row.Card
				t.detail = &card
				t.detailScroll = 0
			}
		}
		return t, nil
	}
	switch {
	case key.Matches(msg, t.keys.Back), key.Matches(msg, t.keys.Project):
		return t.handleAction("projects")
	case key.Matches(msg, t.keys.Refresh):
		return t, t.loadCmd(t.currentProjectID())
	case key.Matches(msg, t.keys.Sync):
		return t.handleAction("sync")
	case key.Matches(msg, t.keys.Search):
		return t.handleAction("search")
	case key.Matches(msg, t.keys.New):
		if t.project == nil {
			return t.handleAction("new-project")
		}
		return t.handleAction("new-card")
	case key.Matches(msg, t.keys.Edit):
		return t.handleAction("edit-card")
	case key.Matches(msg, t.keys.Delete):
		return t.handleAction("delete-card")
	case key.Matches(msg, t.keys.MoveNext):
		return t.handleAction("move-card-next")
	case key.Matches(msg, t.keys.MovePrev):
		return t.handleAction("move-card-prev")
	case key.Matches(msg, t.keys.MoveTo):
		return t.handleAction("move-card-to")
	default:
		t.ensureList(rows)
		updated, cmd := t.rowList.Update(msg)
		t.rowList = updated
		t.index = t.rowList.GlobalIndex()
		t.clampIndex()
		return t, cmd
	}
}

func (t Tasks) handleFilterKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		t.filtering = false
		t.filter = ""
		t.index = 0
	case "enter":
		t.filtering = false
	case "backspace":
		if len(t.filter) > 0 {
			t.filter = t.filter[:len(t.filter)-1]
		}
		t.index = 0
	default:
		if msg.Text != "" {
			t.filter += msg.Text
			t.index = 0
		}
	}
	t.syncList()
	return t, nil
}

func (t Tasks) handleConfirmKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		t.confirm = ""
		return t, t.deleteCardCmd()
	case "n", "N", "esc":
		t.confirm = ""
		t.status = "Cancelled"
	}
	return t, nil
}

func (t Tasks) handlePickerKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		t.picking = false
		t.picker = ""
		return t, nil
	case "enter":
		picked := t.picker
		target := t.resolvePickerColumn()
		t.picking = false
		t.picker = ""
		if target == "" {
			t.status = fmt.Sprintf("Unknown column: %q", picked)
			return t, nil
		}
		row, ok := t.selectedRow()
		if !ok || row.Card == nil {
			return t, nil
		}
		return t, t.moveCardCmd(*row.Card, target)
	case "backspace":
		if len(t.picker) > 0 {
			t.picker = t.picker[:len(t.picker)-1]
		}
		return t, nil
	default:
		if msg.Text != "" {
			t.picker += msg.Text
		}
		return t, nil
	}
}
