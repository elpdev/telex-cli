package screens

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/emailtext"
)

func (m Mail) openConversation() (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	if m.conversation == nil {
		m.status = "Conversation view is not configured"
		return m, nil
	}
	conversationID := m.messages[m.messageIndex].Meta.ConversationID
	if conversationID == 0 {
		m.status = "No conversation for this message"
		return m, nil
	}
	m.previousMode = m.mode
	m.loading = true
	m.status = "Loading conversation..."
	return m, func() tea.Msg {
		entries, err := m.conversation(context.Background(), conversationID)
		return conversationLoadedMsg{conversationID: conversationID, entries: entries, err: err}
	}
}

func (m Mail) handleConversationKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) {
		if m.previousMode == mailModeDetail {
			m.mode = mailModeDetail
		} else {
			m.mode = mailModeList
		}
		m.resetConversationViewport()
		m.status = ""
		return m, nil
	}
	switch msg.String() {
	case "tab":
		if m.conversationIndex < len(m.conversationItems)-1 {
			m.conversationIndex++
			m.resetConversationViewport()
			m.status = ""
			return m, m.loadConversationBodyCmd()
		}
		return m, nil
	case "shift+tab":
		if m.conversationIndex > 0 {
			m.conversationIndex--
			m.resetConversationViewport()
			m.status = ""
			return m, m.loadConversationBodyCmd()
		}
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Up):
		m.syncConversationViewport(mailReadWidth, 1)
		m.conversationViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.syncConversationViewport(mailReadWidth, 1)
		m.conversationViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.Reply):
		if id := m.currentConversationInboundID(); id > 0 {
			return m.editReplyDraftForMessageID(id)
		}
		m.status = "Reply is only available for inbound messages"
	case key.Matches(msg, m.keys.Forward):
		if id := m.currentConversationInboundID(); id > 0 {
			return m.editForwardDraftForMessageID(id, nil)
		}
		m.status = "Forward is only available for inbound messages"
	}
	return m, nil
}

func (m Mail) loadConversationBodyCmd() tea.Cmd {
	if len(m.conversationItems) == 0 || m.conversationIndex >= len(m.conversationItems) || m.conversationBody == nil {
		return nil
	}
	entry := m.conversationItems[m.conversationIndex]
	key := conversationEntryKey(entry)
	if _, ok := m.conversationBodyCache[key]; ok {
		return nil
	}
	return func() tea.Msg {
		body, err := m.conversationBody(context.Background(), entry)
		return conversationBodyLoadedMsg{key: key, body: body, err: err}
	}
}

func (m Mail) currentConversationInboundID() int64 {
	if len(m.conversationItems) == 0 || m.conversationIndex >= len(m.conversationItems) {
		return 0
	}
	entry := m.conversationItems[m.conversationIndex]
	if entry.Kind != "inbound" {
		return 0
	}
	return entry.RecordID
}

func (m Mail) conversationView(width, height int) string {
	var b strings.Builder
	bodyWidth := min(width, mailReadWidth)
	title := fmt.Sprintf("Mail / Conversation %d", m.conversationID)
	counter := ""
	if len(m.conversationItems) > 0 {
		title += " / " + m.conversationItems[m.conversationIndex].Subject
		counter = fmt.Sprintf("%d/%d", m.conversationIndex+1, len(m.conversationItems))
	}
	b.WriteString(mailHeader(title, counter))
	b.WriteByte('\n')
	if m.status != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", m.status))
	}
	b.WriteByte('\n')
	if len(m.conversationItems) == 0 {
		b.WriteString("No timeline entries in this conversation.\n")
		return b.String()
	}
	stripLimit := min(len(m.conversationItems), 5)
	start := max(0, min(m.conversationIndex-2, len(m.conversationItems)-stripLimit))
	layout := mailColumns(width)
	for i := start; i < start+stripLimit; i++ {
		b.WriteString(formatConversationRow(m.conversationItems[i], i == m.conversationIndex, layout))
		b.WriteByte('\n')
	}
	b.WriteString(mailSeparator(bodyWidth))
	b.WriteByte('\n')
	entry := m.conversationItems[m.conversationIndex]
	entryMeta := []string{
		strings.ToUpper(entry.Kind),
		"from " + entry.Sender,
	}
	if recipients := strings.Join(entry.Recipients, ", "); recipients != "" {
		entryMeta = append(entryMeta, "to "+recipients)
	}
	entryMeta = append(entryMeta, entry.OccurredAt.Format("2006-01-02 15:04"))
	if entry.Status != "" {
		entryMeta = append(entryMeta, entry.Status)
	}
	b.WriteString(strings.Join(entryMeta, " · "))
	b.WriteByte('\n')
	b.WriteString(mailSeparator(bodyWidth))
	b.WriteByte('\n')
	body := m.conversationBodyCache[conversationEntryKey(entry)]
	if strings.TrimSpace(body) == "" {
		body = entry.Summary
		if strings.TrimSpace(body) == "" {
			body = "(loading body...)"
		}
	}
	rendered, err := emailtext.Render(body, "", bodyWidth)
	if err != nil {
		rendered = fmt.Sprintf("(could not render body: %v)", err)
	}
	headerLines := strings.Count(b.String(), "\n")
	const reservedFooter = 2
	limit := max(1, height-headerLines-reservedFooter)
	m.syncViewport(&m.conversationViewport, bodyWidth, limit, rendered)
	bodyView := m.conversationViewport.View()
	if bodyView != "" {
		b.WriteString(bodyView)
		b.WriteByte('\n')
	}
	b.WriteString(mailSeparator(bodyWidth))
	b.WriteByte('\n')
	hint := mailFooterHint("[esc] back", "[tab/shift+tab] navigate", "[r] reply", "[f] forward", "[j/k] scroll")
	if rangeHint := viewportRangeHint(m.conversationViewport); rangeHint != "" {
		hint = mailFooterHint(hint, rangeHint)
	}
	b.WriteString(hint)
	return b.String()
}

func conversationKindLabel(kind string) string {
	if kind == "outbound" {
		return "OUT"
	}
	return "IN "
}

func conversationEntryKey(entry ConversationEntry) string {
	return fmt.Sprintf("%s:%d", entry.Kind, entry.RecordID)
}

func (m *Mail) clampConversationSelection() {
	if m.conversationIndex >= len(m.conversationItems) {
		m.conversationIndex = max(0, len(m.conversationItems)-1)
	}
	if len(m.conversationItems) == 0 {
		m.conversationIndex = 0
		m.resetConversationViewport()
	}
}
