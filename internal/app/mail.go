package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/mailsend"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/mailsync"
	"github.com/elpdev/telex-cli/internal/screens"
)

var errMailSyncAlreadyRunning = errors.New("mail sync already running")

type aggregateMailScreen struct {
	id         string
	title      string
	box        string
	unreadOnly bool
}

func aggregateMailScreens() []aggregateMailScreen {
	return []aggregateMailScreen{
		{id: "mail-unread", title: "Unread", box: "inbox", unreadOnly: true},
		{id: "mail-inbox", title: "Inbox", box: "inbox"},
		{id: "mail-sent", title: "Sent", box: "sent"},
		{id: "mail-drafts", title: "Drafts", box: "drafts"},
		{id: "mail-outbox", title: "Outbox", box: "outbox"},
		{id: "mail-junk", title: "Junk", box: "junk"},
		{id: "mail-archive", title: "Archive", box: "archive"},
		{id: "mail-trash", title: "Trash", box: "trash"},
	}
}

func (m *Model) buildMailScreen() screens.Mail {
	return screens.NewMailWithActions(mailstore.New(m.dataPath), m.toggleMessageRead, m.toggleMessageStar, m.archiveMessage, m.trashMessage, m.restoreMessage, m.syncMail, m.sendDraft, m.updateDraft, m.deleteDraft, m.forwardMessage, m.downloadAttachment, m.searchMail).WithConversationActions(m.conversationTimeline, m.conversationBody).WithJunkActions(m.junkMessage, m.notJunkMessage).WithSenderPolicyActions(m.blockSender, m.unblockSender, m.blockDomain, m.unblockDomain, m.trustSender, m.untrustSender)
}

func (m *Model) buildAggregateMailScreen(scope aggregateMailScreen) screens.Mail {
	return screens.NewAggregateMailWithActions(mailstore.New(m.dataPath), scope.title, scope.box, scope.unreadOnly, m.toggleMessageRead, m.toggleMessageStar, m.archiveMessage, m.trashMessage, m.restoreMessage, m.syncMail, m.sendDraft, m.updateDraft, m.deleteDraft, m.forwardMessage, m.downloadAttachment, m.searchMail).WithConversationActions(m.conversationTimeline, m.conversationBody).WithJunkActions(m.junkMessage, m.notJunkMessage).WithSenderPolicyActions(m.blockSender, m.unblockSender, m.blockDomain, m.unblockDomain, m.trustSender, m.untrustSender)
}

func (m *Model) toggleMessageStar(ctx context.Context, id int64, starred bool) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	if starred {
		_, err = service.StarMessage(ctx, id)
	} else {
		_, err = service.UnstarMessage(ctx, id)
	}
	return err
}

func (m *Model) toggleMessageRead(ctx context.Context, id int64, read bool) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	if read {
		_, err = service.MarkMessageRead(ctx, id)
	} else {
		_, err = service.MarkMessageUnread(ctx, id)
	}
	return err
}

func (m *Model) archiveMessage(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.ArchiveMessage(ctx, id)
	return err
}

func (m *Model) trashMessage(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.TrashMessage(ctx, id)
	return err
}

func (m *Model) junkMessage(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.JunkMessage(ctx, id)
	return err
}

func (m *Model) notJunkMessage(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.NotJunkMessage(ctx, id)
	return err
}

func (m *Model) restoreMessage(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.RestoreMessage(ctx, id)
	return err
}

func (m *Model) blockSender(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.BlockSender(ctx, id)
	return err
}

func (m *Model) unblockSender(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.UnblockSender(ctx, id)
	return err
}

func (m *Model) blockDomain(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.BlockDomain(ctx, id)
	return err
}

func (m *Model) unblockDomain(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.UnblockDomain(ctx, id)
	return err
}

func (m *Model) trustSender(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.TrustSender(ctx, id)
	return err
}

func (m *Model) untrustSender(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.UntrustSender(ctx, id)
	return err
}

func (m *Model) syncMail(ctx context.Context) (screens.MailSyncResult, error) {
	if !m.tryStartMailSync() {
		return screens.MailSyncResult{}, errMailSyncAlreadyRunning
	}
	defer m.finishMailSync()

	service, err := m.mailService()
	if err != nil {
		return screens.MailSyncResult{}, err
	}
	result, err := mailsync.Run(ctx, mailstore.New(m.dataPath), service, "")
	return screens.MailSyncResult{
		ActiveMailboxes:  result.ActiveMailboxes,
		SkippedMailboxes: result.SkippedMailboxes,
		OutboxItems:      result.OutboxItems,
		DraftItems:       result.DraftItems,
		InboxMessages:    result.InboxMessages,
		BodyErrors:       result.BodyErrors,
		InboxErrors:      result.InboxErrors,
	}, err
}

func (m *Model) tryStartMailSync() bool {
	if m.syncState == nil {
		return true
	}
	m.syncState.mu.Lock()
	defer m.syncState.mu.Unlock()
	if m.syncState.mailSyncing {
		return false
	}
	m.syncState.mailSyncing = true
	return true
}

