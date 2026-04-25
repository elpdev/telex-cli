package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/mail"
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

func TestRootSyncCommandExists(t *testing.T) {
	cmd := newRootCommand(buildInfo{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"sync", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestMailSyncCommandExists(t *testing.T) {
	cmd := newRootCommand(buildInfo{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mail", "sync", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestMailSearchCommandExists(t *testing.T) {
	cmd := newRootCommand(buildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mail", "search", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "--received-from") || !strings.Contains(got, "--subaddress") {
		t.Fatalf("help = %q", got)
	}
}

func TestInboxListCommandReadsLocalCache(t *testing.T) {
	dataDir := t.TempDir()
	store := mailstore.New(dataDir)
	mailbox := testCommandMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Cached Message",
		FromAddress: "sender@example.net",
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, &mail.MessageBody{Text: "cached body"}, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand(buildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", dataDir, "mail", "inbox", "list", "--mailbox", mailbox.Address})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Cached Message") || !strings.Contains(got, "sender@example.net") {
		t.Fatalf("output = %q", got)
	}
}

func TestInboxShowCommandHandlesMetadataOnlyCache(t *testing.T) {
	dataDir := t.TempDir()
	store := mailstore.New(dataDir)
	mailbox := testCommandMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          456,
		Subject:     "Metadata Only",
		FromAddress: "sender@example.net",
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand(buildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", dataDir, "mail", "inbox", "show", "456", "--mailbox", mailbox.Address})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Metadata Only") || !strings.Contains(got, "body not cached") {
		t.Fatalf("output = %q", got)
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
