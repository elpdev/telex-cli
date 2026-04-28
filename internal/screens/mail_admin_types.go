package screens

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/huh/v2"
	"context"
	"github.com/elpdev/telex-cli/internal/mail"
)

type MailAdminLoadFunc func(context.Context) ([]mail.Domain, []mail.Inbox, error)
type DomainSaveFunc func(context.Context, *int64, mail.DomainInput) error
type DomainDeleteFunc func(context.Context, int64) error
type DomainValidateFunc func(context.Context, int64) (*mail.DomainOutboundValidation, error)
type InboxSaveFunc func(context.Context, *int64, mail.InboxInput) error
type InboxDeleteFunc func(context.Context, int64) error
type InboxPipelineFunc func(context.Context, int64) (*mail.InboxPipeline, error)

type mailAdminFocus int

const (
	mailAdminFocusDomains mailAdminFocus = iota
	mailAdminFocusInboxes
)

type mailAdminFormKind int

const (
	mailAdminFormNone mailAdminFormKind = iota
	mailAdminFormDomainCreate
	mailAdminFormDomainEdit
	mailAdminFormInboxCreate
	mailAdminFormInboxEdit
)

type MailAdmin struct {
	load           MailAdminLoadFunc
	saveDomain     DomainSaveFunc
	deleteDomain   DomainDeleteFunc
	validateDomain DomainValidateFunc
	saveInbox      InboxSaveFunc
	deleteInbox    InboxDeleteFunc
	inboxPipeline  InboxPipelineFunc

	domains     []mail.Domain
	inboxes     []mail.Inbox
	domainList  list.Model
	inboxList   list.Model
	domainIndex int
	inboxIndex  int
	focus       mailAdminFocus

	loading bool
	status  string
	err     error
	detail  string

	confirm string

	form     *huh.Form
	formKind mailAdminFormKind
	formID   int64
	formData *mailAdminFormData

	keys MailAdminKeyMap
}

type mailAdminFormData struct {
	Name                     string
	Active                   bool
	OutboundFromName         string
	OutboundFromAddress      string
	UseFromAddressForReplyTo bool
	ReplyToAddress           string
	SMTPHost                 string
	SMTPPort                 string
	SMTPAuthentication       string
	SMTPEnableStartTLSAuto   bool
	SMTPUsername             string
	SMTPPassword             string
	DriveFolderID            string
	DomainID                 string
	LocalPart                string
	PipelineKey              string
	Description              string
}

type MailAdminKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Focus    key.Binding
	New      key.Binding
	Edit     key.Binding
	Delete   key.Binding
	Refresh  key.Binding
	Validate key.Binding
	Pipeline key.Binding
	Back     key.Binding
}

type MailAdminActionMsg struct{ Action string }

type mailAdminDomainItem struct{ domain mail.Domain }

func (i mailAdminDomainItem) FilterValue() string { return i.domain.Name }

type mailAdminInboxItem struct{ inbox mail.Inbox }

func (i mailAdminInboxItem) FilterValue() string { return i.inbox.Address }

type mailAdminLoadedMsg struct {
	domains []mail.Domain
	inboxes []mail.Inbox
	err     error
}

type mailAdminActionDoneMsg struct {
	status string
	detail string
	err    error
}
