package screens

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/emailtext"
)

func (m Mail) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if m.loading {
		return style.Render("Loading local mail cache...")
	}
	if m.err != nil {
		return style.Render(fmt.Sprintf("Mail cache error: %v\n\nRun `telex sync` to create local mail data.", m.err))
	}
	if len(m.mailboxes) == 0 {
		return style.Render("No synced mailboxes found.\n\nRun `telex sync` to populate the local mail cache.")
	}
	if m.filePickerActive {
		return style.Render(m.filePicker.View(width, height))
	}
	if m.mode == mailModeArticle && len(m.messages) > 0 {
		return style.Render(m.articleView(width, height))
	}
	if m.mode == mailModeLinks && len(m.messages) > 0 {
		return style.Render(m.linksView(width, height))
	}
	if m.mode == mailModeAttachments && len(m.messages) > 0 {
		return style.Render(m.attachmentsView(width, height))
	}
	if m.mode == mailModeConversation {
		return style.Render(m.conversationView(width, height))
	}
	if m.mode == mailModeComposeFrom {
		return style.Render(m.composeFromView(width, height))
	}
	if m.mode == mailModeDetail && len(m.messages) > 0 {
		return style.Render(m.detailView(width, height))
	}
	return style.Render(m.listView(width, height))
}

func (m Mail) listView(width, height int) string {
	var b strings.Builder
	box := m.currentBox()
	title := "Mail / " + m.mailboxes[m.mailboxIndex].Address + " / " + box
	if m.scope.Aggregate {
		title = "Mail / " + m.Title()
	}
	b.WriteString(mailHeader(title, fmt.Sprintf("%d msg", len(m.messages))))
	b.WriteByte('\n')
	if m.status != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", m.status))
	}
	if m.searchQuery != "" {
		b.WriteString(fmt.Sprintf("Filter: %s (%d/%d)\n", m.searchQuery, len(m.messages), len(m.allMessages)))
	}
	if m.remoteResults {
		b.WriteString(fmt.Sprintf("Remote results: %s (%d result(s), transient)\n", m.remoteSearchQuery, len(m.messages)))
	}
	b.WriteString("\n")
	if len(m.messages) == 0 {
		if m.scope.Aggregate {
			if m.scope.StarredOnly {
				b.WriteString("No starred cached messages. Run `telex sync` to refresh local mail.\n")
			} else if m.scope.UnreadOnly {
				b.WriteString("No unread cached inbox messages. Run `telex sync` to refresh local mail.\n")
			} else {
				b.WriteString(fmt.Sprintf("No cached %s messages across mailboxes. Run `telex sync`.\n", box))
			}
			return b.String()
		}
		b.WriteString(fmt.Sprintf("No cached %s messages for this mailbox. Run `telex sync`.\n", box))
		return b.String()
	}
	headerLines := strings.Count(b.String(), "\n")
	limit := max(1, height-headerLines-3)
	start, end, page, pages := paginatedRange(len(m.messages), m.messageIndex, limit)
	b.WriteString(fmt.Sprintf("Page %d/%d · %d-%d/%d\n", page, pages, start+1, end, len(m.messages)))
	layout := mailColumns(width)
	for i := start; i < end; i++ {
		b.WriteString(formatMailRow(m.messages[i], i == m.messageIndex, layout))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(mailListFooterHint(box, m.scope.Aggregate))
	return b.String()
}

func mailListFooterHint(box string, aggregate bool) string {
	if aggregate {
		switch box {
		case "inbox":
			return mailFooterHint("[enter] open", "[c] compose", "[/] filter", "[a] archive", "[J] junk", "[d] trash", "[?] help")
		case "junk":
			return mailFooterHint("[enter] open", "[/] filter", "[U] not junk", "[?] help")
		case "archive", "trash":
			return mailFooterHint("[enter] open", "[/] filter", "[R] restore", "[?] help")
		case "drafts":
			return mailFooterHint("[enter] open", "[c] new", "[e] edit", "[S] send", "[x] delete", "[?] help")
		default:
			return mailFooterHint("[enter] open", "[/] filter", "[?] help")
		}
	}
	switch box {
	case "inbox":
		return mailFooterHint("[enter] open", "[c] compose", "[/] filter", "[a] archive", "[J] junk", "[d] trash", "[h/l] account · [{/}] box", "[?] help")
	case "junk":
		return mailFooterHint("[enter] open", "[/] filter", "[U] not junk", "[h/l] account · [{/}] box", "[?] help")
	case "archive", "trash":
		return mailFooterHint("[enter] open", "[/] filter", "[R] restore", "[h/l] account · [{/}] box", "[?] help")
	case "drafts":
		return mailFooterHint("[enter] open", "[c] new", "[e] edit", "[S] send", "[x] delete", "[h/l] account · [{/}] box", "[?] help")
	default:
		return mailFooterHint("[enter] open", "[/] filter", "[h/l] account · [{/}] box", "[?] help")
	}
}

