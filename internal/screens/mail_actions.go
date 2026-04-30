package screens

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

func (m Mail) toggleSelectedRead() (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	if m.remoteResults {
		m.status = "Remote search results are read-only; run sync to cache actions"
		return m, nil
	}
	if !m.currentBoxSupportsMessageActions() {
		m.status = "Read/unread is only available for message boxes"
		return m, nil
	}
	if m.toggleRead == nil {
		m.status = "Read/unread action is not configured"
		return m, nil
	}
	index := m.messageIndex
	message := m.messages[index]
	desiredRead := !message.Meta.Read
	m.status = "Updating read state..."
	return m, func() tea.Msg {
		if err := m.toggleRead(context.Background(), message.Meta.RemoteID, desiredRead); err != nil {
			return messageReadToggledMsg{index: index, path: message.Path, read: desiredRead, err: err}
		}
		if _, err := mailstore.SetCachedMessageRead(message.Path, desiredRead, time.Now()); err != nil {
			return messageReadToggledMsg{index: index, path: message.Path, read: desiredRead, err: err}
		}
		return messageReadToggledMsg{index: index, path: message.Path, read: desiredRead}
	}
}

func (m Mail) markSelectedRead() (Screen, tea.Cmd) {
	if len(m.messages) == 0 || m.messageIndex < 0 || m.messageIndex >= len(m.messages) {
		return m, nil
	}
	message := m.messages[m.messageIndex]
	if message.Meta.Read {
		return m, nil
	}
	if m.remoteResults || !m.currentBoxSupportsMessageActions() {
		return m, nil
	}
	if m.toggleRead == nil {
		m.status = "Read/unread action is not configured"
		return m, nil
	}
	index := m.messageIndex
	m.status = "Marking read..."
	return m, func() tea.Msg {
		if err := m.toggleRead(context.Background(), message.Meta.RemoteID, true); err != nil {
			return messageReadToggledMsg{index: index, path: message.Path, read: true, err: err}
		}
		if _, err := mailstore.SetCachedMessageRead(message.Path, true, time.Now()); err != nil {
			return messageReadToggledMsg{index: index, path: message.Path, read: true, err: err}
		}
		return messageReadToggledMsg{index: index, path: message.Path, read: true}
	}
}

func (m Mail) toggleSelectedStar() (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	if m.remoteResults {
		m.status = "Remote search results are read-only; run sync to cache actions"
		return m, nil
	}
	if !m.currentBoxSupportsMessageActions() {
		m.status = "Star/unstar is only available for message boxes"
		return m, nil
	}
	if m.toggleStar == nil {
		m.status = "Star/unstar action is not configured"
		return m, nil
	}
	index := m.messageIndex
	message := m.messages[index]
	desiredStarred := !message.Meta.Starred
	m.status = "Updating star state..."
	return m, func() tea.Msg {
		if err := m.toggleStar(context.Background(), message.Meta.RemoteID, desiredStarred); err != nil {
			return messageStarToggledMsg{index: index, path: message.Path, starred: desiredStarred, err: err}
		}
		if _, err := mailstore.SetCachedMessageStarred(message.Path, desiredStarred, time.Now()); err != nil {
			return messageStarToggledMsg{index: index, path: message.Path, starred: desiredStarred, err: err}
		}
		return messageStarToggledMsg{index: index, path: message.Path, starred: desiredStarred}
	}
}

func (m Mail) moveSelectedMessage(action string) (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	if m.remoteResults {
		m.status = "Remote search results are read-only; run sync to cache actions"
		return m, nil
	}
	fromBox := m.selectedMessageBox()
	moveRemote := m.archive
	toBox := "archive"
	status := "Archiving..."
	switch action {
	case "archive":
		if fromBox != "inbox" {
			m.status = "archive is only available from inbox"
			return m, nil
		}
	case "junk":
		if fromBox != "inbox" {
			m.status = "junk is only available from inbox"
			return m, nil
		}
		moveRemote = m.junk
		toBox = "junk"
		status = "Moving to junk..."
	case "not-junk":
		if fromBox != "junk" {
			m.status = "not junk is only available from junk"
			return m, nil
		}
		moveRemote = m.notJunk
		toBox = "inbox"
		status = "Moving to inbox..."
	case "trash":
		if fromBox != "inbox" {
			m.status = "trash is only available from inbox"
			return m, nil
		}
		moveRemote = m.trash
		toBox = "trash"
		status = "Moving to trash..."
	case "restore":
		if fromBox != "archive" && fromBox != "trash" {
			m.status = "restore is only available from archive or trash"
			return m, nil
		}
		moveRemote = m.restore
		toBox = "inbox"
		status = "Restoring..."
	default:
		m.status = fmt.Sprintf("unknown message action %q", action)
		return m, nil
	}
	if moveRemote == nil {
		m.status = fmt.Sprintf("%s action is not configured", action)
		return m, nil
	}
	index := m.messageIndex
	message := m.messages[index]
	mailbox, ok := m.mailboxForMessage(message)
	if !ok {
		m.status = "Could not find selected message mailbox"
		return m, nil
	}
	m.status = status
	return m, func() tea.Msg {
		if err := moveRemote(context.Background(), message.Meta.RemoteID); err != nil {
			return messageMovedMsg{index: index, path: message.Path, action: action, err: err}
		}
		mailboxPath, err := m.store.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
		if err != nil {
			return messageMovedMsg{index: index, path: message.Path, action: action, err: err}
		}
		if _, err := mailstore.MoveCachedMessage(mailboxPath, fromBox, toBox, message.Path, time.Now()); err != nil {
			return messageMovedMsg{index: index, path: message.Path, action: action, err: err}
		}
		return messageMovedMsg{index: index, path: message.Path, action: action}
	}
}

func (m Mail) selectedMessageBox() string {
	if m.scope.StarredOnly && len(m.messages) > 0 && m.messageIndex >= 0 && m.messageIndex < len(m.messages) {
		return m.messages[m.messageIndex].Meta.Mailbox
	}
	return m.currentBox()
}
