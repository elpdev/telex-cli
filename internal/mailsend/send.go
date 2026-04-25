package mailsend

import (
	"context"
	"os"
	"time"

	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

type Result struct {
	DraftID  string
	RemoteID int64
	Status   string
	Path     string
}

func SendDraft(ctx context.Context, store mailstore.Store, service *mail.Service, mailbox mailstore.MailboxMeta, draft mailstore.Draft) (Result, error) {
	domainID := draft.Meta.DomainID
	inboxID := draft.Meta.InboxID
	sourceMessageID := draft.Meta.SourceMessageID
	conversationID := draft.Meta.ConversationID
	input := &mail.OutboundMessageInput{
		DomainID:        &domainID,
		InboxID:         &inboxID,
		SourceMessageID: int64Ptr(sourceMessageID),
		ConversationID:  int64Ptr(conversationID),
		ToAddresses:     draft.Meta.To,
		CCAddresses:     draft.Meta.CC,
		BCCAddresses:    draft.Meta.BCC,
		Subject:         draft.Meta.Subject,
		Body:            draft.Body,
	}
	var outbound *mail.OutboundMessage
	var err error
	if draft.Meta.RemoteID > 0 {
		outbound, err = service.UpdateOutboundMessage(ctx, draft.Meta.RemoteID, input)
	} else {
		outbound, err = service.CreateOutboundMessage(ctx, input, false)
	}
	if err != nil {
		return Result{}, err
	}
	for _, attachment := range draft.Meta.Attachments {
		path := mailstore.AttachmentCachePath(draft.Path, attachment)
		if !localFileExists(path) {
			continue
		}
		if _, err := service.AttachOutboundMessageFile(ctx, outbound.ID, path); err != nil {
			return Result{}, err
		}
	}
	sent, err := service.SendOutboundMessage(ctx, outbound.ID)
	if err != nil {
		return Result{}, err
	}
	moved, err := store.MoveDraftToOutbox(mailbox, draft.Meta.ID, sent.ID, sent.Status, time.Now())
	if err != nil {
		return Result{}, err
	}
	return Result{DraftID: moved.Meta.ID, RemoteID: sent.ID, Status: sent.Status, Path: moved.Path}, nil
}

func localFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func int64Ptr(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}
