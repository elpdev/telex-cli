package screens

import (
	"context"
	"fmt"
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

func (m Mail) loadCmd() tea.Cmd {
	mailboxIndex := m.mailboxIndex
	box := m.currentBox()
	return func() tea.Msg {
		return m.load(mailboxIndex, box)
	}
}

func (m Mail) syncCmd() tea.Cmd {
	mailboxIndex := m.mailboxIndex
	box := m.currentBox()
	return func() tea.Msg {
		result, err := m.sync(context.Background())
		loaded := m.load(mailboxIndex, box)
		return mailSyncedMsg{result: result, loaded: loaded, err: err}
	}
}

func (m Mail) load(mailboxIndex int, box string) mailLoadedMsg {
	mailboxes, err := m.store.ListMailboxes()
	if err != nil {
		return mailLoadedMsg{err: err}
	}
	if len(mailboxes) == 0 {
		return mailLoadedMsg{mailboxes: mailboxes}
	}
	if m.scope.Aggregate {
		messages, err := m.loadAggregate(mailboxes)
		return mailLoadedMsg{mailboxes: mailboxes, messages: messages, err: err}
	}
	if mailboxIndex >= len(mailboxes) {
		mailboxIndex = len(mailboxes) - 1
	}
	mailboxPath, err := m.store.MailboxPath(mailboxes[mailboxIndex].DomainName, mailboxes[mailboxIndex].LocalPart)
	if err != nil {
		return mailLoadedMsg{mailboxes: mailboxes, err: err}
	}
	messages, err := listCachedBox(mailboxPath, box)
	return mailLoadedMsg{mailboxes: mailboxes, messages: messages, err: err}
}

func (m Mail) loadAggregate(mailboxes []mailstore.MailboxMeta) ([]mailstore.CachedMessage, error) {
	messages := []mailstore.CachedMessage{}
	box := m.currentBox()
	for _, mailbox := range mailboxes {
		mailboxPath, err := m.store.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
		if err != nil {
			return nil, err
		}
		cached, err := listCachedBox(mailboxPath, box)
		if err != nil {
			return nil, err
		}
		for _, message := range cached {
			if m.scope.UnreadOnly && message.Meta.Read {
				continue
			}
			messages = append(messages, message)
		}
	}
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Meta.ReceivedAt.After(messages[j].Meta.ReceivedAt)
	})
	return messages, nil
}

func syncStatus(result MailSyncResult) string {
	status := fmt.Sprintf("Synced %d mailbox(es), %d inbox message(s)", result.ActiveMailboxes, result.InboxMessages)
	if result.OutboxItems > 0 {
		status = fmt.Sprintf("%s, %d outbox item(s)", status, result.OutboxItems)
	}
	if result.DraftItems > 0 {
		status = fmt.Sprintf("%s, %d remote draft(s)", status, result.DraftItems)
	}
	if result.BodyErrors > 0 || result.InboxErrors > 0 {
		status = fmt.Sprintf("%s with warnings", status)
	}
	return status
}

func listCachedBox(mailboxPath, box string) ([]mailstore.CachedMessage, error) {
	switch box {
	case "inbox", "junk", "archive", "trash":
		return mailstore.ListMessages(mailboxPath, box)
	case "sent":
		drafts, err := mailstore.ListSent(mailboxPath)
		if err != nil {
			return nil, err
		}
		return draftsToCachedMessages(drafts), nil
	case "outbox":
		drafts, err := mailstore.ListOutbox(mailboxPath)
		if err != nil {
			return nil, err
		}
		return draftsToCachedMessages(drafts), nil
	case "drafts":
		drafts, err := mailstore.ListDrafts(mailboxPath)
		if err != nil {
			return nil, err
		}
		return draftsToCachedMessages(drafts), nil
	default:
		return nil, fmt.Errorf("unknown mail box %q", box)
	}
}
