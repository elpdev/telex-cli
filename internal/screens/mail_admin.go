package screens

import (
	tea "charm.land/bubbletea/v2"
)

func NewMailAdmin(load MailAdminLoadFunc) MailAdmin {
	return MailAdmin{load: load, loading: true, keys: DefaultMailAdminKeyMap(), domainList: newMailAdminDomainList(nil, 0, 0, 0), inboxList: newMailAdminInboxList(nil, 0, 0, 0)}
}

func (m MailAdmin) WithActions(saveDomain DomainSaveFunc, deleteDomain DomainDeleteFunc, validateDomain DomainValidateFunc, saveInbox InboxSaveFunc, deleteInbox InboxDeleteFunc, inboxPipeline InboxPipelineFunc) MailAdmin {
	m.saveDomain = saveDomain
	m.deleteDomain = deleteDomain
	m.validateDomain = validateDomain
	m.saveInbox = saveInbox
	m.deleteInbox = deleteInbox
	m.inboxPipeline = inboxPipeline
	return m
}

func (m MailAdmin) Init() tea.Cmd { return m.loadCmd() }

func (m MailAdmin) Title() string { return "Mail Admin" }
