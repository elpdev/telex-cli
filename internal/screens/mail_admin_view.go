package screens

import (
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/elpdev/telex-cli/internal/mail"
	"strings"
)

func (m MailAdmin) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if m.loading {
		return style.Render("Loading mail admin data...")
	}
	if m.form != nil {
		return style.Render(m.form.WithWidth(max(40, width-4)).WithHeight(max(8, height-3)).View())
	}
	var b strings.Builder
	b.WriteString(mailHeader("Mail Admin", m.status))
	b.WriteByte('\n')
	if m.err != nil {
		b.WriteString(fmt.Sprintf("API error: %v\n", m.err))
	}
	b.WriteString(mailSeparator(width))
	b.WriteString("\n")
	b.WriteString(m.listColumns(width))
	b.WriteString(mailSeparator(width))
	b.WriteByte('\n')
	if m.confirm != "" {
		b.WriteString(m.confirm + " [y/N]\n")
	}
	if m.detail != "" {
		b.WriteString(m.detail + "\n")
	} else {
		b.WriteString(m.selectionDetails() + "\n")
	}
	b.WriteByte('\n')
	b.WriteString(mailFooterHint("[esc] back", "[tab] focus", "[n] new", "[e] edit", "[x] delete", "[v] validate", "[p] pipeline", "[r] refresh"))
	return style.Render(b.String())
}

func (m MailAdmin) listColumns(width int) string {
	domainWidth := max(30, width/2-3)
	inboxWidth := max(30, width-domainWidth-4)
	rowsHeight := max(1, max(len(m.domains), len(m.filteredInboxes())))
	domains := m.domainLines(domainWidth, rowsHeight)
	inboxes := m.inboxLines(inboxWidth, rowsHeight)
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

func (m MailAdmin) domainLines(width, height int) []string {
	if len(m.domains) == 0 {
		return []string{"No domains. Press n to create one."}
	}
	m.ensureLists()
	m.domainList.SetSize(width, height)
	return mailAdminListLines(m.domainList.View())
}

func (m MailAdmin) inboxLines(width, height int) []string {
	inboxes := m.filteredInboxes()
	if len(inboxes) == 0 {
		return []string{"No inboxes for selected domain."}
	}
	m.ensureLists()
	m.inboxList.SetSize(width, height)
	return mailAdminListLines(m.inboxList.View())
}

func mailAdminListLines(view string) []string {
	view = strings.TrimRight(view, "\n")
	if view == "" {
		return nil
	}
	return strings.Split(view, "\n")
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
	if len(m.domains) == 0 {
		return mail.Domain{}, false
	}
	return m.domains[m.clampedDomainIndex()], true
}

func (m MailAdmin) selectedInbox() (mail.Inbox, bool) {
	inboxes := m.filteredInboxes()
	if len(inboxes) == 0 {
		return mail.Inbox{}, false
	}
	return inboxes[m.clampedInboxIndex(inboxes)], true
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
	m.domainIndex = m.clampedDomainIndex()
	m.inboxIndex = m.clampedInboxIndex(m.filteredInboxes())
}

func (m MailAdmin) clampedDomainIndex() int {
	if m.domainIndex < 0 || len(m.domains) == 0 {
		return 0
	}
	if m.domainIndex >= len(m.domains) {
		return len(m.domains) - 1
	}
	return m.domainIndex
}

func (m MailAdmin) clampedInboxIndex(inboxes []mail.Inbox) int {
	if m.inboxIndex < 0 || len(inboxes) == 0 {
		return 0
	}
	if m.inboxIndex >= len(inboxes) {
		return len(inboxes) - 1
	}
	return m.inboxIndex
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
