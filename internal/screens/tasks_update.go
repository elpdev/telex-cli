package screens

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

func (t Tasks) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tasksLoadedMsg:
		t.loading = false
		t.err = msg.err
		if msg.err == nil {
			t.projects = msg.projects
			t.project = msg.project
			t.board = msg.board
			t.cards = msg.cards
			t.rows = t.buildRows()
			t.clampIndex()
			t.syncList()
		}
		return t, nil
	case tasksSyncedMsg:
		t.syncing = false
		t.err = msg.err
		if msg.err == nil {
			t.status = fmt.Sprintf("Synced %d project(s), %d board(s), %d card(s)", msg.result.Projects, msg.result.Boards, msg.result.Cards)
			t.projects = msg.loaded.projects
			t.project = msg.loaded.project
			t.board = msg.loaded.board
			t.cards = msg.loaded.cards
			t.rows = t.buildRows()
			t.clampIndex()
			t.syncList()
		} else {
			t.status = ""
		}
		return t, nil
	case taskActionFinishedMsg:
		t.loading = false
		t.err = msg.err
		if msg.err != nil {
			t.status = fmt.Sprintf("Tasks action failed: %v", msg.err)
			return t, nil
		}
		t.status = msg.status
		t.projects = msg.loaded.projects
		t.project = msg.loaded.project
		t.board = msg.loaded.board
		t.cards = msg.loaded.cards
		t.rows = t.buildRows()
		t.clampIndex()
		t.syncList()
		return t, nil
	case TasksActionMsg:
		return t.handleAction(msg.Action)
	case tea.KeyPressMsg:
		return t.handleKey(msg)
	}
	return t, nil
}
