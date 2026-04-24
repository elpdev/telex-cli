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
		ListParams: ListParams{Page: 1, PerPage: 25},
		InboxID:    42,
		Mailbox:    "inbox",
		Query:      "hello",
		Sort:       "-received_at",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertQuery(t, fake.query, "page", "1")
	assertQuery(t, fake.query, "per_page", "25")
	assertQuery(t, fake.query, "inbox_id", "42")
	assertQuery(t, fake.query, "mailbox", "inbox")
	assertQuery(t, fake.query, "q", "hello")
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

func assertQuery(t *testing.T, query url.Values, key, want string) {
	t.Helper()
	if got := query.Get(key); got != want {
		t.Fatalf("query[%s] = %q, want %q", key, got, want)
	}
}

type fakeClient struct {
	body     []byte
	query    url.Values
	getPath  string
	postPath string
	postBody any
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

func (f *fakeClient) Patch(_ context.Context, _ string, _ any) ([]byte, int, error) {
	return f.body, 200, nil
}

func (f *fakeClient) Delete(_ context.Context, _ string) (int, error) {
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
