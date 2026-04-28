package mailstore

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func (s Store) MoveDraftToOutbox(mailbox MailboxMeta, draftID string, remoteID int64, remoteStatus string, queuedAt time.Time) (*Draft, error) {
	mailboxPath, err := s.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
	if err != nil {
		return nil, err
	}
	draftPath := filepath.Join(mailboxPath, "drafts", draftID)
	draft, err := ReadDraft(draftPath)
	if err != nil {
		return nil, err
	}
	if queuedAt.IsZero() {
		queuedAt = time.Now()
	}
	draft.Meta.Kind = "outbox"
	draft.Meta.RemoteID = remoteID
	draft.Meta.RemoteStatus = remoteStatus
	draft.Meta.UpdatedAt = queuedAt
	if err := writeTOML(filepath.Join(draftPath, "meta.toml"), draft.Meta); err != nil {
		return nil, err
	}
	outboxPath := chronologicalItemPath(mailboxPath, "outbox", queuedAt, remoteID, draft.Meta.Subject)
	if err := os.MkdirAll(filepath.Dir(outboxPath), 0o700); err != nil {
		return nil, err
	}
	if err := os.Rename(draftPath, outboxPath); err != nil {
		return nil, err
	}
	draft.Path = outboxPath
	return draft, nil
}

func (s Store) SyncOutboxItem(mailbox MailboxMeta, remoteID int64, remoteStatus, remoteError string, occurredAt time.Time) (*Draft, error) {
	mailboxPath, err := s.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
	if err != nil {
		return nil, err
	}
	draft, err := findOutboxDraft(mailboxPath, remoteID)
	if err != nil {
		return nil, err
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	draft.Meta.RemoteStatus = remoteStatus
	draft.Meta.RemoteError = remoteError
	draft.Meta.UpdatedAt = occurredAt
	if remoteStatus != "sent" && remoteStatus != "failed" {
		if err := writeTOML(filepath.Join(draft.Path, "meta.toml"), draft.Meta); err != nil {
			return nil, err
		}
		return draft, nil
	}
	draft.Meta.Kind = remoteStatus
	if err := writeTOML(filepath.Join(draft.Path, "meta.toml"), draft.Meta); err != nil {
		return nil, err
	}
	targetPath := chronologicalItemPath(mailboxPath, remoteStatus, occurredAt, remoteID, draft.Meta.Subject)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		return nil, err
	}
	if err := os.Rename(draft.Path, targetPath); err != nil {
		return nil, err
	}
	draft.Path = targetPath
	return draft, nil
}

func findOutboxDraft(mailboxPath string, remoteID int64) (*Draft, error) {
	drafts, err := ListOutbox(mailboxPath)
	if err != nil {
		return nil, err
	}
	for i := range drafts {
		if drafts[i].Meta.RemoteID == remoteID {
			return &drafts[i], nil
		}
	}
	return nil, fmt.Errorf("outbox item for remote id %d not found", remoteID)
}

func chronologicalItemPath(mailboxPath, box string, at time.Time, remoteID int64, subject string) string {
	slug := slugSubject(subject)
	if slug == "" {
		slug = "outbound"
	}
	return filepath.Join(mailboxPath, box, at.Format("2006"), at.Format("01"), at.Format("02"), fmt.Sprintf("%s-%d", slug, remoteID))
}
