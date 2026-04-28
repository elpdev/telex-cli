package screens

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

func (m MailAdmin) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if m.form != nil {
		return m.updateForm(msg)
	}

	switch msg := msg.(type) {
	case mailAdminLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.domains = msg.domains
			m.inboxes = msg.inboxes
			m.clamp()
			m.syncLists()
			m.status = fmt.Sprintf("Loaded %d domain(s), %d inbox(es)", len(m.domains), len(m.inboxes))
		}
		return m, nil
	case mailAdminActionDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Action failed: %v", msg.err)
			return m, nil
		}
		m.status = msg.status
		m.detail = msg.detail
		if msg.detail != "" {
			return m, nil
		}
		return m, m.loadCmd()
	case MailAdminActionMsg:
		return m.handleAction(msg.Action)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m MailAdmin) updateForm(msg tea.Msg) (Screen, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && key.Matches(keyMsg, m.keys.Back) {
		m.form = nil
		m.formKind = mailAdminFormNone
		m.status = "Cancelled"
		return m, nil
	}
	model, cmd := m.form.Update(msg)
	if form, ok := model.(*huh.Form); ok {
		m.form = form
	}
	if m.form.State == huh.StateAborted {
		m.form = nil
		m.formKind = mailAdminFormNone
		m.status = "Cancelled"
		return m, nil
	}
	if m.form.State == huh.StateCompleted {
		kind := m.formKind
		id := m.formID
		data := *m.formData
		m.form = nil
		m.formKind = mailAdminFormNone
		m.loading = true
		m.status = "Saving..."
		return m, m.saveFormCmd(kind, id, data)
	}
	return m, cmd
}
