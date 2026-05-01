package screens

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/components/filepicker"
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

func TestMailScreenMarksUnreadMessageReadWhenDetailCloses(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := testScreenMailbox(12, 34, "example.com", "hello", "hello@example.com")
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	path, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Unread",
		FromAddress: "sender@example.net",
		ToAddresses: []string{"hello@example.com"},
		SystemState: "inbox",
		Read:        false,
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, &mail.MessageBody{Text: "Cached body"}, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var gotID int64
	var gotRead bool
	var calls int
	screen := NewMailWithActions(store, func(ctx context.Context, id int64, read bool) error {
		gotID = id
		gotRead = read
		calls++
		return nil
	}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)

	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Mail)
	if cmd != nil {
		t.Fatal("opening detail should not mark read")
	}
	if calls != 0 {
		t.Fatalf("open marked read %d time(s)", calls)
	}
	readBack, err := mailstore.ReadCachedMessage(path)
	if err != nil {
		t.Fatal(err)
	}
	if readBack.Meta.Read {
		t.Fatal("expected message to remain unread while open")
	}

	updated, cmd = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected mark-read command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if gotID != 123 || !gotRead || calls != 1 {
		t.Fatalf("mark read got id=%d read=%t calls=%d", gotID, gotRead, calls)
	}
	if !screen.messages[0].Meta.Read {
		t.Fatal("expected screen message state to be read")
	}
	readBack, err = mailstore.ReadCachedMessage(path)
	if err != nil {
		t.Fatal(err)
	}
	if !readBack.Meta.Read {
		t.Fatal("expected cached metadata to be read")
	}
}

func TestReplyMarksUnreadMessageRead(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "true")
	store := mailstore.New(t.TempDir())
	mailbox := testScreenMailbox(12, 34, "example.com", "hello", "hello@example.com")
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	path, err := store.StoreInboxMessage(mailbox, mail.Message{
		ID:          123,
		Subject:     "Unread",
		FromAddress: "sender@example.net",
		ToAddresses: []string{"hello@example.com"},
		SystemState: "inbox",
		Read:        false,
		ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
	}, &mail.MessageBody{Text: "Cached body"}, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	message, err := mailstore.ReadCachedMessage(path)
	if err != nil {
		t.Fatal(err)
	}
	var gotID int64
	var gotRead bool
	var calls int
	screen := NewMailWithActions(store, func(ctx context.Context, id int64, read bool) error {
		gotID = id
		gotRead = read
		calls++
		return nil
	}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	screen.mailboxes = []mailstore.MailboxMeta{mailbox}
	screen.messages = []mailstore.CachedMessage{*message}
	screen.allMessages = screen.messages

	updated, cmd := screen.editReplyDraft()
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected reply command")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok || len(batch) != 2 {
		t.Fatalf("reply command = %T, want two-command batch", msg)
	}
	updated, _ = screen.Update(batch[0]())
	screen = updated.(Mail)
	if gotID != 123 || !gotRead || calls != 1 {
		t.Fatalf("mark read got id=%d read=%t calls=%d", gotID, gotRead, calls)
	}
	readBack, err := mailstore.ReadCachedMessage(path)
	if err != nil {
		t.Fatal(err)
	}
	if !readBack.Meta.Read || !screen.messages[0].Meta.Read {
		t.Fatal("expected reply to mark original message read")
	}
	if msg, ok := batch[1]().(draftEditedMsg); ok {
		_ = os.Remove(msg.path)
	}
}

