package mailstore

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/elpdev/telex-cli/internal/mail"
)

type DraftInput struct {
	Mailbox         MailboxMeta
	Subject         string
	To              []string
	CC              []string
	BCC             []string
	Body            string
	SourceMessageID int64
	ConversationID  int64
	Now             time.Time
}

type DraftMeta struct {
	SchemaVersion   int              `toml:"schema_version"`
	Kind            string           `toml:"kind"`
	ID              string           `toml:"id"`
	DomainID        int64            `toml:"domain_id"`
	DomainName      string           `toml:"domain_name"`
	InboxID         int64            `toml:"inbox_id"`
	FromAddress     string           `toml:"from_address"`
	RemoteID        int64            `toml:"remote_id"`
	SourceMessageID int64            `toml:"source_message_id"`
	ConversationID  int64            `toml:"conversation_id"`
	RemoteStatus    string           `toml:"remote_status"`
	RemoteError     string           `toml:"remote_error"`
	Subject         string           `toml:"subject"`
	To              []string         `toml:"to"`
	CC              []string         `toml:"cc"`
	BCC             []string         `toml:"bcc"`
	Attachments     []AttachmentMeta `toml:"attachments"`
	CreatedAt       time.Time        `toml:"created_at"`
	UpdatedAt       time.Time        `toml:"updated_at"`
}

type Draft struct {
	Meta DraftMeta
	Path string
	Body string
}

func (s Store) FindMailboxByAddress(address string) (*MailboxMeta, string, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, "", fmt.Errorf("mailbox address is required")
	}
	parts := strings.Split(address, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, "", fmt.Errorf("mailbox must be an address like hello@example.com")
	}
	path, err := s.MailboxPath(parts[1], parts[0])
	if err != nil {
		return nil, "", err
	}
	meta, err := ReadMailboxMeta(path)
	if err != nil {
		return nil, "", fmt.Errorf("mailbox %s has not been synced: %w", address, err)
	}
	if !strings.EqualFold(meta.Address, address) {
		return nil, "", fmt.Errorf("mailbox metadata address mismatch: found %s", meta.Address)
	}
	return meta, path, nil
}

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

func AttachFileToDraft(draftPath, sourcePath string, now time.Time) (*Draft, error) {
	draft, err := ReadDraft(draftPath)
	if err != nil {
		return nil, err
	}
	if draft.Meta.Kind != "draft" {
		return nil, fmt.Errorf("can only attach files to drafts, got %s", draft.Meta.Kind)
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("cannot attach directory %s", sourcePath)
	}
	if now.IsZero() {
		now = time.Now()
	}
	filename := filepath.Base(sourcePath)
	cacheName := uniqueAttachmentName(filepath.Join(draftPath, "attachments"), AttachmentMeta{Filename: filename})
	if err := copyFile(sourcePath, filepath.Join(draftPath, "attachments", cacheName)); err != nil {
		return nil, err
	}
	draft.Meta.Attachments = append(draft.Meta.Attachments, AttachmentMeta{Filename: filename, CacheName: cacheName, ByteSize: info.Size(), ContentType: "application/octet-stream"})
	draft.Meta.UpdatedAt = now
	if err := writeTOML(filepath.Join(draftPath, "meta.toml"), draft.Meta); err != nil {
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

func DetachFileFromDraft(draftPath, name string, now time.Time) (*Draft, error) {
	draft, err := ReadDraft(draftPath)
	if err != nil {
		return nil, err
	}
	if draft.Meta.Kind != "draft" {
		return nil, fmt.Errorf("can only detach files from drafts, got %s", draft.Meta.Kind)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("attachment name is required")
	}
	index := -1
	for i, attachment := range draft.Meta.Attachments {
		if attachment.CacheName == name || attachment.Filename == name || attachmentFileName(attachment) == name {
			index = i
			break
		}
	}
	if index < 0 {
		return nil, fmt.Errorf("attachment %q was not found", name)
	}
	attachment := draft.Meta.Attachments[index]
	draft.Meta.Attachments = append(draft.Meta.Attachments[:index], draft.Meta.Attachments[index+1:]...)
	if now.IsZero() {
		now = time.Now()
	}
	draft.Meta.UpdatedAt = now
	if err := os.Remove(AttachmentCachePath(draftPath, attachment)); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := writeTOML(filepath.Join(draftPath, "meta.toml"), draft.Meta); err != nil {
		return nil, err
	}
	return ReadDraft(draftPath)
}

func uniqueAttachmentName(dir string, attachment AttachmentMeta) string {
	name := attachmentCacheName(attachment)
	if name == "" {
		name = "attachment"
	}
	if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
		return name
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
			return candidate
		}
	}
}

func copyFile(sourcePath, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
		return err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()
	dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = dest.Close() }()
	_, err = io.Copy(dest, source)
	return err
}

func DeleteDraft(path string) error {
	draft, err := ReadDraft(path)
	if err != nil {
		return err
	}
	if draft.Meta.Kind != "draft" {
		return fmt.Errorf("can only delete drafts, got %s", draft.Meta.Kind)
	}
	return os.RemoveAll(path)
}

func ListDrafts(mailboxPath string) ([]Draft, error) {
	entries, err := os.ReadDir(filepath.Join(mailboxPath, "drafts"))
	if err != nil {
		return nil, err
	}
	drafts := make([]Draft, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(mailboxPath, "drafts", entry.Name())
		draft, err := ReadDraft(path)
		if err != nil {
			return nil, err
		}
		drafts = append(drafts, *draft)
	}
	sort.Slice(drafts, func(i, j int) bool { return drafts[i].Meta.CreatedAt.After(drafts[j].Meta.CreatedAt) })
	return drafts, nil
}

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

func ListOutbox(mailboxPath string) ([]Draft, error) {
	return listItems(mailboxPath, "outbox")
}

func ListSent(mailboxPath string) ([]Draft, error) {
	return listItems(mailboxPath, "sent")
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

func listItems(mailboxPath, box string) ([]Draft, error) {
	root := filepath.Join(mailboxPath, box)
	drafts := []Draft{}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() || path == root {
			return nil
		}
		if _, err := os.Stat(filepath.Join(path, "meta.toml")); err != nil {
			return nil
		}
		draft, err := ReadDraft(path)
		if err != nil {
			return err
		}
		drafts = append(drafts, *draft)
		return filepath.SkipDir
	}); err != nil {
		return nil, err
	}
	sort.Slice(drafts, func(i, j int) bool { return drafts[i].Meta.UpdatedAt.After(drafts[j].Meta.UpdatedAt) })
	return drafts, nil
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

func ReadDraft(path string) (*Draft, error) {
	var meta DraftMeta
	if _, err := toml.DecodeFile(filepath.Join(path, "meta.toml"), &meta); err != nil {
		return nil, err
	}
	body, err := os.ReadFile(filepath.Join(path, "body.md"))
	if err != nil {
		return nil, err
	}
	return &Draft{Meta: meta, Path: path, Body: string(body)}, nil
}

func draftID(now time.Time, subject string) string {
	slug := slugSubject(subject)
	if slug == "" {
		slug = "draft"
	}
	return now.UTC().Format("20060102-150405") + "-" + slug
}

func slugSubject(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(strings.TrimSpace(b.String()), "-")
}

func cleanStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
