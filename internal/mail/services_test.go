package mail

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
)

func TestListMessagesBuildsExpectedQuery(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":[],"meta":{"page":1,"per_page":25,"total_count":0}}`)}
	service := NewService(fake)
	_, _, err := service.ListMessages(context.Background(), MessageListParams{
		ListParams:   ListParams{Page: 1, PerPage: 25},
		InboxID:      42,
		Mailbox:      "inbox",
		LabelID:      7,
		Query:        "hello",
		Sender:       "billing",
		Recipient:    "finance@example.com",
		Status:       "processed",
		Subaddress:   "receipts",
		ReceivedFrom: "2026-04-09",
		ReceivedTo:   "2026-04-10",
		Sort:         "-received_at",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertQuery(t, fake.query, "page", "1")
	assertQuery(t, fake.query, "per_page", "25")
	assertQuery(t, fake.query, "inbox_id", "42")
	assertQuery(t, fake.query, "mailbox", "inbox")
	assertQuery(t, fake.query, "label_id", "7")
	assertQuery(t, fake.query, "q", "hello")
	assertQuery(t, fake.query, "sender", "billing")
	assertQuery(t, fake.query, "recipient", "finance@example.com")
	assertQuery(t, fake.query, "status", "processed")
	assertQuery(t, fake.query, "subaddress", "receipts")
	assertQuery(t, fake.query, "received_from", "2026-04-09")
	assertQuery(t, fake.query, "received_to", "2026-04-10")
	assertQuery(t, fake.query, "sort", "-received_at")
}

func TestArchiveMessageUsesActionEndpoint(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"id":99,"subject":"Archived"}}`)}
	service := NewService(fake)
	message, err := service.ArchiveMessage(context.Background(), 99)
	if err != nil {
		t.Fatal(err)
	}
	if fake.postPath != "/api/v1/messages/99/archive" {
		t.Fatalf("post path = %q", fake.postPath)
	}
	if message.ID != 99 || message.Subject != "Archived" {
		t.Fatalf("message = %#v", message)
	}
}

func TestConversationTimelineUsesTimelineEndpoint(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":[{"kind":"inbound","record_id":123,"subject":"Thread","conversation_id":99}]}`)}
	service := NewService(fake)
	entries, err := service.ConversationTimeline(context.Background(), 99)
	if err != nil {
		t.Fatal(err)
	}
	if fake.getPath != "/api/v1/conversations/99/timeline" {
		t.Fatalf("get path = %q", fake.getPath)
	}
	if len(entries) != 1 || entries[0].Kind != "inbound" || entries[0].RecordID != 123 {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestCreateOutboundMessageSendsExpectedPayload(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"id":123,"status":"draft"}}`)}
	service := NewService(fake)
	domainID := int64(12)
	inboxID := int64(34)
	message, err := service.CreateOutboundMessage(context.Background(), &OutboundMessageInput{
		DomainID:    &domainID,
		InboxID:     &inboxID,
		ToAddresses: []string{"customer@example.net"},
		Subject:     "Product update",
		Body:        "Hello",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if fake.postPath != "/api/v1/outbound_messages" {
		t.Fatalf("post path = %q", fake.postPath)
	}
	outer, ok := fake.postBody.(map[string]any)
	if !ok {
		t.Fatalf("post body = %#v", fake.postBody)
	}
	inner, ok := outer["outbound_message"].(map[string]any)
	if !ok {
		t.Fatalf("outbound_message = %#v", outer["outbound_message"])
	}
	assertJSONValue(t, inner["domain_id"], float64(12))
	assertJSONValue(t, inner["inbox_id"], float64(34))
	assertJSONValue(t, inner["subject"], "Product update")
	assertJSONValue(t, inner["body"], "Hello")
	if message.ID != 123 || message.Status != "draft" {
		t.Fatalf("message = %#v", message)
	}
}

func TestShowOutboundMessageUsesShowEndpoint(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"id":123,"status":"sent"}}`)}
	service := NewService(fake)
	message, err := service.ShowOutboundMessage(context.Background(), 123)
	if err != nil {
		t.Fatal(err)
	}
	if fake.getPath != "/api/v1/outbound_messages/123" {
		t.Fatalf("get path = %q", fake.getPath)
	}
	if message.ID != 123 || message.Status != "sent" {
		t.Fatalf("message = %#v", message)
	}
}

func TestAttachOutboundMessageFileUsesAttachmentEndpoint(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":[{"id":456,"filename":"upload.txt"}]}`)}
	service := NewService(fake)
	attachments, err := service.AttachOutboundMessageFile(context.Background(), 123, "/tmp/upload.txt")
	if err != nil {
		t.Fatal(err)
	}
	if fake.multipartPath != "/api/v1/outbound_messages/123/attachments" || fake.multipartFile != "/tmp/upload.txt" {
		t.Fatalf("multipart path=%q file=%q", fake.multipartPath, fake.multipartFile)
	}
	if len(attachments) != 1 || attachments[0].ID != 456 {
		t.Fatalf("attachments = %#v", attachments)
	}
}

func TestDeleteOutboundMessageUsesDeleteEndpoint(t *testing.T) {
	fake := &fakeClient{}
	service := NewService(fake)
	if err := service.DeleteOutboundMessage(context.Background(), 123); err != nil {
		t.Fatal(err)
	}
	if fake.deletePath != "/api/v1/outbound_messages/123" {
		t.Fatalf("delete path = %q", fake.deletePath)
	}
}

func assertQuery(t *testing.T, query url.Values, key, want string) {
	t.Helper()
	if got := query.Get(key); got != want {
		t.Fatalf("query[%s] = %q, want %q", key, got, want)
	}
}

type fakeClient struct {
	body          []byte
	query         url.Values
	getPath       string
	postPath      string
	postBody      any
	multipartPath string
	multipartFile string
	deletePath    string
}

func (f *fakeClient) Get(_ context.Context, path string, query url.Values) ([]byte, int, error) {
	f.getPath = path
	f.query = query
	return f.body, 200, nil
}

func (f *fakeClient) Post(_ context.Context, path string, body any) ([]byte, int, error) {
	f.postPath = path
	f.postBody = normalizeJSON(body)
	return f.body, 200, nil
}

func (f *fakeClient) PostMultipartFile(_ context.Context, path, _, filePath string) ([]byte, int, error) {
	f.multipartPath = path
	f.multipartFile = filePath
	return f.body, 200, nil
}

func (f *fakeClient) Patch(_ context.Context, _ string, _ any) ([]byte, int, error) {
	return f.body, 200, nil
}

func (f *fakeClient) Delete(_ context.Context, path string) (int, error) {
	f.deletePath = path
	return 204, nil
}

func normalizeJSON(value any) any {
	payload, _ := json.Marshal(value)
	var out any
	_ = json.Unmarshal(payload, &out)
	return out
}

func assertJSONValue(t *testing.T, got, want any) {
	t.Helper()
	if got != want {
		t.Fatalf("value = %#v, want %#v", got, want)
	}
}
