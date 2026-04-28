package screens

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"github.com/elpdev/telex-cli/internal/emailtext"
)

func (m Mail) attachmentsView(width, height int) string {
	message := m.messages[m.messageIndex]
	var b strings.Builder
	attachments := message.Meta.Attachments
	b.WriteString(mailHeader("Mail / Attachments / "+message.Meta.Subject, fmt.Sprintf("%d items", len(attachments))))
	b.WriteByte('\n')
	if m.status != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", m.status))
	}
	b.WriteString("\n")
	if len(attachments) == 0 {
		b.WriteString("No attachments on this message.\n\n")
		b.WriteString(mailFooterHint("[esc] back"))
		return b.String()
	}
	headerLines := strings.Count(b.String(), "\n")
	const reservedFooter = 2
	limit := max(1, height-headerLines-reservedFooter)
	start := 0
	if m.attachmentIndex >= limit {
		start = m.attachmentIndex - limit + 1
	}
	end := min(len(attachments), start+limit)
	for i := start; i < end; i++ {
		attachment := attachments[i]
		cursor := listCursor(i == m.attachmentIndex)
		line := fmt.Sprintf("%s%d. %s %s %s", cursor, i+1, attachment.Filename, attachment.ContentType, formatBytes(attachment.ByteSize))
		b.WriteString(truncate(line, width))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(mailFooterHint("[esc] back", "[enter] open", "[S] save…", "[y] copy URL"))
	return b.String()
}

func (m Mail) linksView(width, height int) string {
	message := m.messages[m.messageIndex]
	var b strings.Builder
	b.WriteString(mailHeader("Mail / Links / "+message.Meta.Subject, fmt.Sprintf("%d links", len(m.links))))
	b.WriteByte('\n')
	if m.status != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", m.status))
	}
	b.WriteString("\n")
	if len(m.links) == 0 {
		b.WriteString("No links found in this message.\n\n")
		b.WriteString(mailFooterHint("[esc] back"))
		return b.String()
	}
	headerLines := strings.Count(b.String(), "\n")
	const reservedFooter = 2
	limit := max(1, height-headerLines-reservedFooter)
	start := 0
	if m.linkIndex >= limit {
		start = m.linkIndex - limit + 1
	}
	end := min(len(m.links), start+limit)
	for i := start; i < end; i++ {
		link := m.links[i]
		cursor := listCursor(i == m.linkIndex)
		line := fmt.Sprintf("%s%s (%s)", cursor, link.Text, link.URL)
		b.WriteString(truncate(line, width))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(mailFooterHint("[esc] back", "[enter] open", "[e] extract", "[y] copy"))
	return b.String()
}

func (m Mail) articleView(width, height int) string {
	var b strings.Builder
	bodyWidth := min(width, mailReadWidth)
	b.WriteString(mailHeader("Mail / Article", m.articleURL))
	b.WriteByte('\n')
	if m.status != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", m.status))
	}
	b.WriteString(mailSeparator(bodyWidth))
	b.WriteByte('\n')
	article, err := emailtext.RenderMarkdown(m.article, bodyWidth)
	if err != nil {
		article = m.article
	}
	headerLines := strings.Count(b.String(), "\n")
	const reservedFooter = 2
	limit := max(1, height-headerLines-reservedFooter)
	m.syncViewport(&m.articleViewport, bodyWidth, limit, article)
	bodyView := m.articleViewport.View()
	if bodyView != "" {
		b.WriteString(bodyView)
		b.WriteByte('\n')
	}
	b.WriteString(mailSeparator(bodyWidth))
	b.WriteByte('\n')
	hint := mailFooterHint("[esc] back", "[enter] open in browser", "[y] copy URL", "[j/k] scroll")
	if rangeHint := viewportRangeHint(m.articleViewport); rangeHint != "" {
		hint = mailFooterHint(hint, rangeHint)
	}
	b.WriteString(hint)
	return b.String()
}

func (m *Mail) resetDetailViewport() {
	m.detailViewport = viewport.New()
}

func (m *Mail) resetArticleViewport() {
	m.articleViewport = viewport.New()
}

func (m *Mail) resetConversationViewport() {
	m.conversationViewport = viewport.New()
}

func (m *Mail) syncDetailViewport(width, height int) {
	if len(m.messages) == 0 {
		m.syncViewport(&m.detailViewport, width, height, "")
		return
	}
	body, err := emailtext.Render(m.messages[m.messageIndex].BodyText, m.messages[m.messageIndex].BodyHTML, width)
	if err != nil {
		body = fmt.Sprintf("(could not render body: %v)", err)
	}
	m.syncViewport(&m.detailViewport, width, height, body)
}

func (m *Mail) syncArticleViewport(width, height int) {
	article, err := emailtext.RenderMarkdown(m.article, width)
	if err != nil {
		article = m.article
	}
	m.syncViewport(&m.articleViewport, width, height, article)
}

func (m *Mail) syncConversationViewport(width, height int) {
	if len(m.conversationItems) == 0 || m.conversationIndex >= len(m.conversationItems) {
		m.syncViewport(&m.conversationViewport, width, height, "")
		return
	}
	entry := m.conversationItems[m.conversationIndex]
	body := m.conversationBodyCache[conversationEntryKey(entry)]
	if strings.TrimSpace(body) == "" {
		body = entry.Summary
		if strings.TrimSpace(body) == "" {
			body = "(loading body...)"
		}
	}
	rendered, err := emailtext.Render(body, "", width)
	if err != nil {
		rendered = fmt.Sprintf("(could not render body: %v)", err)
	}
	m.syncViewport(&m.conversationViewport, width, height, rendered)
}
