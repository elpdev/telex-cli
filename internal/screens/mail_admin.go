package screens

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	huhkey "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
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

func NewMailAdmin(load MailAdminLoadFunc) MailAdmin {
	return MailAdmin{load: load, loading: true, keys: DefaultMailAdminKeyMap()}
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

func DefaultMailAdminKeyMap() MailAdminKeyMap {
	return MailAdminKeyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "move up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "move down")),
		Focus:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "domains/inboxes")),
		New:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
		Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Delete:   key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
		Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		Validate: key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "validate domain")),
		Pipeline: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pipeline")),
		Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
}

func (m MailAdmin) Init() tea.Cmd { return m.loadCmd() }

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

func (m MailAdmin) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if m.loading {
		return style.Render("Loading mail admin data...")
	}
	if m.form != nil {
		return style.Render(m.form.WithWidth(max(40, width-4)).WithHeight(max(8, height-3)).View())
	}
	var b strings.Builder
	b.WriteString("Mail Admin")
	if m.status != "" {
		b.WriteString(" · " + m.status)
	}
	if m.err != nil {
		b.WriteString(fmt.Sprintf("\nAPI error: %v", m.err))
	}
	b.WriteString("\n\n")
	b.WriteString(m.listColumns(width))
	if m.confirm != "" {
		b.WriteString("\n" + m.confirm + " [y/N]\n")
	}
	if m.detail != "" {
		b.WriteString("\n" + m.detail + "\n")
	} else {
		b.WriteString("\n" + m.selectionDetails() + "\n")
	}
	return style.Render(b.String())
}

func (m MailAdmin) Title() string { return "Mail Admin" }

func (m MailAdmin) KeyBindings() []key.Binding {
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Focus, m.keys.New, m.keys.Edit, m.keys.Delete, m.keys.Refresh, m.keys.Validate, m.keys.Pipeline, m.keys.Back}
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

func (m MailAdmin) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if m.confirm != "" {
		return m.handleConfirmKey(msg)
	}
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.focus == mailAdminFocusDomains && m.domainIndex > 0 {
			m.domainIndex--
			m.inboxIndex = 0
		} else if m.focus == mailAdminFocusInboxes && m.inboxIndex > 0 {
			m.inboxIndex--
		}
		m.detail = ""
	case key.Matches(msg, m.keys.Down):
		if m.focus == mailAdminFocusDomains && m.domainIndex < len(m.domains)-1 {
			m.domainIndex++
			m.inboxIndex = 0
		} else if m.focus == mailAdminFocusInboxes && m.inboxIndex < len(m.filteredInboxes())-1 {
			m.inboxIndex++
		}
		m.detail = ""
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

func (m MailAdmin) startDomainForm(kind mailAdminFormKind, domain *mail.Domain) (Screen, tea.Cmd) {
	data := mailAdminFormData{Active: true, UseFromAddressForReplyTo: true, SMTPEnableStartTLSAuto: true, SMTPAuthentication: "login"}
	var id int64
	if domain != nil {
		id = domain.ID
		data.Name = domain.Name
		data.Active = domain.Active
		data.OutboundFromName = domain.OutboundFromName
		data.OutboundFromAddress = domain.OutboundFromAddress
		data.UseFromAddressForReplyTo = domain.UseFromAddressForReplyTo
		data.ReplyToAddress = domain.ReplyToAddress
		data.SMTPHost = domain.SMTPHost
		if domain.SMTPPort > 0 {
			data.SMTPPort = strconv.Itoa(domain.SMTPPort)
		}
		data.SMTPAuthentication = domain.SMTPAuthentication
		data.SMTPEnableStartTLSAuto = domain.SMTPEnableStartTLSAuto
		data.SMTPUsername = domain.SMTPUsername
		if domain.DriveFolderID > 0 {
			data.DriveFolderID = strconv.FormatInt(domain.DriveFolderID, 10)
		}
	}
	m.formData = &data
	m.formID = id
	m.formKind = kind
	m.form = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Domain name").Value(&m.formData.Name).Validate(requiredString),
		huh.NewConfirm().Title("Active").Value(&m.formData.Active),
		huh.NewInput().Title("Drive folder ID").Description("Optional").Value(&m.formData.DriveFolderID).Validate(optionalInt64String),
		huh.NewInput().Title("Outbound from name").Value(&m.formData.OutboundFromName),
		huh.NewInput().Title("Outbound from address").Value(&m.formData.OutboundFromAddress),
		huh.NewConfirm().Title("Use from address for Reply-To").Value(&m.formData.UseFromAddressForReplyTo),
		huh.NewInput().Title("Reply-To address").Value(&m.formData.ReplyToAddress),
		huh.NewInput().Title("SMTP host").Value(&m.formData.SMTPHost),
		huh.NewInput().Title("SMTP port").Value(&m.formData.SMTPPort).Validate(optionalIntString),
		huh.NewInput().Title("SMTP authentication").Description("Allowed: login, plain, cram_md5").Suggestions([]string{"login", "plain", "cram_md5"}).Value(&m.formData.SMTPAuthentication).Validate(validateSMTPAuthentication),
		huh.NewConfirm().Title("SMTP STARTTLS auto").Value(&m.formData.SMTPEnableStartTLSAuto),
		huh.NewInput().Title("SMTP username").Value(&m.formData.SMTPUsername),
		huh.NewInput().Title("SMTP password").Description("Leave blank to keep the existing password when editing.").EchoMode(huh.EchoModePassword).Value(&m.formData.SMTPPassword),
	).Title(domainFormTitle(kind)).Description("Move between fields with up/down, j/k, or tab/shift+tab. Enter advances; submit from the last field."))
	m.form.WithKeyMap(mailAdminFormKeyMap()).WithShowHelp(true)
	return m, m.form.Init()
}

