package screens

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/mail"
)

func (m *MailAdmin) updateFocusedList(msg tea.KeyPressMsg) {
	m.ensureLists()
	if m.focus == mailAdminFocusDomains {
		updated, _ := m.domainList.Update(msg)
		m.domainList = updated
		previousDomain := m.domainIndex
		m.domainIndex = m.domainList.GlobalIndex()
		m.clamp()
		if m.domainIndex != previousDomain {
			m.inboxIndex = 0
			m.syncInboxList()
		}
	} else {
		updated, _ := m.inboxList.Update(msg)
		m.inboxList = updated
		m.inboxIndex = m.inboxList.GlobalIndex()
		m.clamp()
	}
	m.detail = ""
}

func (m *MailAdmin) ensureLists() {
	if len(m.domainList.Items()) != len(m.domains) {
		m.syncDomainList()
	} else {
		m.domainList.Select(m.clampedDomainIndex())
	}
	if len(m.inboxList.Items()) != len(m.filteredInboxes()) {
		m.syncInboxList()
	} else {
		m.inboxList.Select(m.clampedInboxIndex(m.filteredInboxes()))
	}
}

func (m *MailAdmin) syncLists() {
	m.syncDomainList()
	m.syncInboxList()
}

func (m *MailAdmin) syncDomainList() {
	m.domainIndex = m.clampedDomainIndex()
	m.domainList = newMailAdminDomainList(m.domains, m.domainIndex, m.domainList.Width(), m.domainList.Height())
}

func (m *MailAdmin) syncInboxList() {
	inboxes := m.filteredInboxes()
	m.inboxIndex = m.clampedInboxIndex(inboxes)
	m.inboxList = newMailAdminInboxList(inboxes, m.inboxIndex, m.inboxList.Width(), m.inboxList.Height())
}

func newMailAdminDomainList(domains []mail.Domain, selected, width, height int) list.Model {
	items := make([]list.Item, 0, len(domains))
	for _, domain := range domains {
		items = append(items, mailAdminDomainItem{domain: domain})
	}
	m := newMailAdminList(items, mailAdminDomainDelegate{}, selected, width, height)
	return m
}

func newMailAdminInboxList(inboxes []mail.Inbox, selected, width, height int) list.Model {
	items := make([]list.Item, 0, len(inboxes))
	for _, inbox := range inboxes {
		items = append(items, mailAdminInboxItem{inbox: inbox})
	}
	m := newMailAdminList(items, mailAdminInboxDelegate{}, selected, width, height)
	return m
}

func newMailAdminList(items []list.Item, delegate list.ItemDelegate, selected, width, height int) list.Model {
	return newSimpleList(items, delegate, selected, width, height)
}

type mailAdminDomainDelegate struct{ simpleDelegate }

func (d mailAdminDomainDelegate) Render(w io.Writer, model list.Model, index int, item list.Item) {
	domainItem, ok := item.(mailAdminDomainItem)
	if !ok {
		return
	}
	domain := domainItem.domain
	cursor := listCursor(index == model.Index())
	state := "inactive"
	if domain.Active {
		state = "active"
	}
	ready := "not ready"
	if domain.OutboundReady {
		ready = "ready"
	}
	line := fmt.Sprintf("%s%d  %s  %s/%s", cursor, domain.ID, domain.Name, state, ready)
	_, _ = io.WriteString(w, mailAdminPadRight(line, model.Width()))
}

type mailAdminInboxDelegate struct{ simpleDelegate }

func (d mailAdminInboxDelegate) Render(w io.Writer, model list.Model, index int, item list.Item) {
	inboxItem, ok := item.(mailAdminInboxItem)
	if !ok {
		return
	}
	inbox := inboxItem.inbox
	cursor := listCursor(index == model.Index())
	state := "inactive"
	if inbox.Active {
		state = "active"
	}
	line := fmt.Sprintf("%s%d  %s  %s  %d msg", cursor, inbox.ID, inbox.Address, state, inbox.MessageCount)
	_, _ = io.WriteString(w, mailAdminPadRight(line, model.Width()))
}
