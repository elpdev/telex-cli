package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/config"
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

func TestConversationsTimelineCommandExists(t *testing.T) {
	cmd := newRootCommand(buildInfo{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mail", "conversations", "timeline", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestLabelsCommandsExist(t *testing.T) {
	for _, args := range [][]string{
		{"mail", "labels", "list", "--help"},
		{"mail", "messages", "labels", "123", "--help"},
	} {
		cmd := newRootCommand(buildInfo{})
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
}

func TestSenderReputationCommandsExist(t *testing.T) {
	for _, args := range [][]string{
		{"mail", "messages", "junk", "123", "--help"},
		{"mail", "messages", "not-junk", "123", "--help"},
		{"mail", "messages", "block-sender", "123", "--help"},
		{"mail", "messages", "unblock-sender", "123", "--help"},
		{"mail", "messages", "block-domain", "123", "--help"},
		{"mail", "messages", "unblock-domain", "123", "--help"},
		{"mail", "messages", "trust-sender", "123", "--help"},
		{"mail", "messages", "untrust-sender", "123", "--help"},
	} {
		cmd := newRootCommand(buildInfo{})
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
}

func TestUpdatedLabelIDsAddsAndRemoves(t *testing.T) {
	got := updatedLabelIDs([]mail.Label{{ID: 3}, {ID: 1}}, []int64{2, 3}, []int64{1})
	want := []int64{2, 3}
	if len(got) != len(want) {
		t.Fatalf("ids = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids = %#v, want %#v", got, want)
		}
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

func TestDraftDeleteCommandDeletesSyncedRemoteDraft(t *testing.T) {
	var deleted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/outbound_messages/900" || r.Method != http.MethodDelete {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := (&config.Config{BaseURL: server.URL, ClientID: "id", SecretKey: "secret"}).SaveTo(configPath); err != nil {
		t.Fatal(err)
	}
	_, tokenPath := config.Paths(configPath)
	if err := config.SaveTokenTo(tokenPath, &config.TokenCache{Token: "token", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}

	dataDir := t.TempDir()
	store := mailstore.New(dataDir)
	mailbox := testCommandMailboxMeta()
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	draft, err := store.StoreRemoteDraft(mailbox, mail.OutboundMessage{ID: 900, DomainID: mailbox.DomainID, InboxID: mailbox.InboxID, Status: "draft", Subject: "Remote", ToAddresses: []string{"to@example.net"}, BodyText: "Body", CreatedAt: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)}, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	cmd := newRootCommand(buildInfo{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--config", configPath, "--data-dir", dataDir, "mail", "drafts", "delete", draft.Meta.ID, "--mailbox", mailbox.Address})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected remote draft delete request")
	}
	if _, err := mailstore.ReadDraft(draft.Path); err == nil {
		t.Fatal("expected local synced draft cache to be deleted")
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
