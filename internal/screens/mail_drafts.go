package screens

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

func (m Mail) handleForwardKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.forwarding = false
		m.forwardToInput = ""
		m.status = "Cancelled"
		return m, nil
	case "enter":
		to := splitDraftAddresses(m.forwardToInput)
		m.forwarding = false
		m.forwardToInput = ""
		return m.createRemoteForwardDraft(to)
	case "backspace":
		if len(m.forwardToInput) > 0 {
			m.forwardToInput = m.forwardToInput[:len(m.forwardToInput)-1]
		}
		m.status = "Forward to: " + m.forwardToInput
		return m, nil
	}
	if msg.Text != "" {
		m.forwardToInput += msg.Text
		m.status = "Forward to: " + m.forwardToInput
	}
	return m, nil
}

func (m Mail) startForward() (Screen, tea.Cmd) {
	if m.forward == nil {
		return m.editForwardDraft(nil)
	}
	m.forwarding = true
	m.forwardToInput = ""
	m.status = "Forward to: "
	return m, nil
}

func (m Mail) createRemoteForwardDraft(to []string) (Screen, tea.Cmd) {
	if len(to) == 0 {
		m.status = "No forward recipients"
		return m, nil
	}
	return m.editForwardDraft(to)
}

func (m Mail) editComposeDraft() (Screen, tea.Cmd) {
	if len(m.mailboxes) == 0 {
		return m, nil
	}
	if m.scope.Aggregate && len(m.mailboxes) > 1 {
		m.mode = mailModeComposeFrom
		m.composeFromIndex = 0
		m.status = "Choose from address"
		return m, nil
	}
	return m.editDraftForMailbox(draftTemplate(draftFields{From: m.mailboxes[m.mailboxIndex].Address}), "", m.mailboxes[m.mailboxIndex])
}

func (m Mail) sendSelectedDraft() (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	if m.currentBox() != "drafts" {
		m.status = "send is only available from drafts"
		return m, nil
	}
	if m.sendDraft == nil {
		m.status = "send draft action is not configured"
		return m, nil
	}
	index := m.messageIndex
	message := m.messages[index]
	mailbox, ok := m.mailboxForMessage(message)
	if !ok {
		m.status = "Could not find selected draft mailbox"
		return m, nil
	}
	m.status = "Sending draft..."
	return m, func() tea.Msg {
		draft, err := mailstore.ReadDraft(message.Path)
		if err != nil {
			return draftSentMsg{index: index, path: message.Path, err: err}
		}
		if err := m.sendDraft(context.Background(), mailbox, *draft); err != nil {
			return draftSentMsg{index: index, path: message.Path, err: err}
		}
		return draftSentMsg{index: index, path: message.Path}
	}
}

func (m Mail) editReplyDraft() (Screen, tea.Cmd) {
	if len(m.messages) == 0 || len(m.mailboxes) == 0 {
		return m, nil
	}
	index := m.messageIndex
	message := m.messages[index]
	updated, replyCmd := m.editReplyDraftFromMessage(message)
	if readCmd := m.markMessageReadCmd(index, message); readCmd != nil && replyCmd != nil {
		return updated, tea.Batch(readCmd, replyCmd)
	} else if readCmd != nil {
		return updated, readCmd
	}
	return updated, replyCmd
}

func (m Mail) editReplyDraftForMessageID(id int64) (Screen, tea.Cmd) {
	if message, ok := m.findMessageByRemoteID(id); ok {
		return m.editReplyDraftFromMessage(message)
	}
	entry := m.conversationItems[m.conversationIndex]
	body := m.conversationBodyCache[conversationEntryKey(entry)]
	message := mailstore.CachedMessage{Meta: mailstore.MessageMeta{RemoteID: entry.RecordID, ConversationID: entry.ConversationID, Subject: entry.Subject, FromAddress: entry.Sender, To: entry.Recipients, ReceivedAt: entry.OccurredAt}, BodyText: body}
	return m.editReplyDraftFromMessage(message)
}

func (m Mail) editReplyDraftFromMessage(message mailstore.CachedMessage) (Screen, tea.Cmd) {
	if len(m.mailboxes) == 0 {
		return m, nil
	}
	subject := message.Meta.Subject
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(subject)), "re:") {
		subject = "Re: " + subject
	}
	body := quotedReplyBody(message)
	mailbox, ok := m.mailboxForMessage(message)
	if !ok {
		mailbox = m.mailboxes[m.mailboxIndex]
	}
	return m.editDraftForMailbox(draftTemplate(draftFields{From: mailbox.Address, To: []string{message.Meta.FromAddress}, Subject: subject, Body: body, SourceMessageID: message.Meta.RemoteID, ConversationID: message.Meta.ConversationID}), "", mailbox)
}

func (m Mail) editForwardDraft(to []string) (Screen, tea.Cmd) {
	if len(m.messages) == 0 || len(m.mailboxes) == 0 {
		return m, nil
	}
	return m.editForwardDraftFromMessage(m.messages[m.messageIndex], to)
}

