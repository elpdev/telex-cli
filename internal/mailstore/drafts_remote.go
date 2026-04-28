package mailstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/elpdev/telex-cli/internal/mail"
)

func (s Store) StoreRemoteDraft(mailbox MailboxMeta, outbound mail.OutboundMessage, syncedAt time.Time) (*Draft, error) {
	if syncedAt.IsZero() {
		syncedAt = time.Now()
	}
	mailboxPath, err := s.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
	if err != nil {
		return nil, err
	}
	draftPath, err := findDraftPathByRemoteID(mailboxPath, outbound.ID)
	if err != nil {
		return nil, err
	}
	if draftPath == "" {
		draftPath = filepath.Join(mailboxPath, "drafts", remoteDraftID(outbound))
	}
	if err := os.MkdirAll(filepath.Join(draftPath, "attachments"), 0o700); err != nil {
		return nil, err
	}
	body := outbound.BodyText
	if body == "" {
		body = outbound.BodyHTML
	}
	if body == "" {
		body = "\n"
	}
	createdAt := outbound.CreatedAt
	if createdAt.IsZero() {
		createdAt = syncedAt
	}
	updatedAt := outbound.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = syncedAt
	}
	meta := DraftMeta{
		SchemaVersion:   SchemaVersion,
		Kind:            "draft",
		ID:              filepath.Base(draftPath),
		DomainID:        outbound.DomainID,
		DomainName:      mailbox.DomainName,
		InboxID:         outbound.InboxID,
		FromAddress:     mailbox.Address,
		RemoteID:        outbound.ID,
		SourceMessageID: outbound.SourceMessageID,
		ConversationID:  outbound.ConversationID,
		RemoteStatus:    outbound.Status,
		RemoteError:     outbound.LastError,
		DraftKind:       stringValue(outbound.Metadata["draft_kind"]),
		Subject:         outbound.Subject,
		To:              cleanStrings(outbound.ToAddresses),
		CC:              cleanStrings(outbound.CCAddresses),
		BCC:             cleanStrings(outbound.BCCAddresses),
		Attachments:     outboundAttachmentMetas(outbound.Attachments),
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
	if err := writeTOML(filepath.Join(draftPath, "meta.toml"), meta); err != nil {
		return nil, err
	}
	if err := writeFile(filepath.Join(draftPath, "body.md"), []byte(body)); err != nil {
		return nil, err
	}
	return ReadDraft(draftPath)
}

func findDraftPathByRemoteID(mailboxPath string, remoteID int64) (string, error) {
	if remoteID == 0 {
		return "", nil
	}
	drafts, err := ListDrafts(mailboxPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	for _, draft := range drafts {
		if draft.Meta.RemoteID == remoteID {
			return draft.Path, nil
		}
	}
	return "", nil
}

func remoteDraftID(outbound mail.OutboundMessage) string {
	slug := slugSubject(outbound.Subject)
	if slug == "" {
		slug = "draft"
	}
	return fmt.Sprintf("remote-%d-%s", outbound.ID, slug)
}

func outboundAttachmentMetas(attachments []mail.Attachment) []AttachmentMeta {
	metas := make([]AttachmentMeta, 0, len(attachments))
	for _, attachment := range attachments {
		metas = append(metas, AttachmentMeta{
			ID:          attachment.ID,
			Filename:    attachment.Filename,
			CacheName:   attachmentCacheName(AttachmentMeta{ID: attachment.ID, Filename: attachment.Filename}),
			ContentType: attachment.ContentType,
			ByteSize:    attachment.ByteSize,
			Previewable: attachment.Previewable,
			PreviewKind: attachment.PreviewKind,
			PreviewURL:  attachment.PreviewURL,
			DownloadURL: attachment.DownloadURL,
		})
	}
	return metas
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