func (m MailAdmin) startInboxForm(kind mailAdminFormKind, inbox *mail.Inbox) (Screen, tea.Cmd) {
	data := mailAdminFormData{Active: true, PipelineKey: "default"}
	if domain, ok := m.selectedDomain(); ok {
		data.DomainID = strconv.FormatInt(domain.ID, 10)
	}
	var id int64
	if inbox != nil {
		id = inbox.ID
		data.DomainID = strconv.FormatInt(inbox.DomainID, 10)
		data.LocalPart = inbox.LocalPart
		data.PipelineKey = inbox.PipelineKey
		data.Description = inbox.Description
		data.Active = inbox.Active
		if inbox.DriveFolderID > 0 {
			data.DriveFolderID = strconv.FormatInt(inbox.DriveFolderID, 10)
		}
	}
	m.formData = &data
	m.formID = id
	m.formKind = kind
	m.form = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Domain ID").Value(&m.formData.DomainID).Validate(requiredInt64String),
		huh.NewInput().Title("Local part").Description("The part before @domain").Value(&m.formData.LocalPart).Validate(requiredString),
		huh.NewInput().Title("Pipeline").Description("Allowed: default, receipts").Suggestions([]string{"default", "receipts"}).Value(&m.formData.PipelineKey).Validate(validatePipelineKey),
		huh.NewInput().Title("Description").Value(&m.formData.Description),
		huh.NewConfirm().Title("Active").Value(&m.formData.Active),
		huh.NewInput().Title("Drive folder ID").Description("Optional").Value(&m.formData.DriveFolderID).Validate(optionalInt64String),
	).Title(inboxFormTitle(kind)).Description("Move between fields with up/down, j/k, or tab/shift+tab. Enter advances; submit from the last field."))
	m.form.WithKeyMap(mailAdminFormKeyMap()).WithShowHelp(true)
	return m, m.form.Init()
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

func (m MailAdmin) listColumns(width int) string {
	domainWidth := max(30, width/2-3)
	inboxWidth := max(30, width-domainWidth-4)
	domains := m.domainLines()
	inboxes := m.inboxLines()
	rows := max(len(domains), len(inboxes))
	var b strings.Builder
	b.WriteString(mailAdminPadRight(focusTitle("Domains", m.focus == mailAdminFocusDomains), domainWidth) + "  " + focusTitle("Inboxes", m.focus == mailAdminFocusInboxes) + "\n")
	for i := 0; i < rows; i++ {
		left := ""
		if i < len(domains) {
			left = domains[i]
		}
		right := ""
		if i < len(inboxes) {
			right = inboxes[i]
		}
		b.WriteString(mailAdminPadRight(mailAdminTruncate(left, domainWidth), domainWidth) + "  " + mailAdminTruncate(right, inboxWidth) + "\n")
	}
	return b.String()
}

func (m MailAdmin) domainLines() []string {
	if len(m.domains) == 0 {
		return []string{"No domains. Press n to create one."}
	}
	lines := make([]string, 0, len(m.domains))
	for i, domain := range m.domains {
		cursor := "  "
		if i == m.domainIndex {
			cursor = "> "
		}
		state := "inactive"
		if domain.Active {
			state = "active"
		}
		ready := "not ready"
		if domain.OutboundReady {
			ready = "ready"
		}
		lines = append(lines, fmt.Sprintf("%s%d  %s  %s/%s", cursor, domain.ID, domain.Name, state, ready))
	}
	return lines
}

