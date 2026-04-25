package screens

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

var screenANSIRE = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func TestMailScreenLoadsCachedInboxAndOpensDetail(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Cached Subject",
		FromAddress: "sender@example.net",
		ToAddresses: []string{"hello@example.com"},
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, &mail.MessageBody{Text: "Cached body"}, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)

	list := screen.View(100, 20)
	if !strings.Contains(list, "hello@example.com") || !strings.Contains(list, "Cached Subject") {
		t.Fatalf("list view = %q", list)
	}
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Mail)
	detail := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(detail, "Cached body") {
		t.Fatalf("detail view = %q", detail)
	}
}

func TestMailScreenScrollsDetailView(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	body := strings.Join([]string{"line 1", "line 2", "line 3", "line 4"}, "\n")
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Scrollable",
		FromAddress: "sender@example.net",
		ToAddresses: []string{"hello@example.com"},
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, &mail.MessageBody{Text: body}, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Mail)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	screen = updated.(Mail)

	detail := stripScreenANSI(screen.View(100, 9))
	if strings.Contains(detail, "line 1") || !strings.Contains(detail, "line 2") {
		t.Fatalf("detail view = %q", detail)
	}
}

func TestMailScreenOpenHTMLStatusWhenMissing(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Metadata Only",
		FromAddress: "sender@example.net",
		ToAddresses: []string{"hello@example.com"},
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "o", Code: 'o'}))
	screen = updated.(Mail)
	if cmd != nil {
		t.Fatal("expected no command when HTML body is missing")
	}
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "No cached HTML body") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailScreenShowsLinksFromDetail(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Links",
		FromAddress: "sender@example.net",
		ToAddresses: []string{"hello@example.com"},
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, &mail.MessageBody{HTML: `<p><a href="https://example.com/read">Read more</a></p>`}, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Mail)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "L", Code: 'L'}))
	screen = updated.(Mail)

	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Links: Links") || !strings.Contains(view, "Read more") || !strings.Contains(view, "https://example.com/read") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailScreenExtractsSelectedLinkToArticleReader(t *testing.T) {
	originalExtractor := extractArticleURL
	extractArticleURL = func(ctx context.Context, url string) (string, error) {
		if url != "https://example.com/read" {
			t.Fatalf("url = %q", url)
		}
		return "# Extracted Article\n\nReadable article body", nil
	}
	t.Cleanup(func() { extractArticleURL = originalExtractor })

	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Links",
		FromAddress: "sender@example.net",
		ToAddresses: []string{"hello@example.com"},
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, &mail.MessageBody{HTML: `<p><a href="https://example.com/read">Read more</a></p>`}, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Mail)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "L", Code: 'L'}))
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "e", Code: 'e'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected extraction command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)

	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Article reader") || !strings.Contains(view, "Extracted Article") || !strings.Contains(view, "Readable article body") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailScreenTogglesReadState(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	path, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Unread",
		FromAddress: "sender@example.net",
		ToAddresses: []string{"hello@example.com"},
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var gotID int64
	var gotRead bool
	screen := NewMailWithActions(store, func(ctx context.Context, id int64, read bool) error {
		gotID = id
		gotRead = read
		return nil
	}, nil, nil, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "u", Code: 'u'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected toggle command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if gotID != 123 || !gotRead {
		t.Fatalf("toggle got id=%d read=%t", gotID, gotRead)
	}
	if !screen.messages[0].Meta.Read {
		t.Fatal("expected screen message state to be read")
	}
	readBack, err := mailstore.ReadCachedMessage(path)
	if err != nil {
		t.Fatal(err)
	}
	if !readBack.Meta.Read {
		t.Fatal("expected cached metadata to be read")
	}
}

func TestMailScreenTogglesStarState(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	path, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Unstarred",
		FromAddress: "sender@example.net",
		ToAddresses: []string{"hello@example.com"},
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var gotID int64
	var gotStarred bool
	screen := NewMailWithActions(store, nil, func(ctx context.Context, id int64, starred bool) error {
		gotID = id
		gotStarred = starred
		return nil
	}, nil, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "s", Code: 's'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected toggle command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if gotID != 123 || !gotStarred {
		t.Fatalf("toggle got id=%d starred=%t", gotID, gotStarred)
	}
	if !screen.messages[0].Meta.Starred {
		t.Fatal("expected screen message state to be starred")
	}
	readBack, err := mailstore.ReadCachedMessage(path)
	if err != nil {
		t.Fatal(err)
	}
	if !readBack.Meta.Starred {
		t.Fatal("expected cached metadata to be starred")
	}
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "!") {
		t.Fatalf("expected starred marker in view, got %q", view)
	}
}

