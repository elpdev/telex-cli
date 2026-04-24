package mailstore

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/mail"
)

func TestMailboxPathUsesDomainAndLocalPart(t *testing.T) {
	store := New("/tmp/telex-data")
	path, err := store.MailboxPath("example.com", "hello")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/tmp/telex-data", "mail", "example.com", "hello")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestMailboxPathRejectsUnsafeNames(t *testing.T) {
	store := New(t.TempDir())
	if _, err := store.MailboxPath("example.com", "../hello"); err == nil {
		t.Fatal("expected unsafe mailbox name error")
	}
	if _, err := store.MailboxPath("bad/domain", "hello"); err == nil {
		t.Fatal("expected unsafe domain name error")
	}
}

func TestCreateMailboxCreatesStandardFoldersAndMetadata(t *testing.T) {
	store := New(t.TempDir())
	syncedAt := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	meta := MailboxMeta{
		DomainID:   12,
		DomainName: "example.com",
		InboxID:    34,
		Address:    "hello@example.com",
		LocalPart:  "hello",
		Active:     true,
		SyncedAt:   syncedAt,
	}
	if err := store.CreateMailbox(meta); err != nil {
		t.Fatal(err)
	}
	path, err := store.MailboxPath("example.com", "hello")
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range mailboxDirs {
		assertDir(t, filepath.Join(path, dir))
	}
	got, err := ReadMailboxMeta(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != SchemaVersion || got.DomainID != 12 || got.InboxID != 34 || got.Address != "hello@example.com" || !got.Active {
		t.Fatalf("meta = %#v", got)
	}
}

func TestSyncMailboxesCreatesOnlyActiveInboxes(t *testing.T) {
	store := New(t.TempDir())
	bootstrap := &mail.MailboxBootstrap{
		Domains: []mail.Domain{{ID: 12, Name: "example.com"}},
		Inboxes: []mail.Inbox{
			{ID: 34, DomainID: 12, Address: "hello@example.com", LocalPart: "hello", Active: true},
			{ID: 35, DomainID: 12, Address: "old@example.com", LocalPart: "old", Active: false},
		},
	}
	result, err := store.SyncMailboxes(bootstrap, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 1 || result.Created[0].LocalPart != "hello" {
		t.Fatalf("created = %#v", result.Created)
	}
	if len(result.Skipped) != 1 || result.Skipped[0].LocalPart != "old" {
		t.Fatalf("skipped = %#v", result.Skipped)
	}
	activePath, _ := store.MailboxPath("example.com", "hello")
	inactivePath, _ := store.MailboxPath("example.com", "old")
	assertDir(t, activePath)
	if _, err := os.Stat(inactivePath); !os.IsNotExist(err) {
		t.Fatalf("inactive path should not exist: %v", err)
	}
}

func assertDir(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", path)
	}
}
