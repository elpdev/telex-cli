package screens

import (
	"context"
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

func stripScreenANSI(value string) string {
	return screenANSIRE.ReplaceAllString(value, "")
}
