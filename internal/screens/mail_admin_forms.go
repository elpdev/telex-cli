package screens

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/elpdev/telex-cli/internal/mail"
)

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
	keys.Input.Prev = key.NewBinding(key.WithKeys("up", "k", "shift+tab"), key.WithHelp("up/k", "previous"))
	keys.Input.Next = key.NewBinding(key.WithKeys("down", "j", "tab", "enter"), key.WithHelp("down/j", "next"))
	keys.Confirm.Prev = key.NewBinding(key.WithKeys("up", "k", "shift+tab"), key.WithHelp("up/k", "previous"))
	keys.Confirm.Next = key.NewBinding(key.WithKeys("down", "j", "tab", "enter"), key.WithHelp("down/j", "next"))
	keys.Note.Prev = key.NewBinding(key.WithKeys("up", "k", "shift+tab"), key.WithHelp("up/k", "previous"))
	keys.Note.Next = key.NewBinding(key.WithKeys("down", "j", "tab", "enter"), key.WithHelp("down/j", "next"))
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
