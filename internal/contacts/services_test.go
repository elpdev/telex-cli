package contacts

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
)

func TestListContactsBuildsQueryAndDecodesPagination(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":[{"id":9,"display_name":"Alice"}],"meta":{"page":2,"per_page":50,"total_count":1}}`)}
	service := NewService(fake)
	contacts, pagination, err := service.ListContacts(context.Background(), ListContactsParams{ListParams: ListParams{Page: 2, PerPage: 50}, ContactType: "person", Query: "alice", Sort: "name"})
	if err != nil {
		t.Fatal(err)
	}
	assertQuery(t, fake.query, "page", "2")
	assertQuery(t, fake.query, "per_page", "50")
	assertQuery(t, fake.query, "contact_type", "person")
	assertQuery(t, fake.query, "q", "alice")
	assertQuery(t, fake.query, "sort", "name")
	if fake.getPath != "/api/v1/contacts" || len(contacts) != 1 || contacts[0].DisplayName != "Alice" {
		t.Fatalf("path=%q contacts=%#v", fake.getPath, contacts)
	}
	if pagination == nil || pagination.Page != 2 || pagination.PerPage != 50 || pagination.TotalCount != 1 {
		t.Fatalf("pagination = %#v", pagination)
	}
}

func TestShowContactCanIncludeNote(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"id":9,"display_name":"Alice","note":{"contact_id":9,"title":"Alice","body":"Met"}}}`)}
	service := NewService(fake)
	contact, err := service.ShowContact(context.Background(), 9, true)
	if err != nil {
		t.Fatal(err)
	}
	if fake.getPath != "/api/v1/contacts/9" || fake.query.Get("include_note") != "true" || contact.Note == nil || contact.Note.Body != "Met" {
		t.Fatalf("path=%q query=%v contact=%#v", fake.getPath, fake.query, contact)
	}
}

func TestCreateContactUsesContactPayload(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"id":9,"display_name":"Alice"}}`)}
	service := NewService(fake)
	primary := true
	contact, err := service.CreateContact(context.Background(), ContactInput{ContactType: "person", Name: "Alice", EmailAddresses: []ContactEmailAddressInput{{EmailAddress: "alice@example.com", Label: "work", PrimaryAddress: &primary}}, Metadata: map[string]any{"source": "cli"}})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"contact": map[string]any{"contact_type": "person", "name": "Alice", "email_addresses": []any{map[string]any{"email_address": "alice@example.com", "label": "work", "primary_address": true}}, "metadata": map[string]any{"source": "cli"}}}
	if fake.postPath != "/api/v1/contacts" || !jsonEqual(fake.postBody, want) || contact.ID != 9 {
		t.Fatalf("path=%q body=%#v contact=%#v", fake.postPath, fake.postBody, contact)
	}
}

func TestUpdateContactUsesPatchPayload(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"id":9,"title":"Founder"}}`)}
	service := NewService(fake)
	_, err := service.UpdateContact(context.Background(), 9, ContactInput{Title: "Founder"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"contact": map[string]any{"title": "Founder"}}
	if fake.patchPath != "/api/v1/contacts/9" || !jsonEqual(fake.patchBody, want) {
		t.Fatalf("path=%q body=%#v", fake.patchPath, fake.patchBody)
	}
}

func TestDeleteContactUsesDeleteEndpoint(t *testing.T) {
	fake := &fakeClient{}
	service := NewService(fake)
	if err := service.DeleteContact(context.Background(), 9); err != nil {
		t.Fatal(err)
	}
	if fake.deletePath != "/api/v1/contacts/9" {
		t.Fatalf("deletePath = %q", fake.deletePath)
	}
}

func TestContactNoteEndpoints(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"contact_id":9,"title":"Alice","body":"Met"}}`)}
	service := NewService(fake)
	note, err := service.ContactNote(context.Background(), 9)
	if err != nil {
		t.Fatal(err)
	}
	if fake.getPath != "/api/v1/contacts/9/note" || note.Body != "Met" {
		t.Fatalf("path=%q note=%#v", fake.getPath, note)
	}
	_, err = service.UpdateContactNote(context.Background(), 9, ContactNoteInput{Title: "Alice", Body: "Updated"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"note": map[string]any{"title": "Alice", "body": "Updated"}}
	if fake.putPath != "/api/v1/contacts/9/note" || !jsonEqual(fake.putBody, want) {
		t.Fatalf("path=%q body=%#v", fake.putPath, fake.putBody)
	}
}

func TestContactCommunicationsUsesEndpoint(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":[{"id":3,"contact_id":9,"kind":"message","communication":{"subject":"Hi"}}],"meta":{"page":1,"per_page":20,"total_count":1}}`)}
	service := NewService(fake)
	communications, pagination, err := service.ContactCommunications(context.Background(), 9, ListParams{Page: 1, PerPage: 20})
	if err != nil {
		t.Fatal(err)
	}
	if fake.getPath != "/api/v1/contacts/9/communications" || len(communications) != 1 || communications[0].Communication["subject"] != "Hi" || pagination == nil {
		t.Fatalf("path=%q communications=%#v pagination=%#v", fake.getPath, communications, pagination)
	}
}

type fakeClient struct {
	body       []byte
	query      url.Values
	getPath    string
	postPath   string
	postBody   any
	putPath    string
	putBody    any
	patchPath  string
	patchBody  any
	deletePath string
}

func (f *fakeClient) Get(_ context.Context, path string, query url.Values) ([]byte, int, error) {
	f.getPath = path
	f.query = query
	return f.body, 200, nil
}

func (f *fakeClient) Post(_ context.Context, path string, body any) ([]byte, int, error) {
	f.postPath = path
	f.postBody = normalizeJSON(body)
	return f.body, 201, nil
}

func (f *fakeClient) Put(_ context.Context, path string, body any) ([]byte, int, error) {
	f.putPath = path
	f.putBody = normalizeJSON(body)
	return f.body, 200, nil
}

func (f *fakeClient) Patch(_ context.Context, path string, body any) ([]byte, int, error) {
	f.patchPath = path
	f.patchBody = normalizeJSON(body)
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

func jsonEqual(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}

func assertQuery(t *testing.T, query url.Values, key, want string) {
	t.Helper()
	if got := query.Get(key); got != want {
		t.Fatalf("query[%s] = %q, want %q", key, got, want)
	}
}