func TestAggregateUnreadMailLoadsAllUnreadInboxMessages(t *testing.T) {
	store := mailstore.New(t.TempDir())
	first := testScreenMailbox(12, 34, "example.com", "hello", "hello@example.com")
	second := testScreenMailbox(13, 35, "agent.test", "support", "support@agent.test")
	for _, mailbox := range []mailstore.MailboxMeta{first, second} {
		if err := store.CreateMailbox(mailbox); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)
	if _, err := store.StoreInboxMessage(first, mail.Message{ID: 1, Subject: "Read Subject", FromAddress: "read@example.net", SystemState: "inbox", Read: true, ReceivedAt: now}, nil, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(first, mail.Message{ID: 2, Subject: "First Unread", FromAddress: "first@example.net", SystemState: "inbox", Read: false, ReceivedAt: now.Add(-time.Hour)}, nil, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(second, mail.Message{ID: 3, Subject: "Second Unread", FromAddress: "second@example.net", SystemState: "inbox", Read: false, ReceivedAt: now.Add(time.Hour)}, nil, now); err != nil {
		t.Fatal(err)
	}

	screen := NewAggregateMail(store, "Unread", "inbox", true)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	view := stripScreenANSI(screen.View(100, 20))

	if !strings.Contains(view, "Mail / Unread") || !strings.Contains(view, "First Unread") || !strings.Contains(view, "Second Unread") {
		t.Fatalf("view = %q", view)
	}
	if strings.Contains(view, "Read Subject") {
		t.Fatalf("view contains read message: %q", view)
	}
}

func TestAggregateDraftsLoadsAllMailboxDrafts(t *testing.T) {
	store := mailstore.New(t.TempDir())
	first := testScreenMailbox(12, 34, "example.com", "hello", "hello@example.com")
	second := testScreenMailbox(13, 35, "agent.test", "support", "support@agent.test")
	for _, mailbox := range []mailstore.MailboxMeta{first, second} {
		if err := store.CreateMailbox(mailbox); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)
	if _, err := store.CreateDraft(mailstore.DraftInput{Mailbox: first, Subject: "First Draft", To: []string{"to@example.net"}, Body: "body", Now: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateDraft(mailstore.DraftInput{Mailbox: second, Subject: "Second Draft", To: []string{"to@example.net"}, Body: "body", Now: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}

	screen := NewAggregateMail(store, "Drafts", "drafts", false)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	view := stripScreenANSI(screen.View(100, 20))

	if !strings.Contains(view, "Mail / Drafts") || !strings.Contains(view, "First Draft") || !strings.Contains(view, "Second Draft") {
		t.Fatalf("view = %q", view)
	}
}

func TestAggregateStarredMailLoadsAllStarredMessages(t *testing.T) {
	store := mailstore.New(t.TempDir())
	first := testScreenMailbox(12, 34, "example.com", "hello", "hello@example.com")
	second := testScreenMailbox(13, 35, "agent.test", "support", "support@agent.test")
	for _, mailbox := range []mailstore.MailboxMeta{first, second} {
		if err := store.CreateMailbox(mailbox); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)
	if _, err := store.StoreInboxMessage(first, mail.Message{ID: 1, Subject: "Unstarred Subject", FromAddress: "plain@example.net", SystemState: "inbox", Starred: false, ReceivedAt: now}, nil, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(first, mail.Message{ID: 2, Subject: "First Starred", FromAddress: "first@example.net", SystemState: "inbox", Starred: true, ReceivedAt: now.Add(-time.Hour)}, nil, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(second, mail.Message{ID: 3, Subject: "Second Starred", FromAddress: "second@example.net", SystemState: "inbox", Starred: true, ReceivedAt: now.Add(time.Hour)}, nil, now); err != nil {
		t.Fatal(err)
	}

	screen := NewAggregateMail(store, "Starred", "starred", false).WithStarredOnly()
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	view := stripScreenANSI(screen.View(100, 20))

	if !strings.Contains(view, "Mail / Starred") || !strings.Contains(view, "First Starred") || !strings.Contains(view, "Second Starred") {
		t.Fatalf("view = %q", view)
	}
	if strings.Contains(view, "Unstarred Subject") {
		t.Fatalf("view contains unstarred message: %q", view)
	}
}

func TestAggregateStarredMailRemovesUnstarredMessage(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := testScreenMailbox(12, 34, "example.com", "hello", "hello@example.com")
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)
	path, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 123, Subject: "Starred", FromAddress: "sender@example.net", SystemState: "inbox", Starred: true, ReceivedAt: now}, nil, now)
	if err != nil {
		t.Fatal(err)
	}

	screen := NewAggregateMailWithActions(store, "Starred", "starred", false, nil, func(ctx context.Context, id int64, starred bool) error {
		if id != 123 || starred {
			t.Fatalf("toggle got id=%d starred=%t", id, starred)
		}
		return nil
	}, nil, nil, nil, nil, nil, nil, nil, nil, nil).WithStarredOnly()
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "s", Code: 's'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected toggle command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)

	if len(screen.messages) != 0 {
		t.Fatalf("expected message to be removed, got %d", len(screen.messages))
	}
	readBack, err := mailstore.ReadCachedMessage(path)
	if err != nil {
		t.Fatal(err)
	}
	if readBack.Meta.Starred {
		t.Fatal("expected cached metadata to be unstarred")
	}
}

func TestAggregateComposeOpensFromPicker(t *testing.T) {
	store := mailstore.New(t.TempDir())
	first := testScreenMailbox(12, 34, "example.com", "hello", "hello@example.com")
	second := testScreenMailbox(13, 35, "agent.test", "support", "support@agent.test")
	for _, mailbox := range []mailstore.MailboxMeta{first, second} {
		if err := store.CreateMailbox(mailbox); err != nil {
			t.Fatal(err)
		}
	}
	screen := NewAggregateMail(store, "Unread", "inbox", true)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "c", Code: 'c'}))
	screen = updated.(Mail)

	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Compose From") || !strings.Contains(view, "hello@example.com") || !strings.Contains(view, "support@agent.test") {
		t.Fatalf("view = %q", view)
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

	detail := stripScreenANSI(screen.View(100, 6))
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
	if !strings.Contains(view, "Mail / Links") || !strings.Contains(view, "Read more") || !strings.Contains(view, "https://example.com/read") {
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
	if !strings.Contains(view, "Mail / Article") || !strings.Contains(view, "Extracted Article") || !strings.Contains(view, "Readable article body") {
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
	}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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
	}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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
	if !strings.Contains(view, "★") {
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
	}, nil, nil, nil, nil, nil, nil, nil, nil)
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
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "}", Code: '}'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected box switch load command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "/ archive") || !strings.Contains(view, "Archived Subject") {
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
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "}", Code: '}'}))
		screen = updated.(Mail)
		if cmd == nil {
			t.Fatal("expected box switch load command")
		}
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	sentView := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(sentView, "/ sent") || !strings.Contains(sentView, "Sent Subject") {
		t.Fatalf("sent view = %q", sentView)
	}
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "}", Code: '}'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected box switch load command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	outboxView := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(outboxView, "/ outbox") || !strings.Contains(outboxView, "Queued Subject") {
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
	}, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	for range 5 {
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "}", Code: '}'}))
		screen = updated.(Mail)
		if cmd == nil {
			t.Fatal("expected box load command")
		}
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "S", Code: 'S'}))
	screen = updated.(Mail)
	if cmd != nil {
		t.Fatal("expected confirmation before send command")
	}
	updated, cmd = screen.Update(tea.KeyPressMsg(tea.Key{Text: "y", Code: 'y'}))
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
	}, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	for range 5 {
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "}", Code: '}'}))
		screen = updated.(Mail)
		if cmd == nil {
			t.Fatal("expected box load command")
		}
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "S", Code: 'S'}))
	screen = updated.(Mail)
	if cmd != nil {
		t.Fatal("expected confirmation before send command")
	}
	updated, cmd = screen.Update(tea.KeyPressMsg(tea.Key{Text: "y", Code: 'y'}))
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
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "}", Code: '}'}))
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

