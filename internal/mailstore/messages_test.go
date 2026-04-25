package mailstore

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/mail"
)

func TestStoreInboxMessageWritesChronologicalMessageCache(t *testing.T) {
	store := New(t.TempDir())
	mailbox := testMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	receivedAt := time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)
	path, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:             123,
		ConversationID: 456,
		Subject:        "Hello Inbox",
		FromAddress:    "sender@example.net",
		ToAddresses:    []string{"hello@example.com"},
		SystemState:    "inbox",
		Read:           true,
		Attachments:    []mail.Attachment{{ID: 55, Filename: "invoice.pdf", ContentType: "application/pdf", ByteSize: 2048, DownloadURL: "/download/55"}},
		ReceivedAt:     receivedAt,
	}, &mail.MessageBody{Text: "Plain text", HTML: "<p>Plain text</p>"}, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(store.MailRoot(), "example.com", "hello", "inbox", "2026", "04", "24", "hello-inbox-123")
	if path != wantPath {
		t.Fatalf("path = %q, want %q", path, wantPath)
	}
	assertFile(t, filepath.Join(path, "meta.toml"))
	assertFile(t, filepath.Join(path, "body.txt"))
	assertFile(t, filepath.Join(path, "body.html"))
	assertDir(t, filepath.Join(path, "attachments"))
	readBack, err := ReadCachedMessage(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(readBack.Meta.Attachments) != 1 || readBack.Meta.Attachments[0].Filename != "invoice.pdf" {
		t.Fatalf("attachments = %#v", readBack.Meta.Attachments)
	}
}

func TestListInboxReturnsNewestFirstAndReadsBodies(t *testing.T) {
	store := New(t.TempDir())
	mailbox := testMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	older := time.Date(2026, 4, 23, 13, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 1, Subject: "Older", FromAddress: "old@example.net", SystemState: "inbox", ReceivedAt: older}, &mail.MessageBody{Text: "older body"}, newer); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 2, Subject: "Newer", FromAddress: "new@example.net", SystemState: "inbox", ReceivedAt: newer}, &mail.MessageBody{Text: "newer body"}, newer); err != nil {
		t.Fatal(err)
	}
	mailboxPath, err := store.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
	if err != nil {
		t.Fatal(err)
	}
	messages, err := ListInbox(mailboxPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(messages))
	}
	if messages[0].Meta.RemoteID != 2 || messages[0].BodyText != "newer body" {
		t.Fatalf("newest message = %#v", messages[0])
	}
	if messages[1].Meta.RemoteID != 1 || messages[1].BodyText != "older body" {
		t.Fatalf("oldest message = %#v", messages[1])
	}
}

func TestListInboxReturnsUnreadBeforeRead(t *testing.T) {
	store := New(t.TempDir())
	mailbox := testMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	olderUnread := time.Date(2026, 4, 23, 13, 0, 0, 0, time.UTC)
	newerRead := time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 1, Subject: "New Read", FromAddress: "read@example.net", SystemState: "inbox", Read: true, ReceivedAt: newerRead}, nil, newerRead); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 2, Subject: "Old Unread", FromAddress: "unread@example.net", SystemState: "inbox", Read: false, ReceivedAt: olderUnread}, nil, newerRead); err != nil {
		t.Fatal(err)
	}
	mailboxPath, err := store.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
	if err != nil {
		t.Fatal(err)
	}
	messages, err := ListInbox(mailboxPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(messages))
	}
	if messages[0].Meta.RemoteID != 2 || messages[1].Meta.RemoteID != 1 {
		t.Fatalf("message order = %d, %d; want unread before read", messages[0].Meta.RemoteID, messages[1].Meta.RemoteID)
	}
}

func TestFindInboxMessageAllowsMissingBody(t *testing.T) {
	store := New(t.TempDir())
	mailbox := testMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 7, Subject: "Metadata Only", FromAddress: "sender@example.net", SystemState: "inbox", ReceivedAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	mailboxPath, err := store.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
	if err != nil {
		t.Fatal(err)
	}
	message, err := FindInboxMessage(mailboxPath, 7)
	if err != nil {
		t.Fatal(err)
	}
	if message.BodyText != "" || message.BodyHTML != "" {
		t.Fatalf("body = %q/%q, want empty", message.BodyText, message.BodyHTML)
	}
}

func TestSetCachedMessageReadUpdatesMetadata(t *testing.T) {
	store := New(t.TempDir())
	mailbox := testMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	path, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 9, Subject: "Unread", FromAddress: "sender@example.net", SystemState: "inbox", ReceivedAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	message, err := SetCachedMessageRead(path, true, time.Date(2026, 4, 24, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !message.Meta.Read {
		t.Fatal("expected message to be read")
	}
	readBack, err := ReadCachedMessage(path)
	if err != nil {
		t.Fatal(err)
	}
	if !readBack.Meta.Read {
		t.Fatal("expected persisted metadata to be read")
	}
}

func TestSetCachedMessageStarredUpdatesMetadata(t *testing.T) {
	store := New(t.TempDir())
	mailbox := testMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	path, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 10, Subject: "Star", FromAddress: "sender@example.net", SystemState: "inbox", ReceivedAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	message, err := SetCachedMessageStarred(path, true, time.Date(2026, 4, 24, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !message.Meta.Starred {
		t.Fatal("expected message to be starred")
	}
	readBack, err := ReadCachedMessage(path)
	if err != nil {
		t.Fatal(err)
	}
	if !readBack.Meta.Starred {
		t.Fatal("expected persisted metadata to be starred")
	}
}

func TestMoveCachedMessageMovesFolderAndUpdatesMetadata(t *testing.T) {
	store := New(t.TempDir())
	mailbox := testMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	receivedAt := time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)
	path, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 11, Subject: "Archive Me", FromAddress: "sender@example.net", SystemState: "inbox", ReceivedAt: receivedAt}, &mail.MessageBody{Text: "body", HTML: "<p>body</p>"}, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	mailboxPath, err := store.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
	if err != nil {
		t.Fatal(err)
	}
	moved, err := MoveCachedMessage(mailboxPath, "inbox", "archive", path, time.Date(2026, 4, 24, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(mailboxPath, "archive", "2026", "04", "24", "archive-me-11")
	if moved.Path != wantPath {
		t.Fatalf("path = %q, want %q", moved.Path, wantPath)
	}
	if moved.Meta.Mailbox != "archive" {
		t.Fatalf("mailbox = %q, want archive", moved.Meta.Mailbox)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("original path should not exist: %v", err)
	}
	assertFile(t, filepath.Join(wantPath, "body.txt"))
	assertFile(t, filepath.Join(wantPath, "body.html"))
}

func TestUpdateCachedMessageByRemoteIDUpdatesLabelsAndSenderPolicy(t *testing.T) {
	store := New(t.TempDir())
	mailbox := testMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 12, Subject: "Policy", FromAddress: "sender@example.net", SystemState: "inbox", ReceivedAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	updated, err := store.UpdateCachedMessageByRemoteID(12, mail.Message{ID: 12, Subject: "Policy", FromAddress: "sender@example.net", SystemState: "inbox", SenderTrusted: true, Labels: []mail.Label{{ID: 3, Name: "Billing", Color: "#ff0"}}, ReceivedAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)}, time.Date(2026, 4, 24, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Meta.SenderTrusted || len(updated.Meta.Labels) != 1 || updated.Meta.Labels[0].Name != "Billing" {
		t.Fatalf("updated = %#v", updated.Meta)
	}
}

func assertFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDir() {
		t.Fatalf("%s is a directory", path)
	}
}