func TestMailScreenArchivesSelectedMessage(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	path, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Archive",
		FromAddress: "sender@example.net",
		ToAddresses: []string{"hello@example.com"},
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var gotID int64
	screen := NewMailWithActions(store, nil, nil, func(ctx context.Context, id int64) error {
		gotID = id
		return nil
	}, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "a", Code: 'a'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected archive command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if gotID != 123 {
		t.Fatalf("archive got id=%d", gotID)
	}
	if len(screen.messages) != 0 {
		t.Fatalf("len(messages) = %d, want 0", len(screen.messages))
	}
	if _, err := mailstore.ReadCachedMessage(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("original cache should be moved: %v", err)
	}
	mailboxPath, err := store.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
	if err != nil {
		t.Fatal(err)
	}
	moved, err := mailstore.ReadCachedMessage(filepath.Join(mailboxPath, "archive", "2026", "04", "24", "archive-123"))
	if err != nil {
		t.Fatal(err)
	}
	if moved.Meta.Mailbox != "archive" {
		t.Fatalf("mailbox = %q, want archive", moved.Meta.Mailbox)
	}
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Archived") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailScreenSwitchesMessageBoxes(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	path, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Archived Subject",
		FromAddress: "sender@example.net",
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	mailboxPath, err := store.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mailstore.MoveCachedMessage(mailboxPath, "inbox", "archive", path, time.Date(2026, 4, 24, 15, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "]", Code: ']'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected box switch load command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Box 2/6: archive") || !strings.Contains(view, "Archived Subject") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailScreenShowsSentAndOutbox(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	queuedAt := time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)
	outboxDraft, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Queued Subject", To: []string{"to@example.net"}, Body: "queued body", Now: queuedAt.Add(-time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MoveDraftToOutbox(mailbox, outboxDraft.Meta.ID, 201, "queued", queuedAt); err != nil {
		t.Fatal(err)
	}
	sentDraft, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Sent Subject", To: []string{"to@example.net"}, Body: "sent body", Now: queuedAt.Add(-2 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MoveDraftToOutbox(mailbox, sentDraft.Meta.ID, 202, "queued", queuedAt.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SyncOutboxItem(mailbox, 202, "sent", "", queuedAt.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	for range 3 {
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "]", Code: ']'}))
		screen = updated.(Mail)
		if cmd == nil {
			t.Fatal("expected box switch load command")
		}
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	sentView := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(sentView, "Box 4/6: sent") || !strings.Contains(sentView, "Sent Subject") {
		t.Fatalf("sent view = %q", sentView)
	}
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "]", Code: ']'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected box switch load command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	outboxView := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(outboxView, "Box 5/6: outbox") || !strings.Contains(outboxView, "Queued Subject") {
		t.Fatalf("outbox view = %q", outboxView)
	}
}

func TestMailScreenSendsSelectedDraft(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	draft, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Send Draft", To: []string{"to@example.net"}, Body: "body", Now: time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	var gotDraftID string
	screen := NewMailWithActions(store, nil, nil, nil, nil, nil, nil, func(ctx context.Context, mailbox mailstore.MailboxMeta, draft mailstore.Draft) error {
		gotDraftID = draft.Meta.ID
		_, err := store.MoveDraftToOutbox(mailbox, draft.Meta.ID, 301, "queued", time.Date(2026, 4, 24, 15, 0, 0, 0, time.UTC))
		return err
	})
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	for range 5 {
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "]", Code: ']'}))
		screen = updated.(Mail)
		if cmd == nil {
			t.Fatal("expected box load command")
		}
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "S", Code: 'S'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected send command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if gotDraftID != draft.Meta.ID {
		t.Fatalf("sent draft id = %q, want %q", gotDraftID, draft.Meta.ID)
	}
	if len(screen.messages) != 0 {
		t.Fatalf("len(messages) = %d, want 0", len(screen.messages))
	}
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Draft sent") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailScreenDraftSendFailureLeavesDraft(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Send Draft", To: []string{"to@example.net"}, Body: "body", Now: time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}
	screen := NewMailWithActions(store, nil, nil, nil, nil, nil, nil, func(ctx context.Context, mailbox mailstore.MailboxMeta, draft mailstore.Draft) error {
		return errors.New("remote unavailable")
	})
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	for range 5 {
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "]", Code: ']'}))
		screen = updated.(Mail)
		if cmd == nil {
			t.Fatal("expected box load command")
		}
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "S", Code: 'S'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected send command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if len(screen.messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(screen.messages))
	}
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Could not send draft") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailScreenShowsOutboundDeliveryMetadata(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	queuedAt := time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)
	draft, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Failed Subject", To: []string{"to@example.net"}, Body: "failed body", Now: queuedAt.Add(-time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MoveDraftToOutbox(mailbox, draft.Meta.ID, 201, "queued", queuedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SyncOutboxItem(mailbox, 201, "queued", "smtp retry pending", queuedAt.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	for range 4 {
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "]", Code: ']'}))
		screen = updated.(Mail)
		if cmd == nil {
			t.Fatal("expected box switch load command")
		}
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Mail)
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Delivery status: queued") || !strings.Contains(view, "Delivery error: smtp retry pending") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailScreenSearchFiltersCurrentBox(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 1, Subject: "Alpha", FromAddress: "a@example.net", SystemState: "inbox", ReceivedAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 2, Subject: "Beta", FromAddress: "b@example.net", SystemState: "inbox", ReceivedAt: time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)}, &mail.MessageBody{Text: "needle body"}, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	for _, key := range []tea.KeyPressMsg{
		tea.KeyPressMsg(tea.Key{Text: "/", Code: '/'}),
		tea.KeyPressMsg(tea.Key{Text: "needle"}),
		tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}),
	} {
		updated, _ = screen.Update(key)
		screen = updated.(Mail)
	}
	view := stripScreenANSI(screen.View(100, 20))
	if strings.Contains(view, "Alpha") || !strings.Contains(view, "Beta") || !strings.Contains(view, "Filter: needle (1/2)") {
		t.Fatalf("view = %q", view)
	}
}