func TestMailScreenListViewPaginatesVisibleMessages(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := testScreenMailbox(12, 34, "example.com", "hello", "hello@example.com")
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		if _, err := store.StoreInboxMessage(mailbox, mail.Message{ID: int64(i + 1), Subject: fmt.Sprintf("Msg %d", i+1), FromAddress: "sender@example.net", SystemState: "inbox", ReceivedAt: now.Add(-time.Duration(i) * time.Hour)}, nil, now); err != nil {
			t.Fatal(err)
		}
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)

	view := stripScreenANSI(screen.View(100, 8))
	if !strings.Contains(view, "Page 1/2 · 1-3/5") || !strings.Contains(view, "Msg 1") || !strings.Contains(view, "Msg 3") || strings.Contains(view, "Msg 4") {
		t.Fatalf("page 1 view = %q", view)
	}
	for i := 0; i < 3; i++ {
		updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		screen = updated.(Mail)
	}
	view = stripScreenANSI(screen.View(100, 8))
	if !strings.Contains(view, "Page 2/2 · 4-5/5") || !strings.Contains(view, "Msg 4") || !strings.Contains(view, "Msg 5") || strings.Contains(view, "Msg 1") {
		t.Fatalf("page 2 view = %q", view)
	}
}

func TestMailScreenSearchFiltersMessagesOutsideVisiblePage(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := testScreenMailbox(12, 34, "example.com", "hello", "hello@example.com")
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		if _, err := store.StoreInboxMessage(mailbox, mail.Message{ID: int64(i + 1), Subject: fmt.Sprintf("Msg %d", i+1), FromAddress: "sender@example.net", SystemState: "inbox", ReceivedAt: now.Add(-time.Duration(i) * time.Hour)}, nil, now); err != nil {
			t.Fatal(err)
		}
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	initial := stripScreenANSI(screen.View(100, 8))
	if strings.Contains(initial, "Msg 5") {
		t.Fatalf("expected Msg 5 to start outside first page: %q", initial)
	}
	for _, key := range []tea.KeyPressMsg{
		tea.KeyPressMsg(tea.Key{Text: "/", Code: '/'}),
		tea.KeyPressMsg(tea.Key{Text: "Msg 5"}),
		tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}),
	} {
		updated, _ = screen.Update(key)
		screen = updated.(Mail)
	}
	view := stripScreenANSI(screen.View(100, 8))
	if !strings.Contains(view, "Msg 5") || !strings.Contains(view, "Filter: Msg 5 (1/5)") || !strings.Contains(view, "Page 1/1 · 1-1/1") {
		t.Fatalf("filtered view = %q", view)
	}
}

