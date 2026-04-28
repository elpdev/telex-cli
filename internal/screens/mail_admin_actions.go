package screens

import (
	"context"
	"errors"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (m MailAdmin) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if m.confirm != "" {
		return m.handleConfirmKey(msg)
	}
	switch {
	case key.Matches(msg, m.keys.Focus):
		if m.focus == mailAdminFocusDomains {
			m.focus = mailAdminFocusInboxes
		} else {
			m.focus = mailAdminFocusDomains
		}
		m.detail = ""
	case key.Matches(msg, m.keys.New):
		return m.startNewForm()
	case key.Matches(msg, m.keys.Edit):
		return m.startEditForm()
	case key.Matches(msg, m.keys.Delete):
		return m.startDeleteConfirm()
	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		m.status = "Reloading..."
		m.detail = ""
		return m, m.loadCmd()
	case key.Matches(msg, m.keys.Validate):
		return m.validateSelectedDomain()
	case key.Matches(msg, m.keys.Pipeline):
		return m.showSelectedPipeline()
	case key.Matches(msg, m.keys.Back):
		m.detail = ""
	default:
		m.updateFocusedList(msg)
	}
	return m, nil
}

func (m MailAdmin) handleAction(action string) (Screen, tea.Cmd) {
	switch action {
	case "new-domain":
		m.focus = mailAdminFocusDomains
		return m.startDomainForm(mailAdminFormDomainCreate, nil)
	case "new-inbox":
		m.focus = mailAdminFocusInboxes
		return m.startInboxForm(mailAdminFormInboxCreate, nil)
	case "validate-domain":
		return m.validateSelectedDomain()
	case "pipeline":
		return m.showSelectedPipeline()
	case "refresh":
		m.loading = true
		m.status = "Reloading..."
		return m, m.loadCmd()
	}
	return m, nil
}

func (m MailAdmin) handleConfirmKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if msg.String() != "y" && msg.String() != "Y" {
		m.confirm = ""
		m.status = "Cancelled"
		return m, nil
	}
	m.confirm = ""
	m.loading = true
	if m.focus == mailAdminFocusDomains {
		domain, ok := m.selectedDomain()
		if !ok || m.deleteDomain == nil {
			return m, nil
		}
		return m, func() tea.Msg {
			err := m.deleteDomain(context.Background(), domain.ID)
			return mailAdminActionDoneMsg{status: "Domain deleted", err: err}
		}
	}
	inbox, ok := m.selectedInbox()
	if !ok || m.deleteInbox == nil {
		return m, nil
	}
	return m, func() tea.Msg {
		err := m.deleteInbox(context.Background(), inbox.ID)
		return mailAdminActionDoneMsg{status: "Inbox deleted", err: err}
	}
}

func (m MailAdmin) startNewForm() (Screen, tea.Cmd) {
	if m.focus == mailAdminFocusDomains {
		return m.startDomainForm(mailAdminFormDomainCreate, nil)
	}
	return m.startInboxForm(mailAdminFormInboxCreate, nil)
}

func (m MailAdmin) startEditForm() (Screen, tea.Cmd) {
	if m.focus == mailAdminFocusDomains {
		domain, ok := m.selectedDomain()
		if !ok {
			return m, nil
		}
		return m.startDomainForm(mailAdminFormDomainEdit, &domain)
	}
	inbox, ok := m.selectedInbox()
	if !ok {
		return m, nil
	}
	return m.startInboxForm(mailAdminFormInboxEdit, &inbox)
}

func (m MailAdmin) startDeleteConfirm() (Screen, tea.Cmd) {
	if m.focus == mailAdminFocusDomains {
		domain, ok := m.selectedDomain()
		if ok {
			m.confirm = "Delete domain " + domain.Name + " and all of its inboxes?"
		}
		return m, nil
	}
	inbox, ok := m.selectedInbox()
	if ok {
		m.confirm = "Delete inbox " + inbox.Address + "?"
	}
	return m, nil
}

func (m MailAdmin) saveFormCmd(kind mailAdminFormKind, id int64, data mailAdminFormData) tea.Cmd {
	return func() tea.Msg {
		switch kind {
		case mailAdminFormDomainCreate, mailAdminFormDomainEdit:
			if m.saveDomain == nil {
				return mailAdminActionDoneMsg{err: errors.New("domain save is unavailable")}
			}
			input, err := domainInputFromForm(data)
			if err != nil {
				return mailAdminActionDoneMsg{err: err}
			}
			var ptr *int64
			if kind == mailAdminFormDomainEdit {
				ptr = &id
			}
			err = m.saveDomain(context.Background(), ptr, input)
			return mailAdminActionDoneMsg{status: "Domain saved", err: err}
		case mailAdminFormInboxCreate, mailAdminFormInboxEdit:
			if m.saveInbox == nil {
				return mailAdminActionDoneMsg{err: errors.New("inbox save is unavailable")}
			}
			input, err := inboxInputFromForm(data)
			if err != nil {
				return mailAdminActionDoneMsg{err: err}
			}
			var ptr *int64
			if kind == mailAdminFormInboxEdit {
				ptr = &id
			}
			err = m.saveInbox(context.Background(), ptr, input)
			return mailAdminActionDoneMsg{status: "Inbox saved", err: err}
		}
		return mailAdminActionDoneMsg{err: errors.New("unknown form")}
	}
}

func (m MailAdmin) validateSelectedDomain() (Screen, tea.Cmd) {
	domain, ok := m.selectedDomain()
	if !ok || m.validateDomain == nil {
		return m, nil
	}
	m.loading = true
	m.status = "Validating outbound configuration..."
	return m, func() tea.Msg {
		validation, err := m.validateDomain(context.Background(), domain.ID)
		if err != nil {
			return mailAdminActionDoneMsg{err: err}
		}
		return mailAdminActionDoneMsg{status: "Domain validation complete", detail: formatDomainValidation(validation)}
	}
}

func (m MailAdmin) showSelectedPipeline() (Screen, tea.Cmd) {
	inbox, ok := m.selectedInbox()
	if !ok || m.inboxPipeline == nil {
		return m, nil
	}
	m.loading = true
	m.status = "Loading pipeline..."
	return m, func() tea.Msg {
		pipeline, err := m.inboxPipeline(context.Background(), inbox.ID)
		if err != nil {
			return mailAdminActionDoneMsg{err: err}
		}
		return mailAdminActionDoneMsg{status: "Pipeline loaded", detail: formatPipeline(pipeline)}
	}
}

func (m MailAdmin) loadCmd() tea.Cmd {
	return func() tea.Msg {
		if m.load == nil {
			return mailAdminLoadedMsg{err: errors.New("mail admin API is unavailable")}
		}
		domains, inboxes, err := m.load(context.Background())
		return mailAdminLoadedMsg{domains: domains, inboxes: inboxes, err: err}
	}
}
