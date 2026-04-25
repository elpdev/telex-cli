package mailstore

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindMailboxByAddressUsesSyncedMetadata(t *testing.T) {
	store := New(t.TempDir())
	meta := testMailboxMeta()
	if err := store.CreateMailbox(meta); err != nil {
		t.Fatal(err)
	}
	found, path, err := store.FindMailboxByAddress("hello@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if found.Address != "hello@example.com" || found.LocalPart != "hello" {
		t.Fatalf("mailbox = %#v", found)
	}
	wantPath := filepath.Join(store.MailRoot(), "example.com", "hello")
	if path != wantPath {
		t.Fatalf("path = %q, want %q", path, wantPath)
	}
}

func TestFindMailboxByAddressRejectsUnsyncedMailbox(t *testing.T) {
	store := New(t.TempDir())
	if _, _, err := store.FindMailboxByAddress("hello@example.com"); err == nil {
		t.Fatal("expected unsynced mailbox error")
	}
}

func TestCreateDraftWritesMetadataBodyAndAttachmentsFolder(t *testing.T) {
	store := New(t.TempDir())
	mailbox := testMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 24, 10, 30, 0, 0, time.UTC)
	draft, err := store.CreateDraft(DraftInput{
		Mailbox: mailbox,
		Subject: "Product Update!",
		To:      []string{" customer@example.net ", ""},
		Body:    "Hello\n",
		Now:     now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if draft.Meta.ID != "20260424-103000-product-update" {
		t.Fatalf("id = %q", draft.Meta.ID)
	}
	if draft.Meta.FromAddress != "hello@example.com" || len(draft.Meta.To) != 1 || draft.Meta.To[0] != "customer@example.net" {
		t.Fatalf("meta = %#v", draft.Meta)
	}
	assertDir(t, filepath.Join(draft.Path, "attachments"))
	read, err := ReadDraft(draft.Path)
	if err != nil {
		t.Fatal(err)
	}
	if read.Body != "Hello\n" || read.Meta.Subject != "Product Update!" {
		t.Fatalf("draft = %#v", read)
	}
}

func TestAttachFileToDraftCopiesFileAndUpdatesMetadata(t *testing.T) {
	store := New(t.TempDir())
	mailbox := testMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	draft, err := store.CreateDraft(DraftInput{Mailbox: mailbox, Subject: "Attach", Now: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(source, []byte("pdf"), 0o600); err != nil {
		t.Fatal(err)
	}
	updated, err := AttachFileToDraft(draft.Path, source, time.Date(2026, 4, 24, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Meta.Attachments) != 1 || updated.Meta.Attachments[0].Filename != "invoice.pdf" || updated.Meta.Attachments[0].CacheName != "invoice.pdf" {
		t.Fatalf("attachments = %#v", updated.Meta.Attachments)
	}
	data, err := os.ReadFile(filepath.Join(draft.Path, "attachments", "invoice.pdf"))
	if err != nil || string(data) != "pdf" {
		t.Fatalf("data = %q err=%v", string(data), err)
	}
}

func TestListDraftsNewestFirst(t *testing.T) {
	store := New(t.TempDir())
	mailbox := testMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	oldDraft, err := store.CreateDraft(DraftInput{Mailbox: mailbox, Subject: "Old", Now: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	newDraft, err := store.CreateDraft(DraftInput{Mailbox: mailbox, Subject: "New", Now: time.Date(2026, 4, 24, 11, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	mailboxPath, err := store.MailboxPath("example.com", "hello")
	if err != nil {
		t.Fatal(err)
	}
	drafts, err := ListDrafts(mailboxPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(drafts) != 2 || drafts[0].Meta.ID != newDraft.Meta.ID || drafts[1].Meta.ID != oldDraft.Meta.ID {
		t.Fatalf("drafts = %#v", drafts)
	}
}

func TestMoveDraftToOutboxUsesChronologicalRemoteIDPath(t *testing.T) {
	store := New(t.TempDir())
	mailbox := testMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	draft, err := store.CreateDraft(DraftInput{Mailbox: mailbox, Subject: "Send me", Now: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	queuedAt := time.Date(2026, 4, 24, 11, 0, 0, 0, time.UTC)
	moved, err := store.MoveDraftToOutbox(mailbox, draft.Meta.ID, 987, "queued", queuedAt)
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(store.MailRoot(), "example.com", "hello", "outbox", "2026", "04", "24", "send-me-987")
	if moved.Path != wantPath {
		t.Fatalf("path = %q, want %q", moved.Path, wantPath)
	}
	read, err := ReadDraft(moved.Path)
	if err != nil {
		t.Fatal(err)
	}
	if read.Meta.Kind != "outbox" || read.Meta.RemoteID != 987 || read.Meta.RemoteStatus != "queued" || !read.Meta.UpdatedAt.Equal(queuedAt) {
		t.Fatalf("meta = %#v", read.Meta)
	}
	if _, err := ReadDraft(draft.Path); err == nil {
		t.Fatal("expected original draft path to be moved")
	}
}

func TestSyncOutboxItemMovesSentAndFailedItemsChronologically(t *testing.T) {
	store := New(t.TempDir())
	mailbox := testMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	draft, err := store.CreateDraft(DraftInput{Mailbox: mailbox, Subject: "Send me", Now: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MoveDraftToOutbox(mailbox, draft.Meta.ID, 987, "queued", time.Date(2026, 4, 24, 11, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	sentAt := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	moved, err := store.SyncOutboxItem(mailbox, 987, "sent", "", sentAt)
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(store.MailRoot(), "example.com", "hello", "sent", "2026", "04", "25", "send-me-987")
	if moved.Path != wantPath {
		t.Fatalf("path = %q, want %q", moved.Path, wantPath)
	}
	read, err := ReadDraft(moved.Path)
	if err != nil {
		t.Fatal(err)
	}
	if read.Meta.Kind != "sent" || read.Meta.RemoteStatus != "sent" {
		t.Fatalf("meta = %#v", read.Meta)
	}
}

func testMailboxMeta() MailboxMeta {
	return MailboxMeta{
		SchemaVersion: SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
}