func TestParseDraftFile(t *testing.T) {
	fields, err := parseDraftFile("From: hello@example.com\nTo: a@example.net, b@example.net\nCc: c@example.net\nBcc: d@example.net\nSubject: Hello\n\nDraft body")
	if err != nil {
		t.Fatal(err)
	}
	if fields.Subject != "Hello" || len(fields.To) != 2 || fields.To[1] != "b@example.net" || fields.Body != "Draft body" {
		t.Fatalf("fields = %#v", fields)
	}
}

func TestSaveEditedDraftCreatesLocalDraft(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	file, err := os.CreateTemp(t.TempDir(), "draft-*.md")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	if _, err := file.WriteString("From: hello@example.com\nTo: a@example.net\nCc: \nBcc: \nSubject: Saved\n\nBody"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	draft, err := saveEditedDraft(store, mailbox, path, "")
	if err != nil {
		t.Fatal(err)
	}
	if draft.Meta.Subject != "Saved" || draft.Body != "Body" || draft.Meta.To[0] != "a@example.net" {
		t.Fatalf("draft = %#v", draft)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temp draft should be removed: %v", err)
	}
}

func TestSaveEditedDraftUpdatesExistingDraft(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	draft, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Old", To: []string{"old@example.net"}, Body: "old body", Now: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.CreateTemp(t.TempDir(), "draft-*.md")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	if _, err := file.WriteString("From: hello@example.com\nTo: new@example.net\nCc: cc@example.net\nBcc: \nSubject: New\nX-Telex-Source-Message-ID: 123\nX-Telex-Conversation-ID: 456\n\nnew body"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	updated, err := saveEditedDraft(store, mailbox, path, draft.Path)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Meta.ID != draft.Meta.ID || updated.Meta.Subject != "New" || updated.Meta.To[0] != "new@example.net" || updated.Meta.CC[0] != "cc@example.net" || updated.Meta.SourceMessageID != 123 || updated.Meta.ConversationID != 456 || updated.Body != "new body" {
		t.Fatalf("updated draft = %#v", updated)
	}
}