func TestMailScreenRemoteSearchShowsTransientResults(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 1, Subject: "Cached", FromAddress: "cache@example.net", SystemState: "inbox", ReceivedAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	var gotParams MailSearchParams
	screen := NewMailWithActions(store, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, func(ctx context.Context, params MailSearchParams) ([]mailstore.CachedMessage, error) {
		gotParams = params
		return []mailstore.CachedMessage{{
			Meta: mailstore.MessageMeta{SchemaVersion: mailstore.SchemaVersion, Kind: "remote-message", RemoteID: 99, InboxID: params.InboxID, Mailbox: params.Mailbox, Subject: "Remote Invoice", FromAddress: "billing@example.net", ReceivedAt: time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)},
			Path: "remote:99",
		}}, nil
	})
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	for _, key := range []tea.KeyPressMsg{
		tea.KeyPressMsg(tea.Key{Code: 'f', Mod: tea.ModCtrl}),
		tea.KeyPressMsg(tea.Key{Text: "invoice"}),
		tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}),
	} {
		updated, cmd := screen.Update(key)
		screen = updated.(Mail)
		if cmd != nil {
			updated, _ = screen.Update(cmd())
			screen = updated.(Mail)
		}
	}
	if gotParams.Query != "invoice" || gotParams.InboxID != mailbox.InboxID || gotParams.Mailbox != "inbox" {
		t.Fatalf("params = %#v", gotParams)
	}
	view := stripScreenANSI(screen.View(100, 20))
	if strings.Contains(view, "Cached") || !strings.Contains(view, "Remote Invoice") || !strings.Contains(view, "Remote results: invoice") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailScreenConversationOpensFromListAndNavigates(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 1, ConversationID: 77, Subject: "Thread", FromAddress: "a@example.net", SystemState: "inbox", ReceivedAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store).WithConversationActions(func(ctx context.Context, id int64) ([]ConversationEntry, error) {
		if id != 77 {
			t.Fatalf("conversation id = %d", id)
		}
		return []ConversationEntry{
			{Kind: "inbound", RecordID: 1, ConversationID: 77, Subject: "First", Sender: "a@example.net", Summary: "first summary", OccurredAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)},
			{Kind: "outbound", RecordID: 2, ConversationID: 77, Subject: "Second", Sender: "hello@example.com", Recipients: []string{"a@example.net"}, Summary: "second summary", Status: "sent", OccurredAt: time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)},
		}, nil
	}, func(ctx context.Context, entry ConversationEntry) (string, error) {
		return entry.Subject + " body", nil
	})
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "T", Code: 'T'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected conversation load command")
	}
	updated, cmd = screen.Update(cmd())
	screen = updated.(Mail)
	if cmd != nil {
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Conversation 77") || !strings.Contains(view, "First body") {
		t.Fatalf("view = %q", view)
	}
	updated, cmd = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	screen = updated.(Mail)
	if cmd != nil {
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	view = stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Second body") {
		t.Fatalf("view after tab = %q", view)
	}
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}))
	screen = updated.(Mail)
	view = stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "First body") {
		t.Fatalf("view after shift-tab = %q", view)
	}
}

