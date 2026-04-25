package mailsend

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/api"
	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

func TestSendDraftCreatesOutboundAndMovesToOutbox(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/outbound_messages":
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"id":900,"status":"draft"}}`))
		case "/api/v1/outbound_messages/900/send_message":
			_, _ = w.Write([]byte(`{"data":{"id":900,"status":"queued"}}`))
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
	draft, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Hello", To: []string{"to@example.net"}, Body: "Body", SourceMessageID: 55, ConversationID: 66, Now: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	service := mail.NewService(testAPIClient(t, server.URL))
	result, err := SendDraft(context.Background(), store, service, mailbox, *draft)
	if err != nil {
		t.Fatal(err)
	}
	if result.RemoteID != 900 || result.Status != "queued" {
		t.Fatalf("result = %#v", result)
	}
	if _, err := mailstore.ReadDraft(draft.Path); err == nil {
		t.Fatal("draft should have moved out of drafts")
	}
	if payload["outbound_message"] == nil {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestSendDraftUploadsAttachmentsBeforeQueueing(t *testing.T) {
	var uploadedName string
	var uploadedBody string
	var sent bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/outbound_messages":
			_, _ = w.Write([]byte(`{"data":{"id":900,"status":"draft"}}`))
		case "/api/v1/outbound_messages/900/attachments":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatal(err)
			}
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = file.Close() }()
			body, err := io.ReadAll(file)
			if err != nil {
				t.Fatal(err)
			}
			uploadedName = header.Filename
			uploadedBody = string(body)
			_, _ = w.Write([]byte(`{"data":[{"id":77,"filename":"invoice.pdf"}]}`))
		case "/api/v1/outbound_messages/900/send_message":
			sent = true
			_, _ = w.Write([]byte(`{"data":{"id":900,"status":"queued"}}`))
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
	draft, err := store.CreateDraft(mailstore.DraftInput{Mailbox: mailbox, Subject: "Hello", To: []string{"to@example.net"}, Body: "Body", Now: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(source, []byte("pdf"), 0o600); err != nil {
		t.Fatal(err)
	}
	draft, err = mailstore.AttachFileToDraft(draft.Path, source, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	service := mail.NewService(testAPIClient(t, server.URL))
	result, err := SendDraft(context.Background(), store, service, mailbox, *draft)
	if err != nil {
		t.Fatal(err)
	}
	if result.RemoteID != 900 || !sent {
		t.Fatalf("result=%#v sent=%t", result, sent)
	}
	if uploadedName != "invoice.pdf" || uploadedBody != "pdf" {
		t.Fatalf("uploaded name=%q body=%q", uploadedName, uploadedBody)
	}
}

func testAPIClient(t *testing.T, baseURL string) *api.Client {
	t.Helper()
	tokenPath := filepath.Join(t.TempDir(), "token.toml")
	if err := config.SaveTokenTo(tokenPath, &config.TokenCache{Token: "token", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	return api.NewClient(&config.Config{BaseURL: baseURL, ClientID: "id", SecretKey: "secret"}, tokenPath)
}
