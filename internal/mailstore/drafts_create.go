package mailstore

import (
	"os"
	"path/filepath"
	"time"
)

func (s Store) CreateDraft(input DraftInput) (*Draft, error) {
	if input.Now.IsZero() {
		input.Now = time.Now()
	}
	id := draftID(input.Now, input.Subject)
	mailboxPath, err := s.MailboxPath(input.Mailbox.DomainName, input.Mailbox.LocalPart)
	if err != nil {
		return nil, err
	}
	draftPath := filepath.Join(mailboxPath, "drafts", id)
	if err := os.MkdirAll(filepath.Join(draftPath, "attachments"), 0o700); err != nil {
		return nil, err
	}
	meta := DraftMeta{
		SchemaVersion:   SchemaVersion,
		Kind:            "draft",
		ID:              id,
		DomainID:        input.Mailbox.DomainID,
		DomainName:      input.Mailbox.DomainName,
		InboxID:         input.Mailbox.InboxID,
		FromAddress:     input.Mailbox.Address,
		Subject:         input.Subject,
		DraftKind:       input.DraftKind,
		SourceMessageID: input.SourceMessageID,
		ConversationID:  input.ConversationID,
		To:              cleanStrings(input.To),
		CC:              cleanStrings(input.CC),
		BCC:             cleanStrings(input.BCC),
		CreatedAt:       input.Now,
		UpdatedAt:       input.Now,
	}
	if err := writeTOML(filepath.Join(draftPath, "meta.toml"), meta); err != nil {
		return nil, err
	}
	body := input.Body
	if body == "" {
		body = "\n"
	}
	if err := writeFile(filepath.Join(draftPath, "body.md"), []byte(body)); err != nil {
		return nil, err
	}
	return &Draft{Meta: meta, Path: draftPath, Body: body}, nil
}

func (s Store) UpdateDraft(path string, input DraftInput) (*Draft, error) {
	draft, err := ReadDraft(path)
	if err != nil {
		return nil, err
	}
	if input.Now.IsZero() {
		input.Now = time.Now()
	}
	draft.Meta.Subject = input.Subject
	draft.Meta.DraftKind = input.DraftKind
	draft.Meta.To = cleanStrings(input.To)
	draft.Meta.CC = cleanStrings(input.CC)
	draft.Meta.BCC = cleanStrings(input.BCC)
	draft.Meta.SourceMessageID = input.SourceMessageID
	draft.Meta.ConversationID = input.ConversationID
	draft.Meta.UpdatedAt = input.Now
	if err := writeTOML(filepath.Join(path, "meta.toml"), draft.Meta); err != nil {
		return nil, err
	}
	body := input.Body
	if body == "" {
		body = "\n"
	}
	if err := writeFile(filepath.Join(path, "body.md"), []byte(body)); err != nil {
		return nil, err
	}
	return ReadDraft(path)
}
