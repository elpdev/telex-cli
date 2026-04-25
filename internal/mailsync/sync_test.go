package mailsync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/api"
	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

func TestRunSyncsMailboxesOutboxAndInbox(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/mailboxes":
			_, _ = w.Write([]byte(`{"data":{"counts":{},"domains":[{"id":12,"name":"example.com","active":true}],"inboxes":[{"id":34,"domain_id":12,"address":"hello@example.com","local_part":"hello","active":true}]}}`))
		case "/api/v1/messages":
			_, _ = w.Write([]byte(`{"data":[{"id":77,"inbox_id":34,"subject":"Inbound","from_address":"sender@example.net","to_addresses":["hello@example.com"],"system_state":"inbox","received_at":"2026-04-24T13:00:00Z"}],"meta":{"page":1,"per_page":100,"total_count":1}}`))
		case "/api/v1/outbound_messages":
			_, _ = w.Write([]byte(`{"data":[{"id":99,"domain_id":12,"inbox_id":34,"status":"draft","subject":"Remote Draft","to_addresses":["to@example.net"],"body_text":"remote body","created_at":"2026-04-24T12:00:00Z","updated_at":"2026-04-24T12:30:00Z"}],"meta":{"page":1,"per_page":100,"total_count":1}}`))
		case "/api/v1/messages/77/body":
			_, _ = w.Write([]byte(`{"data":{"text":"Body"}}`))
		case "/api/v1/outbound_messages/88":
			_, _ = w.Write([]byte(`{"data":{"id":88,"status":"sent","subject":"Outbound","sent_at":"2026-04-24T15:00:00Z"}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	store := mailstore.New(t.TempDir())
	mailbox := mailstore.MailboxMeta{SchemaVersion: mailstore.SchemaVersion, DomainID: 12, DomainName: "example.com", InboxID: 34, Address: "hello@example.com", LocalPart: "hello", Active: true, SyncedAt: time.Now()}
	if err := store.CreateMailbox(mailbox); err != nil {
		t.Fatal(err)
	}
	draft, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Outbound", To: []string{"to@example.net"}, Body: "body", Now: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MoveDraftToOutbox(mailbox, draft.Meta.ID, 88, "queued", time.Date(2026, 4, 24, 11, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	service := mail.NewService(testSyncAPIClient(t, server.URL))
	result, err := Run(context.Background(), store, service, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.InboxMessages != 1 || result.OutboxItems != 1 || result.DraftItems != 1 {
		t.Fatalf("result = %#v", result)
	}
	mailboxPath, err := store.MailboxPath("example.com", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mailstore.FindInboxMessage(mailboxPath, 77); err != nil {
		t.Fatal(err)
	}
	drafts, err := mailstore.ListDrafts(mailboxPath)
	if err != nil || len(drafts) != 1 || drafts[0].Meta.RemoteID != 99 {
		t.Fatalf("drafts = %#v err=%v", drafts, err)
	}
	sent, err := mailstore.ListSent(mailboxPath)
	if err != nil || len(sent) != 1 {
		t.Fatalf("sent = %#v err=%v", sent, err)
	}
}

func testSyncAPIClient(t *testing.T, baseURL string) *api.Client {
	t.Helper()
	tokenPath := filepath.Join(t.TempDir(), "token.toml")
	if err := config.SaveTokenTo(tokenPath, &config.TokenCache{Token: "token", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	return api.NewClient(&config.Config{BaseURL: baseURL, ClientID: "id", SecretKey: "secret"}, tokenPath)
}