func (m *Model) finishMailSync() {
	if m.syncState == nil {
		return
	}
	m.syncState.mu.Lock()
	m.syncState.mailSyncing = false
	m.syncState.mu.Unlock()
}

func (m *Model) sendDraft(ctx context.Context, mailbox mailstore.MailboxMeta, draft mailstore.Draft) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = mailsend.SendDraft(ctx, mailstore.New(m.dataPath), service, mailbox, draft)
	return err
}

func (m *Model) updateDraft(ctx context.Context, draft mailstore.Draft) error {
	if draft.Meta.RemoteID == 0 {
		return nil
	}
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.UpdateOutboundMessage(ctx, draft.Meta.RemoteID, outboundInputFromDraft(draft))
	return err
}

func (m *Model) deleteDraft(ctx context.Context, draft mailstore.Draft) error {
	if draft.Meta.RemoteID == 0 {
		return nil
	}
	service, err := m.mailService()
	if err != nil {
		return err
	}
	return service.DeleteOutboundMessage(ctx, draft.Meta.RemoteID)
}

func outboundInputFromDraft(draft mailstore.Draft) *mail.OutboundMessageInput {
	domainID := draft.Meta.DomainID
	inboxID := draft.Meta.InboxID
	return &mail.OutboundMessageInput{
		DomainID:        &domainID,
		InboxID:         &inboxID,
		SourceMessageID: int64Ptr(draft.Meta.SourceMessageID),
		ConversationID:  int64Ptr(draft.Meta.ConversationID),
		ToAddresses:     draft.Meta.To,
		CCAddresses:     draft.Meta.CC,
		BCCAddresses:    draft.Meta.BCC,
		Subject:         draft.Meta.Subject,
		Body:            draft.Body,
	}
}

func int64Ptr(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

func (m *Model) forwardMessage(ctx context.Context, id int64, draft mailstore.Draft) (int64, string, error) {
	service, err := m.mailService()
	if err != nil {
		return 0, "", err
	}
	outbound, err := service.Forward(ctx, id, draft.Meta.To)
	if err != nil {
		return 0, "", err
	}
	outbound, err = service.UpdateOutboundMessage(ctx, outbound.ID, outboundInputFromDraft(draft))
	if err != nil {
		return 0, "", err
	}
	store := mailstore.New(m.dataPath)
	mailboxes, _ := store.ListMailboxes()
	for _, mailbox := range mailboxes {
		if mailbox.InboxID == outbound.InboxID || mailbox.DomainID == outbound.DomainID {
			_, _ = store.StoreRemoteDraft(mailbox, *outbound, time.Now())
			break
		}
	}
	return outbound.ID, outbound.Status, nil
}

func (m *Model) downloadAttachment(ctx context.Context, attachment mailstore.AttachmentMeta) ([]byte, error) {
	if attachment.DownloadURL == "" {
		return nil, fmt.Errorf("attachment has no download URL")
	}
	if _, err := m.mailService(); err != nil {
		return nil, err
	}
	body, _, err := m.client.Download(ctx, attachment.DownloadURL)
	return body, err
}

func (m *Model) searchMail(ctx context.Context, params screens.MailSearchParams) ([]mailstore.CachedMessage, error) {
	service, err := m.mailService()
	if err != nil {
		return nil, err
	}
	messages, _, err := service.ListMessages(ctx, mail.MessageListParams{
		ListParams: mail.ListParams{Page: params.Page, PerPage: params.PerPage},
		InboxID:    params.InboxID,
		Mailbox:    params.Mailbox,
		Query:      params.Query,
		Sort:       params.Sort,
	})
	if err != nil {
		return nil, err
	}
	results := make([]mailstore.CachedMessage, 0, len(messages))
	for _, message := range messages {
		results = append(results, cachedRemoteMessage(message))
	}
	return results, nil
}

func (m *Model) conversationTimeline(ctx context.Context, id int64) ([]screens.ConversationEntry, error) {
	service, err := m.mailService()
	if err != nil {
		return nil, err
	}
	entries, err := service.ConversationTimeline(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]screens.ConversationEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, screens.ConversationEntry{
			Kind:           entry.Kind,
			RecordID:       entry.RecordID,
			OccurredAt:     entry.OccurredAt,
			Sender:         entry.Sender,
			Recipients:     entry.Recipients,
			Summary:        entry.Summary,
			Status:         entry.Status,
			Subject:        entry.Subject,
			ConversationID: entry.ConversationID,
		})
	}
	return out, nil
}

func (m *Model) conversationBody(ctx context.Context, entry screens.ConversationEntry) (string, error) {
	service, err := m.mailService()
	if err != nil {
		return "", err
	}
	if entry.Kind == "outbound" {
		message, err := service.ShowOutboundMessage(ctx, entry.RecordID)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(message.BodyText) != "" {
			return message.BodyText, nil
		}
		return message.BodyHTML, nil
	}
	body, err := service.MessageBody(ctx, entry.RecordID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(body.Text) != "" {
		return body.Text, nil
	}
	return body.HTML, nil
}