func TestMailScreenConversationOpensFromDetail(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 1, ConversationID: 88, Subject: "Detail Thread", FromAddress: "a@example.net", SystemState: "inbox", ReceivedAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store).WithConversationActions(func(ctx context.Context, id int64) ([]ConversationEntry, error) {
		return []ConversationEntry{{Kind: "inbound", RecordID: 1, ConversationID: id, Subject: "Detail Thread", Sender: "a@example.net", Summary: "summary", OccurredAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)}}, nil
	}, func(ctx context.Context, entry ConversationEntry) (string, error) { return "detail thread body", nil })
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "T", Code: 'T'}))
	screen = updated.(Mail)
	updated, cmd = screen.Update(cmd())
	screen = updated.(Mail)
	if cmd != nil {
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "detail thread body") {
		t.Fatalf("view = %q", view)
	}
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	screen = updated.(Mail)
	if screen.mode != mailModeDetail {
		t.Fatalf("mode = %v, want detail", screen.mode)
	}
}

func TestMailScreenShowsAndCachesAttachment(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	path, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 1, Subject: "With Attachment", FromAddress: "a@example.net", SystemState: "inbox", Attachments: []mail.Attachment{{ID: 9, Filename: "invoice.pdf", ContentType: "application/pdf", ByteSize: 2048, DownloadURL: "/attachments/9"}}, ReceivedAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	screen := NewMailWithActions(store, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, func(ctx context.Context, attachment mailstore.AttachmentMeta) ([]byte, error) {
		if attachment.ID != 9 {
			t.Fatalf("attachment id = %d", attachment.ID)
		}
		return []byte("pdf data"), nil
	})
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Mail)
	detail := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(detail, "Attachments: 1") {
		t.Fatalf("detail = %q", detail)
	}
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "A", Code: 'A'}))
	screen = updated.(Mail)
	attachments := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(attachments, "invoice.pdf") || !strings.Contains(attachments, "2.0 KB") {
		t.Fatalf("attachments = %q", attachments)
	}
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected attachment download command")
	}
	msg := cmd().(attachmentDownloadedMsg)
	if msg.err != nil || !msg.open {
		t.Fatalf("download msg = %#v", msg)
	}
	cached := filepath.Join(path, "attachments", "9-invoice.pdf")
	if msg.path != cached {
		t.Fatalf("path = %q, want %q", msg.path, cached)
	}
	if data, err := os.ReadFile(cached); err != nil || string(data) != "pdf data" {
		t.Fatalf("cached data = %q err=%v", string(data), err)
	}
}

func TestMailScreenSavesAttachmentToDirectory(t *testing.T) {
	store := mailstore.New(t.TempDir())
	dir := t.TempDir()
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 1, Subject: "With Attachment", FromAddress: "a@example.net", SystemState: "inbox", Attachments: []mail.Attachment{{ID: 9, Filename: "invoice.pdf", ContentType: "application/pdf", ByteSize: 2048, DownloadURL: "/attachments/9"}}, ReceivedAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "9-invoice.pdf"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	screen := NewMailWithActions(store, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, func(ctx context.Context, attachment mailstore.AttachmentMeta) ([]byte, error) {
		return []byte("new"), nil
	})
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Mail)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "A", Code: 'A'}))
	screen = updated.(Mail)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "S", Code: 'S'}))
	screen = updated.(Mail)
	screen.saveDirInput = dir
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected save command")
	}
	msg := cmd().(attachmentDownloadedMsg)
	if msg.err != nil || msg.open {
		t.Fatalf("save msg = %#v", msg)
	}
	if msg.path != filepath.Join(dir, "9-invoice-2.pdf") {
		t.Fatalf("path = %q", msg.path)
	}
}

