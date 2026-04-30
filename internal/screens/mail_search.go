package screens

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

func (m Mail) handleSearchKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.searchInput = ""
		m.status = ""
		return m, nil
	case "enter":
		m.searching = false
		m.searchQuery = strings.TrimSpace(m.searchInput)
		m.messageIndex = 0
		m.applySearch()
		m.clampSelection()
		if m.searchQuery == "" {
			m.status = "Search cleared"
		} else {
			m.status = fmt.Sprintf("Search: %s", m.searchQuery)
		}
		return m, nil
	case "backspace":
		if len(m.searchInput) > 0 {
			m.searchInput = m.searchInput[:len(m.searchInput)-1]
		}
		m.status = "Search: " + m.searchInput
		return m, nil
	}
	if msg.Text != "" {
		m.searchInput += msg.Text
		m.status = "Search: " + m.searchInput
	}
	return m, nil
}

func (m Mail) handleRemoteSearchKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.remoteSearching = false
		m.remoteSearchInput = ""
		m.status = ""
		return m, nil
	case "enter":
		query := strings.TrimSpace(m.remoteSearchInput)
		m.remoteSearching = false
		m.remoteSearchInput = ""
		if query == "" {
			m.status = "Remote search query is empty"
			return m, nil
		}
		return m.startRemoteSearch(query)
	case "backspace":
		if len(m.remoteSearchInput) > 0 {
			m.remoteSearchInput = m.remoteSearchInput[:len(m.remoteSearchInput)-1]
		}
		m.status = "Remote search: " + m.remoteSearchInput
		return m, nil
	}
	if msg.Text != "" {
		m.remoteSearchInput += msg.Text
		m.status = "Remote search: " + m.remoteSearchInput
	}
	return m, nil
}

func (m Mail) handleComposeFromKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.mode = mailModeList
		m.status = "Cancelled"
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.composeFromIndex > 0 {
			m.composeFromIndex--
		}
	case key.Matches(msg, m.keys.Down):
		if m.composeFromIndex < len(m.mailboxes)-1 {
			m.composeFromIndex++
		}
	case key.Matches(msg, m.keys.Open):
		if len(m.mailboxes) == 0 {
			m.mode = mailModeList
			return m, nil
		}
		mailbox := m.mailboxes[m.composeFromIndex]
		m.mode = mailModeList
		return m.editDraftForMailbox(draftTemplate(draftFields{From: mailbox.Address}), "", mailbox)
	}
	return m, nil
}

func (m Mail) startRemoteSearch(query string) (Screen, tea.Cmd) {
	mailbox := m.mailboxes[m.mailboxIndex]
	params := MailSearchParams{InboxID: mailbox.InboxID, Mailbox: remoteMailboxName(m.currentBox()), Query: query, Page: 1, PerPage: 25, Sort: "-received_at"}
	m.loading = true
	m.status = "Searching remote mail..."
	return m, func() tea.Msg {
		messages, err := m.remoteSearch(context.Background(), params)
		return remoteSearchLoadedMsg{query: query, messages: messages, err: err}
	}
}

func (m *Mail) applySearch() {
	query := strings.ToLower(strings.TrimSpace(m.searchQuery))
	if query == "" {
		m.messages = append([]mailstore.CachedMessage(nil), m.allMessages...)
		return
	}
	m.messages = m.messages[:0]
	for _, message := range m.allMessages {
		if cachedMessageMatches(message, query) {
			m.messages = append(m.messages, message)
		}
	}
}

func cachedMessageMatches(message mailstore.CachedMessage, query string) bool {
	values := []string{
		message.Meta.Subject,
		message.Meta.FromAddress,
		message.Meta.FromName,
		strings.Join(cachedLabelNames(message.Meta.Labels), " "),
		strings.Join(message.Meta.To, " "),
		strings.Join(message.Meta.CC, " "),
		message.Meta.Status,
		message.Meta.RemoteError,
		message.BodyText,
		message.BodyHTML,
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func cachedLabelNames(labels []mailstore.LabelMeta) []string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		if strings.TrimSpace(label.Name) != "" {
			names = append(names, label.Name)
		}
	}
	return names
}

func (m *Mail) updateMessageByPath(path string, update func(*mailstore.CachedMessage)) {
	for i := range m.allMessages {
		if m.allMessages[i].Path == path {
			update(&m.allMessages[i])
		}
	}
	m.applySearch()
}

func (m *Mail) removeMessageByPath(path string) {
	m.allMessages = removeCachedMessageByPath(m.allMessages, path)
	m.applySearch()
}

func removeCachedMessageByPath(messages []mailstore.CachedMessage, path string) []mailstore.CachedMessage {
	for i := range messages {
		if messages[i].Path == path {
			return append(messages[:i], messages[i+1:]...)
		}
	}
	return messages
}

func (m Mail) currentBox() string {
	if m.scope.Aggregate && m.scope.Box != "" {
		return m.scope.Box
	}
	if m.boxIndex < 0 || m.boxIndex >= len(mailBoxes) {
		return "inbox"
	}
	return mailBoxes[m.boxIndex]
}

func (m Mail) currentBoxSupportsMessageActions() bool {
	if m.remoteResults {
		return false
	}
	if m.scope.StarredOnly {
		return true
	}
	switch m.currentBox() {
	case "inbox", "junk", "archive", "trash":
		return true
	default:
		return false
	}
}

func (m Mail) currentBoxSupportsRemoteSearch() bool {
	if m.scope.StarredOnly {
		return false
	}
	switch m.currentBox() {
	case "inbox", "junk", "archive", "trash":
		return true
	default:
		return false
	}
}

func remoteMailboxName(box string) string {
	if box == "archive" {
		return "archived"
	}
	return box
}

func (m *Mail) clampSelection() {
	if m.mailboxIndex >= len(m.mailboxes) {
		m.mailboxIndex = max(0, len(m.mailboxes)-1)
	}
	if m.messageIndex >= len(m.messages) {
		m.messageIndex = max(0, len(m.messages)-1)
	}
	if len(m.messages) == 0 {
		m.mode = mailModeList
		m.resetDetailViewport()
	}
}