func (m Mail) detailView(width, height int) string {
	message := m.messages[m.messageIndex]
	bodyWidth := min(width, mailReadWidth)
	var b strings.Builder
	b.WriteString(message.Meta.Subject)
	b.WriteByte('\n')

	metaParts := []string{"from " + message.Meta.FromAddress}
	if to := strings.Join(message.Meta.To, ", "); to != "" {
		metaParts = append(metaParts, "to "+to)
	}
	if len(message.Meta.CC) > 0 {
		metaParts = append(metaParts, "cc "+strings.Join(message.Meta.CC, ", "))
	}
	metaParts = append(metaParts, "in "+message.Meta.Mailbox, message.Meta.ReceivedAt.Format("2006-01-02 15:04"))
	if message.Meta.RemoteID > 0 {
		metaParts = append(metaParts, fmt.Sprintf("#%d", message.Meta.RemoteID))
	}
	b.WriteString(strings.Join(metaParts, " · "))
	b.WriteByte('\n')

	var flags []string
	if m.currentBoxSupportsMessageActions() {
		if message.Meta.Starred {
			flags = append(flags, "★ starred")
		}
		if message.Meta.SenderTrusted {
			flags = append(flags, "✓ sender trusted")
		}
		if message.Meta.SenderBlocked {
			flags = append(flags, "⛔ sender blocked")
		}
		if message.Meta.DomainBlocked {
			flags = append(flags, "⛔ domain blocked")
		}
	}
	if len(message.Meta.Attachments) > 0 {
		flags = append(flags, fmt.Sprintf("Attachments: %d (A)", len(message.Meta.Attachments)))
	}
	if names := cachedLabelNames(message.Meta.Labels); len(names) > 0 {
		flags = append(flags, "["+strings.Join(names, ", ")+"]")
	}
	if len(flags) > 0 {
		b.WriteString(strings.Join(flags, " · "))
		b.WriteByte('\n')
	}
	if message.Meta.Status != "" {
		b.WriteString(fmt.Sprintf("Delivery status: %s\n", message.Meta.Status))
	}
	if message.Meta.RemoteError != "" {
		b.WriteString(fmt.Sprintf("Delivery error: %s\n", message.Meta.RemoteError))
	}
	if m.status != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", m.status))
	}
	b.WriteString(mailSeparator(bodyWidth))
	b.WriteByte('\n')

	body, err := emailtext.Render(message.BodyText, message.BodyHTML, bodyWidth)
	if err != nil {
		body = fmt.Sprintf("(could not render body: %v)", err)
	}
	headerLines := strings.Count(b.String(), "\n")
	const reservedFooter = 2
	limit := max(1, height-headerLines-reservedFooter)
	m.syncViewport(&m.detailViewport, bodyWidth, limit, body)
	bodyView := m.detailViewport.View()
	if bodyView != "" {
		b.WriteString(bodyView)
		b.WriteByte('\n')
	}
	b.WriteString(mailSeparator(bodyWidth))
	b.WriteByte('\n')
	hint := mailDetailFooterHint(m.currentBox())
	if rangeHint := viewportRangeHint(m.detailViewport); rangeHint != "" {
		hint = mailFooterHint(hint, rangeHint)
	}
	b.WriteString(hint)
	return b.String()
}

func (m Mail) composeFromView(width, height int) string {
	var b strings.Builder
	b.WriteString(mailHeader("Compose From", fmt.Sprintf("%d mailbox(es)", len(m.mailboxes))))
	b.WriteString("\n")
	if m.status != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", m.status))
	}
	b.WriteString("\n")
	if len(m.mailboxes) == 0 {
		b.WriteString("No synced mailboxes found. Run `telex sync`.\n")
		return b.String()
	}
	headerLines := strings.Count(b.String(), "\n")
	limit := max(1, height-headerLines-2)
	start := 0
	if m.composeFromIndex >= limit {
		start = m.composeFromIndex - limit + 1
	}
	end := min(len(m.mailboxes), start+limit)
	for i := start; i < end; i++ {
		cursor := listCursor(i == m.composeFromIndex)
		b.WriteString(cursor)
		b.WriteString(truncate(m.mailboxes[i].Address, max(8, width-2)))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(mailFooterHint("[enter] select", "[esc] cancel", "[j/k] move"))
	return b.String()
}

func mailDetailFooterHint(box string) string {
	switch box {
	case "inbox":
		return mailFooterHint("[esc] back", "[r] reply", "[a] archive", "[J] junk", "[d] trash", "[j/k] scroll")
	case "junk":
		return mailFooterHint("[esc] back", "[U] not junk", "[j/k] scroll")
	case "archive", "trash":
		return mailFooterHint("[esc] back", "[R] restore", "[j/k] scroll")
	case "drafts":
		return mailFooterHint("[esc] back", "[e] edit", "[S] send", "[x] delete", "[j/k] scroll")
	default:
		return mailFooterHint("[esc] back", "[j/k] scroll")
	}
}