func TestMailScreenConfirmsDraftAttachmentDetach(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	draft, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Draft", To: []string{"to@example.net"}, Body: "body", Now: time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(source, []byte("pdf"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := mailstore.AttachFileToDraft(draft.Path, source, time.Now()); err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	for range 5 {
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "}", Code: '}'}))
		screen = updated.(Mail)
		if cmd == nil {
			t.Fatal("expected box load command")
		}
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Mail)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "A", Code: 'A'}))
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
	screen = updated.(Mail)
	if cmd != nil {
		t.Fatal("expected confirmation before detach command")
	}
	if !strings.Contains(stripScreenANSI(screen.View(100, 20)), "Detach this attachment") {
		t.Fatalf("view = %q", stripScreenANSI(screen.View(100, 20)))
	}
	updated, cmd = screen.Update(tea.KeyPressMsg(tea.Key{Text: "y", Code: 'y'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected detach command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	readBack, err := mailstore.ReadDraft(draft.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(readBack.Meta.Attachments) != 0 {
		t.Fatalf("attachments = %#v", readBack.Meta.Attachments)
	}
}

func TestMailScreenAttachPickerSelectionAddsAttachmentToDraft(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	draft, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Draft", To: []string{"to@example.net"}, Body: "body", Now: time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "report.txt")
	if err := os.WriteFile(source, []byte("report"), 0o600); err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	for range 5 {
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "}", Code: '}'}))
		screen = updated.(Mail)
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "a", Code: 'a'}))
	screen = updated.(Mail)
	if !screen.filePickerActive {
		t.Fatal("expected attachment picker to be active")
	}
	screen.filePicker = filepicker.New("", sourceDir, filepicker.ModeOpenFile)
	updated, _ = screen.Update(screen.filePicker.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected attach command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	readBack, err := mailstore.ReadDraft(draft.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(readBack.Meta.Attachments) != 1 || readBack.Meta.Attachments[0].Filename != "report.txt" {
		t.Fatalf("attachments = %#v", readBack.Meta.Attachments)
	}
	if !strings.Contains(stripScreenANSI(screen.View(100, 20)), "Draft saved") {
		t.Fatalf("view = %q", stripScreenANSI(screen.View(100, 20)))
	}
}

func TestMailScreenAttachPickerCancelDoesNotModifyDraft(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	draft, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Draft", To: []string{"to@example.net"}, Body: "body", Now: time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	screen := NewMail(store)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	for range 5 {
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "}", Code: '}'}))
		screen = updated.(Mail)
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "a", Code: 'a'}))
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "esc"}))
	screen = updated.(Mail)
	if cmd != nil || screen.filePickerActive {
		t.Fatalf("cmd = %v active = %v", cmd, screen.filePickerActive)
	}
	readBack, err := mailstore.ReadDraft(draft.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(readBack.Meta.Attachments) != 0 {
		t.Fatalf("attachments = %#v", readBack.Meta.Attachments)
	}
}

func TestParseDraftFile(t *testing.T) {
	fields, err := parseDraftFile("From: hello@example.com\nTo: a@example.net, b@example.net\nCc: c@example.net\nBcc: d@example.net\nSubject: Hello\nX-Telex-Draft-Kind: forward\n\nDraft body")
	if err != nil {
		t.Fatal(err)
	}
	if fields.Subject != "Hello" || len(fields.To) != 2 || fields.To[1] != "b@example.net" || fields.DraftKind != "forward" || fields.Body != "Draft body" {
		t.Fatalf("fields = %#v", fields)
	}
}

func TestQuotedForwardBodyIncludesOriginalHeadersAndBody(t *testing.T) {
	message := mailstore.CachedMessage{
		Meta: mailstore.MessageMeta{
			Subject:     "Launch",
			FromName:    "Sender",
			FromAddress: "sender@example.net",
			To:          []string{"hello@example.com"},
			CC:          []string{"copy@example.com"},
			ReceivedAt:  time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC),
		},
		BodyText: "Original body",
	}
	body := quotedForwardBody(message)
	for _, want := range []string{"---------- Forwarded message ---------", "From: Sender <sender@example.net>", "To: hello@example.com", "Cc: copy@example.com", "Subject: Launch", "Original body"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %q", want, body)
		}
	}
}