func (m Mail) editForwardDraftForMessageID(id int64, to []string) (Screen, tea.Cmd) {
	if message, ok := m.findMessageByRemoteID(id); ok {
		return m.editForwardDraftFromMessage(message, to)
	}
	entry := m.conversationItems[m.conversationIndex]
	body := m.conversationBodyCache[conversationEntryKey(entry)]
	message := mailstore.CachedMessage{Meta: mailstore.MessageMeta{RemoteID: entry.RecordID, ConversationID: entry.ConversationID, Subject: entry.Subject, FromAddress: entry.Sender, To: entry.Recipients, ReceivedAt: entry.OccurredAt}, BodyText: body}
	return m.editForwardDraftFromMessage(message, to)
}

func (m Mail) editForwardDraftFromMessage(message mailstore.CachedMessage, to []string) (Screen, tea.Cmd) {
	if len(m.mailboxes) == 0 {
		return m, nil
	}
	subject := message.Meta.Subject
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(subject)), "fwd:") {
		subject = "Fwd: " + subject
	}
	mailbox, ok := m.mailboxForMessage(message)
	if !ok {
		mailbox = m.mailboxes[m.mailboxIndex]
	}
	return m.editDraftForMailbox(draftTemplate(draftFields{From: mailbox.Address, To: to, Subject: subject, Body: quotedForwardBody(message), SourceMessageID: message.Meta.RemoteID, ConversationID: message.Meta.ConversationID, DraftKind: "forward"}), "", mailbox)
}

func (m Mail) findMessageByRemoteID(id int64) (mailstore.CachedMessage, bool) {
	for _, message := range m.allMessages {
		if message.Meta.RemoteID == id {
			return message, true
		}
	}
	for _, message := range m.messages {
		if message.Meta.RemoteID == id {
			return message, true
		}
	}
	return mailstore.CachedMessage{}, false
}

func (m Mail) mailboxForMessage(message mailstore.CachedMessage) (mailstore.MailboxMeta, bool) {
	for _, mailbox := range m.mailboxes {
		if message.Meta.InboxID != 0 && mailbox.InboxID == message.Meta.InboxID {
			return mailbox, true
		}
		if message.Meta.DomainName != "" && mailbox.DomainName == message.Meta.DomainName && mailbox.Address == message.Meta.FromAddress {
			return mailbox, true
		}
	}
	if !m.scope.Aggregate && len(m.mailboxes) > 0 && m.mailboxIndex >= 0 && m.mailboxIndex < len(m.mailboxes) {
		return m.mailboxes[m.mailboxIndex], true
	}
	return mailstore.MailboxMeta{}, false
}

func (m Mail) editSelectedDraft() (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	if m.currentBox() != "drafts" {
		m.status = "edit is only available from drafts"
		return m, nil
	}
	draft, err := mailstore.ReadDraft(m.messages[m.messageIndex].Path)
	if err != nil {
		m.status = fmt.Sprintf("Could not read draft: %v", err)
		return m, nil
	}
	return m.editDraft(draftTemplate(draftFields{From: draft.Meta.FromAddress, To: draft.Meta.To, CC: draft.Meta.CC, BCC: draft.Meta.BCC, Subject: draft.Meta.Subject, Body: draft.Body, SourceMessageID: draft.Meta.SourceMessageID, ConversationID: draft.Meta.ConversationID}), draft.Path)
}

func (m Mail) editDraft(content, existingPath string) (Screen, tea.Cmd) {
	mailbox := m.mailboxes[m.mailboxIndex]
	if existingPath != "" && len(m.messages) > 0 {
		if selected, ok := m.mailboxForMessage(m.messages[m.messageIndex]); ok {
			mailbox = selected
		}
	}
	return m.editDraftForMailbox(content, existingPath, mailbox)
}

func (m Mail) editDraftForMailbox(content, existingPath string, mailbox mailstore.MailboxMeta) (Screen, tea.Cmd) {
	file, err := os.CreateTemp("", "telex-draft-*.md")
	if err != nil {
		m.status = fmt.Sprintf("Could not create draft file: %v", err)
		return m, nil
	}
	path := file.Name()
	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		m.status = fmt.Sprintf("Could not write draft file: %v", err)
		return m, nil
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		m.status = fmt.Sprintf("Could not close draft file: %v", err)
		return m, nil
	}
	cmd, err := editorCommand(path)
	if err != nil {
		_ = os.Remove(path)
		m.status = err.Error()
		return m, nil
	}
	m.loading = true
	m.status = "Editing draft..."
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return draftEditedMsg{path: path, existingPath: existingPath, mailbox: mailbox, err: err}
	})
}

func (m Mail) deleteSelectedDraft() (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	if m.currentBox() != "drafts" {
		m.status = "delete is only available from drafts"
		return m, nil
	}
	index := m.messageIndex
	path := m.messages[index].Path
	m.status = "Deleting draft..."
	return m, func() tea.Msg {
		draft, err := mailstore.ReadDraft(path)
		if err != nil {
			return draftDeletedMsg{index: index, path: path, err: err}
		}
		if mailstore.HasRemoteDraft(*draft) && m.deleteDraft != nil {
			if err := m.deleteDraft(context.Background(), *draft); err != nil {
				return draftDeletedMsg{index: index, path: path, err: err}
			}
		}
		return draftDeletedMsg{index: index, path: path, err: mailstore.DeleteDraft(path)}
	}
}
