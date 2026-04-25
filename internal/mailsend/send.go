package mailsend

import (
	"context"
	"fmt"
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
	if len(draft.Meta.Attachments) > 0 {
		return Result{}, fmt.Errorf("draft attachments are not supported by the remote API yet")
	}
	domainID := draft.Meta.DomainID
	inboxID := draft.Meta.InboxID
	sourceMessageID := draft.Meta.SourceMessageID
	conversationID := draft.Meta.ConversationID
	outbound, err := service.CreateOutboundMessage(ctx, &mail.OutboundMessageInput{
		DomainID:        &domainID,
		InboxID:         &inboxID,
		SourceMessageID: int64Ptr(sourceMessageID),
		ConversationID:  int64Ptr(conversationID),
		ToAddresses:     draft.Meta.To,
		CCAddresses:     draft.Meta.CC,
		BCCAddresses:    draft.Meta.BCC,
		Subject:         draft.Meta.Subject,
		Body:            draft.Body,
	}, false)
	if err != nil {
		return Result{}, err
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

func int64Ptr(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}
