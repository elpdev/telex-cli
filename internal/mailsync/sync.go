package mailsync

import (
	"context"
	"fmt"
	"time"

	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

type Result struct {
	ActiveMailboxes  int
	SkippedMailboxes int
	OutboxItems      int
	DraftItems       int
	InboxMessages    int
	BodyErrors       int
	InboxErrors      int
	OutboxUpdates    []OutboxUpdate
}

type OutboxUpdate struct {
	Mailbox  string
	RemoteID int64
	Status   string
	Subject  string
	Path     string
}

func SyncMailboxes(ctx context.Context, store mailstore.Store, service *mail.Service) (*mailstore.SyncResult, error) {
	bootstrap, err := service.Mailboxes(ctx)
	if err != nil {
		return nil, err
	}
	return store.SyncMailboxes(bootstrap, time.Now())
}

func Run(ctx context.Context, store mailstore.Store, service *mail.Service, mailboxAddress string) (Result, error) {
	syncResult, err := SyncMailboxes(ctx, store, service)
	if err != nil {
		return Result{}, err
	}
	result := Result{ActiveMailboxes: len(syncResult.Created), SkippedMailboxes: len(syncResult.Skipped)}
	mailboxes := syncResult.Created
	if mailboxAddress != "" {
		mailbox, _, err := store.FindMailboxByAddress(mailboxAddress)
		if err != nil {
			return result, err
		}
		mailboxes = []mailstore.MailboxMeta{*mailbox}
	}
	for _, mailbox := range mailboxes {
		draftCount, err := SyncRemoteDraftsForMailbox(ctx, service, store, mailbox)
		if err != nil {
			return result, fmt.Errorf("sync remote drafts for %s: %w", mailbox.Address, err)
		}
		result.DraftItems += draftCount
		outboxUpdates, err := SyncOutboxForMailbox(ctx, service, store, mailbox)
		if err != nil {
			return result, fmt.Errorf("sync outbox for %s: %w", mailbox.Address, err)
		}
		for i := range outboxUpdates {
			outboxUpdates[i].Mailbox = mailbox.Address
		}
		result.OutboxItems += len(outboxUpdates)
		result.OutboxUpdates = append(result.OutboxUpdates, outboxUpdates...)
		count, bodyErrors, err := SyncInboxForMailbox(ctx, service, store, mailbox)
		result.InboxMessages += count
		result.BodyErrors += bodyErrors
		if err != nil {
			result.InboxErrors++
			continue
		}
	}
	return result, nil
}

func SyncRemoteDraftsForMailbox(ctx context.Context, service *mail.Service, store mailstore.Store, mailbox mailstore.MailboxMeta) (int, error) {
	count := 0
	page := 1
	const perPage = 100
	for {
		outbound, pagination, err := service.ListOutboundMessages(ctx, mail.OutboundMessageListParams{
			ListParams: mail.ListParams{Page: page, PerPage: perPage},
			DomainID:   mailbox.DomainID,
			Status:     "draft",
			Sort:       "-updated_at",
		})
		if err != nil {
			return count, fmt.Errorf("list page %d: %w", page, err)
		}
		if len(outbound) == 0 {
			return count, nil
		}
		for _, draft := range outbound {
			if draft.InboxID != 0 && draft.InboxID != mailbox.InboxID {
				continue
			}
			if _, err := store.StoreRemoteDraft(mailbox, draft, time.Now()); err != nil {
				return count, fmt.Errorf("store remote draft %d: %w", draft.ID, err)
			}
			count++
		}
		if pagination == nil || page*pagination.PerPage >= pagination.TotalCount {
			return count, nil
		}
		page++
	}
}

func SyncOutboxForMailbox(ctx context.Context, service *mail.Service, store mailstore.Store, mailbox mailstore.MailboxMeta) ([]OutboxUpdate, error) {
	_, mailboxPath, err := store.FindMailboxByAddress(mailbox.Address)
	if err != nil {
		return nil, err
	}
	items, err := mailstore.ListOutbox(mailboxPath)
	if err != nil {
		return nil, err
	}
	updates := make([]OutboxUpdate, 0, len(items))
	for _, item := range items {
		remote, err := service.ShowOutboundMessage(ctx, item.Meta.RemoteID)
		if err != nil {
			return updates, fmt.Errorf("fetch outbound %d: %w", item.Meta.RemoteID, err)
		}
		moved, err := store.SyncOutboxItem(mailbox, remote.ID, remote.Status, remote.LastError, outboundOccurredAt(remote))
		if err != nil {
			return updates, fmt.Errorf("store outbound %d status: %w", remote.ID, err)
		}
		updates = append(updates, OutboxUpdate{RemoteID: remote.ID, Status: remote.Status, Subject: remote.Subject, Path: moved.Path})
	}
	return updates, nil
}

func SyncInboxForMailbox(ctx context.Context, service *mail.Service, store mailstore.Store, mailbox mailstore.MailboxMeta) (int, int, error) {
	count := 0
	bodyErrors := 0
	page := 1
	const perPage = 100
	for {
		messages, pagination, err := service.ListMessages(ctx, mail.MessageListParams{
			ListParams: mail.ListParams{Page: page, PerPage: perPage},
			InboxID:    mailbox.InboxID,
			Mailbox:    "inbox",
			Sort:       "-received_at",
		})
		if err != nil {
			return count, bodyErrors, fmt.Errorf("list page %d: %w", page, err)
		}
		if len(messages) == 0 {
			return count, bodyErrors, nil
		}
		for _, message := range messages {
			body, err := service.MessageBody(ctx, message.ID)
			if err != nil {
				bodyErrors++
			}
			if _, err := store.StoreInboxMessage(mailbox, message, body, time.Now()); err != nil {
				return count, bodyErrors, fmt.Errorf("store message %d: %w", message.ID, err)
			}
			count++
		}
		if pagination == nil || page*pagination.PerPage >= pagination.TotalCount {
			return count, bodyErrors, nil
		}
		page++
	}
}

func outboundOccurredAt(message *mail.OutboundMessage) time.Time {
	if message == nil {
		return time.Now()
	}
	if message.SentAt != nil {
		return *message.SentAt
	}
	if message.FailedAt != nil {
		return *message.FailedAt
	}
	if message.QueuedAt != nil {
		return *message.QueuedAt
	}
	return message.UpdatedAt
}