func (m MailAdmin) inboxLines() []string {
	inboxes := m.filteredInboxes()
	if len(inboxes) == 0 {
		return []string{"No inboxes for selected domain."}
	}
	lines := make([]string, 0, len(inboxes))
	for i, inbox := range inboxes {
		cursor := "  "
		if i == m.inboxIndex {
			cursor = "> "
		}
		state := "inactive"
		if inbox.Active {
			state = "active"
		}
		lines = append(lines, fmt.Sprintf("%s%d  %s  %s  %d msg", cursor, inbox.ID, inbox.Address, state, inbox.MessageCount))
	}
	return lines
}

func (m MailAdmin) selectionDetails() string {
	if m.focus == mailAdminFocusDomains {
		domain, ok := m.selectedDomain()
		if !ok {
			return ""
		}
		return fmt.Sprintf("Domain %d · %s\nOutbound: %s · SMTP: %s:%d · From: %s", domain.ID, domain.Name, readyText(domain.OutboundReady), domain.SMTPHost, domain.SMTPPort, domain.OutboundFromAddress)
	}
	inbox, ok := m.selectedInbox()
	if !ok {
		return ""
	}
	return fmt.Sprintf("Inbox %d · %s\nPipeline: %s · Description: %s", inbox.ID, inbox.Address, inbox.PipelineKey, inbox.Description)
}

func (m MailAdmin) selectedDomain() (mail.Domain, bool) {
	if len(m.domains) == 0 || m.domainIndex < 0 || m.domainIndex >= len(m.domains) {
		return mail.Domain{}, false
	}
	return m.domains[m.domainIndex], true
}

func (m MailAdmin) selectedInbox() (mail.Inbox, bool) {
	inboxes := m.filteredInboxes()
	if len(inboxes) == 0 || m.inboxIndex < 0 || m.inboxIndex >= len(inboxes) {
		return mail.Inbox{}, false
	}
	return inboxes[m.inboxIndex], true
}

func (m MailAdmin) filteredInboxes() []mail.Inbox {
	domain, ok := m.selectedDomain()
	if !ok {
		return nil
	}
	items := make([]mail.Inbox, 0)
	for _, inbox := range m.inboxes {
		if inbox.DomainID == domain.ID {
			items = append(items, inbox)
		}
	}
	return items
}

func (m *MailAdmin) clamp() {
	if m.domainIndex >= len(m.domains) {
		m.domainIndex = max(0, len(m.domains)-1)
	}
	if m.inboxIndex >= len(m.filteredInboxes()) {
		m.inboxIndex = max(0, len(m.filteredInboxes())-1)
	}
}

func domainInputFromForm(data mailAdminFormData) (mail.DomainInput, error) {
	active := data.Active
	useReplyTo := data.UseFromAddressForReplyTo
	startTLS := data.SMTPEnableStartTLSAuto
	input := mail.DomainInput{Name: strings.TrimSpace(data.Name), Active: &active, OutboundFromName: strings.TrimSpace(data.OutboundFromName), OutboundFromAddress: strings.TrimSpace(data.OutboundFromAddress), UseFromAddressForReplyTo: &useReplyTo, ReplyToAddress: strings.TrimSpace(data.ReplyToAddress), SMTPHost: strings.TrimSpace(data.SMTPHost), SMTPAuthentication: strings.TrimSpace(data.SMTPAuthentication), SMTPEnableStartTLSAuto: &startTLS, SMTPUsername: strings.TrimSpace(data.SMTPUsername), SMTPPassword: data.SMTPPassword}
	if data.SMTPPort != "" {
		port, err := strconv.Atoi(strings.TrimSpace(data.SMTPPort))
		if err != nil || port <= 0 {
			return input, fmt.Errorf("invalid SMTP port")
		}
		input.SMTPPort = &port
	}
	if data.DriveFolderID != "" {
		id, err := strconv.ParseInt(strings.TrimSpace(data.DriveFolderID), 10, 64)
		if err != nil || id <= 0 {
			return input, fmt.Errorf("invalid drive folder ID")
		}
		input.DriveFolderID = &id
	}
	return input, nil
}