func TestDeleteDraftRemovesSelectedDraft(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Delete Me", To: []string{"to@example.net"}, Body: "body", Now: time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	for range 5 {
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "]", Code: ']'}))
		screen = updated.(Mail)
		if cmd == nil {
			t.Fatal("expected box load command")
		}
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected delete command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if len(screen.messages) != 0 {
		t.Fatalf("len(messages) = %d, want 0", len(screen.messages))
	}
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Draft deleted") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailScreenSyncsAndReloadsCurrentBox(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	synced := false
	screen := NewMailWithActions(store, nil, nil, nil, nil, nil, func(ctx context.Context) (MailSyncResult, error) {
		synced = true
		if _, err := store.StoreInboxMessage(mailbox, mail.Message{
			ID:          123,
			Subject:     "Synced Subject",
			FromAddress: "sender@example.net",
			SystemState: "inbox",
			ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
		}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
			return MailSyncResult{}, err
		}
		return MailSyncResult{InboxMessages: 1}, nil
	}, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "r", Code: 'r'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected sync command")
	}
	inProgress := stripScreenANSI(screen.View(100, 20))
	if strings.Contains(inProgress, "Loading local mail cache") || !strings.Contains(inProgress, "Syncing...") {
		t.Fatalf("in-progress view = %q", inProgress)
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if !synced {
		t.Fatal("expected sync function to run")
	}
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Synced Subject") || !strings.Contains(view, "Synced 1 inbox message(s)") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailScreenSyncFailureReloadsCache(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Cached Subject",
		FromAddress: "sender@example.net",
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	screen := NewMailWithActions(store, nil, nil, nil, nil, nil, func(ctx context.Context) (MailSyncResult, error) {
		return MailSyncResult{}, errors.New("remote unavailable")
	}, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "r", Code: 'r'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected sync command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Cached Subject") || !strings.Contains(view, "Sync failed: remote unavailable") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailScreenRestoresArchivedMessage(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	path, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Restore Me",
		FromAddress: "sender@example.net",
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	mailboxPath, err := store.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
	if err != nil {
		t.Fatal(err)
	}
	archived, err := mailstore.MoveCachedMessage(mailboxPath, "inbox", "archive", path, time.Date(2026, 4, 24, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var gotID int64
	screen := NewMailWithActions(store, nil, nil, nil, nil, func(ctx context.Context, id int64) error {
		gotID = id
		return nil
	}, nil, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "]", Code: ']'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected archive load command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	updated, cmd = screen.Update(tea.KeyPressMsg(tea.Key{Text: "R", Code: 'R'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected restore command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if gotID != 123 {
		t.Fatalf("restore got id=%d", gotID)
	}
	if len(screen.messages) != 0 {
		t.Fatalf("len(messages) = %d, want 0", len(screen.messages))
	}
	if _, err := mailstore.ReadCachedMessage(archived.Path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("archive cache should be moved: %v", err)
	}
	restored, err := mailstore.ReadCachedMessage(filepath.Join(mailboxPath, "inbox", "2026", "04", "24", "restore-me-123"))
	if err != nil {
		t.Fatal(err)
	}
	if restored.Meta.Mailbox != "inbox" {
		t.Fatalf("mailbox = %q, want inbox", restored.Meta.Mailbox)
	}
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Restored") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailScreenRestoreFailureLeavesTrashUnchanged(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	path, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Stay Trash",
		FromAddress: "sender@example.net",
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	mailboxPath, err := store.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
	if err != nil {
		t.Fatal(err)
	}
	trashed, err := mailstore.MoveCachedMessage(mailboxPath, "inbox", "trash", path, time.Date(2026, 4, 24, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	screen := NewMailWithActions(store, nil, nil, nil, nil, func(ctx context.Context, id int64) error {
		return errors.New("remote unavailable")
	}, nil, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	for range 2 {
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "]", Code: ']'}))
		screen = updated.(Mail)
		if cmd == nil {
			t.Fatal("expected box load command")
		}
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "R", Code: 'R'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected restore command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if len(screen.messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(screen.messages))
	}
	readBack, err := mailstore.ReadCachedMessage(trashed.Path)
	if err != nil {
		t.Fatal(err)
	}
	if readBack.Meta.Mailbox != "trash" {
		t.Fatalf("mailbox = %q, want trash", readBack.Meta.Mailbox)
	}
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Could not restore message") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailScreenTrashFailureLeavesCacheUnchanged(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      12,
		DomainName:    "example.com",
		InboxID:       34,
		Address:       "hello@example.com",
		LocalPart:     "hello",
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	path, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Trash",
		FromAddress: "sender@example.net",
		ToAddresses: []string{"hello@example.com"},
		SystemState: "inbox",
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	screen := NewMailWithActions(store, nil, nil, nil, func(ctx context.Context, id int64) error {
		return errors.New("remote unavailable")
	}, nil, nil, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "d", Code: 'd'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected trash command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if len(screen.messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(screen.messages))
	}
	readBack, err := mailstore.ReadCachedMessage(path)
	if err != nil {
		t.Fatal(err)
	}
	if readBack.Meta.Mailbox != "inbox" {
		t.Fatalf("mailbox = %q, want inbox", readBack.Meta.Mailbox)
	}
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Could not trash message") {
		t.Fatalf("view = %q", view)
	}
}

func stripScreenANSI(value string) string {
	return screenANSIRE.ReplaceAllString(value, "")
}