func TestQuotedReplyBodyConvertsHTMLOnlyBody(t *testing.T) {
	message := mailstore.CachedMessage{BodyHTML: `<html><body><p>First line</p><p>Second line</p></body></html>`}

	body := quotedReplyBody(message)

	if !strings.Contains(body, "> First line") || !strings.Contains(body, "> Second line") {
		t.Fatalf("body = %q", body)
	}
	if strings.Contains(body, "<p>") || strings.Contains(body, "<html") {
		t.Fatalf("body contains raw HTML: %q", body)
	}
}

func TestQuotedForwardBodyConvertsHTMLStoredAsTextBody(t *testing.T) {
	message := mailstore.CachedMessage{
		Meta:     mailstore.MessageMeta{Subject: "Launch", FromAddress: "sender@example.net"},
		BodyText: `<div><p>First line</p><p>Second line</p></div>`,
	}

	body := quotedForwardBody(message)

	if !strings.Contains(body, "First line") || !strings.Contains(body, "Second line") {
		t.Fatalf("body = %q", body)
	}
	if strings.Contains(body, "<p>") || strings.Contains(body, "<div") {
		t.Fatalf("body contains raw HTML: %q", body)
	}
}

func TestMailScreenCreatesRemoteForwardDraft(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	var gotID int64
	var gotDraft mailstore.Draft
	screen := NewMailWithActions(store, nil, nil, nil, nil, nil, nil, nil, nil, nil, func(ctx context.Context, id int64, draft mailstore.Draft) (int64, string, error) {
		gotID = id
		gotDraft = draft
		return 900, "draft", nil
	}, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	file, err := os.CreateTemp(t.TempDir(), "draft-*.md")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	_, err = file.WriteString(draftTemplate(draftFields{From: mailbox.Address, To: []string{"team@example.com"}, Subject: "Fwd: Forward Me", Body: "reviewed body", SourceMessageID: 123, DraftKind: "forward"}))
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	updated, cmd := screen.Update(draftEditedMsg{path: path, mailbox: mailbox})
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected reviewed forward command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if gotID != 123 || len(gotDraft.Meta.To) != 1 || gotDraft.Meta.To[0] != "team@example.com" || gotDraft.Meta.DraftKind != "forward" {
		t.Fatalf("forward got id=%d draft=%#v", gotID, gotDraft.Meta)
	}
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Forward draft created remotely: 900 (draft)") {
		t.Fatalf("view = %q", view)
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
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "}", Code: '}'}))
		screen = updated.(Mail)
		if cmd == nil {
			t.Fatal("expected box load command")
		}
		updated, _ = screen.Update(cmd())
		screen = updated.(Mail)
	}
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
	screen = updated.(Mail)
	if cmd != nil {
		t.Fatal("expected confirmation before delete command")
	}
	updated, cmd = screen.Update(tea.KeyPressMsg(tea.Key{Text: "y", Code: 'y'}))
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

