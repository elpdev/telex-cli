package main

import (
	"strings"
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/mailstore"
)

func TestResolveDraftIDUsesLatestWhenRequested(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := testCommandMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Old", Now: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}
	latest, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "New", Now: time.Date(2026, 4, 24, 11, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	mailboxPath, err := store.MailboxPath("highlydisposable.com", "nunya")
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolveDraftID("nunya@highlydisposable.com", mailboxPath, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != latest.Meta.ID {
		t.Fatalf("draft id = %q, want %q", got, latest.Meta.ID)
	}
}

func TestResolveDraftIDRequiresChoiceWhenMultipleDraftsExist(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := testCommandMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Old", Now: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "New", Now: time.Date(2026, 4, 24, 11, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}
	mailboxPath, err := store.MailboxPath("highlydisposable.com", "nunya")
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolveDraftID("nunya@highlydisposable.com", mailboxPath, nil, false)
	if err == nil || !strings.Contains(err.Error(), "provide a draft ID or use --latest") {
		t.Fatalf("error = %v", err)
	}
}

func testCommandMailboxMeta() mailstore.MailboxMeta {
	return mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "highlydisposable.com",
		InboxID:       34,
		Address:       "nunya@highlydisposable.com",
		LocalPart:     "nunya",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
}
