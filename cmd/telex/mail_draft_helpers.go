package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

func resolveDraftID(mailboxAddress, mailboxPath string, args []string, latest bool) (string, error) {
	if len(args) > 0 && latest {
		return "", fmt.Errorf("provide either a draft ID or --latest, not both")
	}
	if len(args) > 0 {
		return args[0], nil
	}
	drafts, err := mailstore.ListDrafts(mailboxPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("no drafts exist for %s", mailboxAddress)
		}
		return "", err
	}
	if len(drafts) == 0 {
		return "", fmt.Errorf("no drafts exist for %s", mailboxAddress)
	}
	if latest || len(drafts) == 1 {
		return drafts[0].Meta.ID, nil
	}
	ids := make([]string, 0, len(drafts))
	for _, draft := range drafts {
		ids = append(ids, draft.Meta.ID)
	}
	return "", fmt.Errorf("multiple drafts exist for %s; provide a draft ID or use --latest. Available drafts: %s", mailboxAddress, strings.Join(ids, ", "))
}

func draftNotFoundError(mailboxAddress, draftID, mailboxPath string) error {
	drafts, err := mailstore.ListDrafts(mailboxPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("draft %q was not found for %s; no drafts exist yet", draftID, mailboxAddress)
		}
		return fmt.Errorf("draft %q was not found for %s; listing drafts failed: %w", draftID, mailboxAddress, err)
	}
	if len(drafts) == 0 {
		return fmt.Errorf("draft %q was not found for %s; no drafts exist yet", draftID, mailboxAddress)
	}
	ids := make([]string, 0, len(drafts))
	for _, draft := range drafts {
		ids = append(ids, draft.Meta.ID)
	}
	return fmt.Errorf("draft %q was not found for %s; available drafts: %s", draftID, mailboxAddress, strings.Join(ids, ", "))
}

func draftRows(drafts []mailstore.Draft) [][]string {
	rows := make([][]string, 0, len(drafts))
	for _, draft := range drafts {
		rows = append(rows, []string{
			strconv.FormatInt(draft.Meta.RemoteID, 10),
			draft.Meta.RemoteStatus,
			draft.Meta.Subject,
			strings.Join(draft.Meta.To, ", "),
			draft.Meta.UpdatedAt.Format("2006-01-02 15:04"),
			draft.Path,
		})
	}
	return rows
}

func draftFields(draft mailstore.Draft) [][]string {
	rows := [][]string{
		{"id", draft.Meta.ID},
		{"from", draft.Meta.FromAddress},
		{"to", strings.Join(draft.Meta.To, ", ")},
		{"cc", strings.Join(draft.Meta.CC, ", ")},
		{"bcc", strings.Join(draft.Meta.BCC, ", ")},
		{"subject", draft.Meta.Subject},
		{"updated_at", draft.Meta.UpdatedAt.Format("2006-01-02 15:04")},
		{"attachments", strconv.Itoa(len(draft.Meta.Attachments))},
		{"path", draft.Path},
	}
	if draft.Meta.SourceMessageID > 0 {
		rows = append(rows, []string{"source_message_id", strconv.FormatInt(draft.Meta.SourceMessageID, 10)})
	}
	if draft.Meta.RemoteID > 0 {
		rows = append(rows, []string{"remote_id", strconv.FormatInt(draft.Meta.RemoteID, 10)})
	}
	if draft.Meta.ConversationID > 0 {
		rows = append(rows, []string{"conversation_id", strconv.FormatInt(draft.Meta.ConversationID, 10)})
	}
	return rows
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
