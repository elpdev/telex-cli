package screens

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

type taskCopyFinishedMsg struct {
	label string
	err   error
}

func (t Tasks) copyValue(value, label string) (Screen, tea.Cmd) {
	if value == "" {
		t.status = "Nothing to copy"
		return t, nil
	}
	cmd, err := clipboardCommand(value)
	if err != nil {
		t.status = err.Error()
		return t, nil
	}
	t.status = "Copying " + label + "..."
	return t, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return taskCopyFinishedMsg{label: label, err: err}
	})
}

func (t Tasks) copyCardBody() (Screen, tea.Cmd) {
	if t.detail == nil {
		return t, nil
	}
	return t.copyValue(t.detail.Body, "card body")
}

func (t Tasks) copyCardID() (Screen, tea.Cmd) {
	if t.detail == nil {
		return t, nil
	}
	value := fmt.Sprintf("%d %d", t.detail.Meta.ProjectID, t.detail.Meta.RemoteID)
	return t.copyValue(value, "card id")
}

func (t Tasks) copySelectedCardBody() (Screen, tea.Cmd) {
	row, ok := t.selectedRow()
	if !ok || row.Card == nil {
		t.status = "Nothing to copy"
		return t, nil
	}
	return t.copyValue(row.Card.Body, "card body")
}

func (t Tasks) copySelectedCardID() (Screen, tea.Cmd) {
	row, ok := t.selectedRow()
	if !ok || row.Card == nil {
		t.status = "Nothing to copy"
		return t, nil
	}
	value := fmt.Sprintf("%d %d", row.Card.Meta.ProjectID, row.Card.Meta.RemoteID)
	return t.copyValue(value, "card id")
}

func (t Tasks) copySelectedProjectName() (Screen, tea.Cmd) {
	row, ok := t.selectedRow()
	if !ok || row.Project == nil {
		t.status = "Nothing to copy"
		return t, nil
	}
	return t.copyValue(row.Project.Meta.Name, "project name")
}

func (t Tasks) copySelectedProjectID() (Screen, tea.Cmd) {
	row, ok := t.selectedRow()
	if !ok || row.Project == nil {
		t.status = "Nothing to copy"
		return t, nil
	}
	value := fmt.Sprintf("%d", row.Project.Meta.RemoteID)
	return t.copyValue(value, "project id")
}
