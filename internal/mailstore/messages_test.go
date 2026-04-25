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
