package mailstore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/elpdev/telex-cli/internal/mail"
)

type MessageMeta struct {
	SchemaVersion  int       `toml:"schema_version"`
	Kind           string    `toml:"kind"`
	RemoteID       int64     `toml:"remote_id"`
	ConversationID int64     `toml:"conversation_id"`
	DomainID       int64     `toml:"domain_id"`
	DomainName     string    `toml:"domain_name"`
	InboxID        int64     `toml:"inbox_id"`
	Mailbox        string    `toml:"mailbox"`
	Status         string    `toml:"status"`
	RemoteError    string    `toml:"remote_error"`
	Subject        string    `toml:"subject"`
	FromAddress    string    `toml:"from_address"`
	FromName       string    `toml:"from_name"`
	To             []string  `toml:"to"`
	CC             []string  `toml:"cc"`
	Read           bool      `toml:"read"`
	Starred        bool      `toml:"starred"`
	ReceivedAt     time.Time `toml:"received_at"`
	SyncedAt       time.Time `toml:"synced_at"`
}

type CachedMessage struct {
	Meta     MessageMeta
	Path     string
	BodyText string
	BodyHTML string
}

func (s Store) StoreInboxMessage(mailbox MailboxMeta, message mail.Message, body *mail.MessageBody, syncedAt time.Time) (string, error) {
	mailboxPath, err := s.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
	if err != nil {
		return "", err
	}
	receivedAt := message.ReceivedAt
	if receivedAt.IsZero() {
		receivedAt = syncedAt
	}
	path := messageItemPath(mailboxPath, "inbox", receivedAt, message.ID, message.Subject)
	if err := os.MkdirAll(filepath.Join(path, "attachments"), 0o700); err != nil {
		return "", err
	}
	meta := MessageMeta{
		SchemaVersion:  SchemaVersion,
		Kind:           "message",
		RemoteID:       message.ID,
		ConversationID: message.ConversationID,
		DomainID:       mailbox.DomainID,
		DomainName:     mailbox.DomainName,
		InboxID:        mailbox.InboxID,
		Mailbox:        message.SystemState,
		Status:         message.Status,
		Subject:        message.Subject,
		FromAddress:    message.FromAddress,
		FromName:       message.FromName,
		To:             message.ToAddresses,
		CC:             message.CCAddresses,
		Read:           message.Read,
		Starred:        message.Starred,
		ReceivedAt:     receivedAt,
		SyncedAt:       syncedAt,
	}
	if err := writeTOML(filepath.Join(path, "meta.toml"), meta); err != nil {
		return "", err
	}
	if body != nil {
		if err := writeFile(filepath.Join(path, "body.txt"), []byte(body.Text)); err != nil {
			return "", err
		}
		if err := writeFile(filepath.Join(path, "body.html"), []byte(body.HTML)); err != nil {
			return "", err
		}
	}
	return path, nil
}

func messageItemPath(mailboxPath, box string, at time.Time, remoteID int64, subject string) string {
	slug := slugSubject(subject)
	if slug == "" {
		slug = "message"
	}
	return filepath.Join(mailboxPath, box, at.Format("2006"), at.Format("01"), at.Format("02"), fmt.Sprintf("%s-%d", slug, remoteID))
}

func ListInbox(mailboxPath string) ([]CachedMessage, error) {
	return ListMessages(mailboxPath, "inbox")
}

func ListMessages(mailboxPath, box string) ([]CachedMessage, error) {
	messages, err := listMessages(mailboxPath, box)
	if err != nil {
		return nil, err
	}
	sort.Slice(messages, func(i, j int) bool {
		if box == "inbox" && messages[i].Meta.Read != messages[j].Meta.Read {
			return !messages[i].Meta.Read
		}
		return messages[i].Meta.ReceivedAt.After(messages[j].Meta.ReceivedAt)
	})
	return messages, nil
}

func FindInboxMessage(mailboxPath string, remoteID int64) (*CachedMessage, error) {
	messages, err := ListInbox(mailboxPath)
	if err != nil {
		return nil, err
	}
	for i := range messages {
		if messages[i].Meta.RemoteID == remoteID {
			return &messages[i], nil
		}
	}
	return nil, fmt.Errorf("inbox message %d not found", remoteID)
}

func SetCachedMessageRead(path string, read bool, syncedAt time.Time) (*CachedMessage, error) {
	message, err := ReadCachedMessage(path)
	if err != nil {
		return nil, err
	}
	message.Meta.Read = read
	message.Meta.SyncedAt = syncedAt
	if err := writeTOML(filepath.Join(path, "meta.toml"), message.Meta); err != nil {
		return nil, err
	}
	return ReadCachedMessage(path)
}

func SetCachedMessageStarred(path string, starred bool, syncedAt time.Time) (*CachedMessage, error) {
	message, err := ReadCachedMessage(path)
	if err != nil {
		return nil, err
	}
	message.Meta.Starred = starred
	message.Meta.SyncedAt = syncedAt
	if err := writeTOML(filepath.Join(path, "meta.toml"), message.Meta); err != nil {
		return nil, err
	}
	return ReadCachedMessage(path)
}

func MoveCachedMessage(mailboxPath, fromBox, toBox, messagePath string, syncedAt time.Time) (*CachedMessage, error) {
	message, err := ReadCachedMessage(messagePath)
	if err != nil {
		return nil, err
	}
	fromRoot := filepath.Join(mailboxPath, fromBox)
	if rel, err := filepath.Rel(fromRoot, messagePath); err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil, fmt.Errorf("message path %q is not in %s", messagePath, fromBox)
	}
	destPath := messageItemPath(mailboxPath, toBox, message.Meta.ReceivedAt, message.Meta.RemoteID, message.Meta.Subject)
	if _, err := os.Stat(destPath); err == nil {
		return nil, fmt.Errorf("cached message already exists in %s", toBox)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
		return nil, err
	}
	if err := os.Rename(messagePath, destPath); err != nil {
		return nil, err
	}
	message.Meta.Mailbox = toBox
	message.Meta.SyncedAt = syncedAt
	if err := writeTOML(filepath.Join(destPath, "meta.toml"), message.Meta); err != nil {
		return nil, err
	}
	return ReadCachedMessage(destPath)
}

func ReadCachedMessage(path string) (*CachedMessage, error) {
	var meta MessageMeta
	if _, err := toml.DecodeFile(filepath.Join(path, "meta.toml"), &meta); err != nil {
		return nil, err
	}
	message := &CachedMessage{Meta: meta, Path: path}
	if body, err := os.ReadFile(filepath.Join(path, "body.txt")); err == nil {
		message.BodyText = string(body)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if body, err := os.ReadFile(filepath.Join(path, "body.html")); err == nil {
		message.BodyHTML = string(body)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return message, nil
}

func listMessages(mailboxPath, box string) ([]CachedMessage, error) {
	root := filepath.Join(mailboxPath, box)
	messages := []CachedMessage{}
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
		message, err := ReadCachedMessage(path)
		if err != nil {
			return err
		}
		messages = append(messages, *message)
		return filepath.SkipDir
	}); err != nil {
		return nil, err
	}
	return messages, nil
}