func inboxInputFromForm(data mailAdminFormData) (mail.InboxInput, error) {
	active := data.Active
	domainID, err := strconv.ParseInt(strings.TrimSpace(data.DomainID), 10, 64)
	if err != nil || domainID <= 0 {
		return mail.InboxInput{}, fmt.Errorf("invalid domain ID")
	}
	input := mail.InboxInput{DomainID: &domainID, LocalPart: strings.TrimSpace(data.LocalPart), PipelineKey: strings.TrimSpace(data.PipelineKey), Description: strings.TrimSpace(data.Description), Active: &active}
	if data.DriveFolderID != "" {
		id, err := strconv.ParseInt(strings.TrimSpace(data.DriveFolderID), 10, 64)
		if err != nil || id <= 0 {
			return input, fmt.Errorf("invalid drive folder ID")
		}
		input.DriveFolderID = &id
	}
	return input, nil
}

func requiredString(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("required")
	}
	return nil
}

func validateSMTPAuthentication(value string) error {
	switch strings.TrimSpace(value) {
	case "", "login", "plain", "cram_md5":
		return nil
	default:
		return errors.New("must be login, plain, or cram_md5")
	}
}

func validatePipelineKey(value string) error {
	switch strings.TrimSpace(value) {
	case "default", "receipts":
		return nil
	default:
		return errors.New("must be default or receipts")
	}
}

func mailAdminFormKeyMap() *huh.KeyMap {
	keys := huh.NewDefaultKeyMap()
	keys.Input.Prev = huhkey.NewBinding(huhkey.WithKeys("up", "k", "shift+tab"), huhkey.WithHelp("up/k", "previous"))
	keys.Input.Next = huhkey.NewBinding(huhkey.WithKeys("down", "j", "tab", "enter"), huhkey.WithHelp("down/j", "next"))
	keys.Confirm.Prev = huhkey.NewBinding(huhkey.WithKeys("up", "k", "shift+tab"), huhkey.WithHelp("up/k", "previous"))
	keys.Confirm.Next = huhkey.NewBinding(huhkey.WithKeys("down", "j", "tab", "enter"), huhkey.WithHelp("down/j", "next"))
	keys.Note.Prev = huhkey.NewBinding(huhkey.WithKeys("up", "k", "shift+tab"), huhkey.WithHelp("up/k", "previous"))
	keys.Note.Next = huhkey.NewBinding(huhkey.WithKeys("down", "j", "tab", "enter"), huhkey.WithHelp("down/j", "next"))
	return keys
}

func optionalIntString(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return errors.New("must be a positive number")
	}
	return nil
}

func optionalInt64String(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed <= 0 {
		return errors.New("must be a positive number")
	}
	return nil
}

func requiredInt64String(value string) error {
	if err := requiredString(value); err != nil {
		return err
	}
	return optionalInt64String(value)
}

func formatDomainValidation(validation *mail.DomainOutboundValidation) string {
	if validation == nil {
		return "No validation response."
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Valid: %t · Outbound ready: %t", validation.Valid, validation.OutboundReady))
	if len(validation.OutboundConfigurationErrors) > 0 {
		b.WriteString("\nErrors: " + strings.Join(validation.OutboundConfigurationErrors, "; "))
	}
	return b.String()
}

func formatPipeline(pipeline *mail.InboxPipeline) string {
	if pipeline == nil {
		return "No pipeline response."
	}
	var b strings.Builder
	b.WriteString("Pipeline: " + pipeline.Key)
	if len(pipeline.Steps) > 0 {
		b.WriteString("\nSteps: " + strings.Join(pipeline.Steps, " -> "))
	}
	if len(pipeline.Overrides) > 0 {
		b.WriteString(fmt.Sprintf("\nOverrides: %v", pipeline.Overrides))
	}
	return b.String()
}

func domainFormTitle(kind mailAdminFormKind) string {
	if kind == mailAdminFormDomainEdit {
		return "Edit Domain"
	}
	return "New Domain"
}

func inboxFormTitle(kind mailAdminFormKind) string {
	if kind == mailAdminFormInboxEdit {
		return "Edit Inbox"
	}
	return "New Inbox"
}

func focusTitle(title string, focused bool) string {
	if focused {
		return "> " + title
	}
	return "  " + title
}

func readyText(ready bool) string {
	if ready {
		return "ready"
	}
	return "not ready"
}

func mailAdminPadRight(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}

func mailAdminTruncate(value string, width int) string {
	if width <= 0 || len(value) <= width {
		return value
	}
	if width <= 1 {
		return value[:width]
	}
	return value[:width-1] + "."
}