func TestDeleteDraftSkipsRemoteDeleteForLocalForwardWithStaleRemoteID(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	draft, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Fwd: Delete Me", To: []string{"to@example.net"}, Body: "body", SourceMessageID: 21, DraftKind: "forward", Now: time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	metaPath := filepath.Join(draft.Path, "meta.toml")
	meta, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	staleMeta := strings.Replace(string(meta), "remote_id = 0", "remote_id = 21", 1)
	if staleMeta == string(meta) {
		t.Fatalf("draft metadata did not contain zero remote_id: %s", meta)
	}
	if err := os.WriteFile(metaPath, []byte(staleMeta), 0o600); err != nil {
		t.Fatal(err)
	}

	remoteDeleteCalled := false
	screen := NewMailWithActions(store, nil, nil, nil, nil, nil, nil, nil, nil, func(ctx context.Context, draft mailstore.Draft) error {
		remoteDeleteCalled = true
		return nil
	}, nil, nil)
	screen.scope = MailScope{Box: "drafts", Aggregate: true}
	screen.messages = []mailstore.CachedMessage{{Path: draft.Path}}

	updated, cmd := screen.deleteSelectedDraft()
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected delete command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if remoteDeleteCalled {
		t.Fatal("remote delete should not be called for stale local forward draft")
	}
	if _, err := mailstore.ReadDraft(draft.Path); err == nil {
		t.Fatal("expected local draft to be deleted")
	}
	if screen.status != "Draft deleted" {
		t.Fatalf("status = %q", screen.status)
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
	}, nil, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "r", Code: 'r'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected sync command")
	}
	inProgress := stripScreenANSI(screen.View(100, 20))
	if strings.Contains(inProgress, "Loading local mail cache") || !strings.Contains(inProgress, "Syncing mailboxes, outbox, and inbox...") {
		t.Fatalf("in-progress view = %q", inProgress)
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if !synced {
		t.Fatal("expected sync function to run")
	}
	view := stripScreenANSI(screen.View(100, 20))
	if !strings.Contains(view, "Synced Subject") || !strings.Contains(view, "Synced 0 mailbox(es), 1 inbox message(s)") {
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
	}, nil, nil, nil, nil, nil)
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
	}, nil, nil, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "}", Code: '}'}))
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
	}, nil, nil, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	for range 2 {
		updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "}", Code: '}'}))
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
	}, nil, nil, nil, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "d", Code: 'd'}))
	screen = updated.(Mail)
	if cmd != nil {
		t.Fatal("expected confirmation before trash command")
	}
	updated, cmd = screen.Update(tea.KeyPressMsg(tea.Key{Text: "y", Code: 'y'}))
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

func TestMailScreenMarksSelectedMessageAsJunk(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	path, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 123, Subject: "Spam", FromAddress: "sender@example.net", SystemState: "inbox", ReceivedAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var gotID int64
	screen := NewMail(store).WithJunkActions(func(ctx context.Context, id int64) error {
		gotID = id
		return nil
	}, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "J", Code: 'J'}))
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected junk command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if gotID != 123 {
		t.Fatalf("junk got id=%d", gotID)
	}
	if len(screen.messages) != 0 {
		t.Fatalf("len(messages) = %d, want 0", len(screen.messages))
	}
	if _, err := mailstore.ReadCachedMessage(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("inbox cache should be moved: %v", err)
	}
	mailboxPath, err := store.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
	if err != nil {
		t.Fatal(err)
	}
	moved, err := mailstore.ReadCachedMessage(filepath.Join(mailboxPath, "junk", "2026", "04", "24", "spam-123"))
	if err != nil {
		t.Fatal(err)
	}
	if moved.Meta.Mailbox != "junk" {
		t.Fatalf("mailbox = %q, want junk", moved.Meta.Mailbox)
	}
}

func TestMailScreenUpdatesSenderPolicy(t *testing.T) {
	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	path, err := store.StoreInboxMessage(mailbox, mail.Message{ID: 123, Subject: "Policy", FromAddress: "sender@example.net", SystemState: "inbox", ReceivedAt: time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)}, nil, time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var gotID int64
	screen := NewMail(store).WithSenderPolicyActions(func(ctx context.Context, id int64) error {
		gotID = id
		return nil
	}, nil, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.Init()())
	screen = updated.(Mail)
	updated, cmd := screen.Update(MailActionMsg{Action: "block-sender"})
	screen = updated.(Mail)
	if cmd == nil {
		t.Fatal("expected policy command")
	}
	updated, _ = screen.Update(cmd())
	screen = updated.(Mail)
	if gotID != 123 || !screen.messages[0].Meta.SenderBlocked {
		t.Fatalf("got id=%d state=%#v", gotID, screen.messages[0].Meta)
	}
	readBack, err := mailstore.ReadCachedMessage(path)
	if err != nil {
		t.Fatal(err)
	}
	if !readBack.Meta.SenderBlocked {
		t.Fatal("expected cached sender block")
	}
}

func stripScreenANSI(value string) string {
	return screenANSIRE.ReplaceAllString(value, "")
}

func testScreenMailbox(domainID, inboxID int64, domainName, localPart, address string) mailstore.MailboxMeta {
	return mailstore.MailboxMeta{
		SchemaVersion: mailstore.SchemaVersion,
		DomainID:      domainID,
		DomainName:    domainName,
		InboxID:       inboxID,
		Address:       address,
		LocalPart:     localPart,
		Active:        true,
		SyncedAt:      time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
	}
}
