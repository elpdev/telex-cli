package contactstore

import (
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/contacts"
)

func TestStoreContactRoundTrip(t *testing.T) {
	store := New(t.TempDir())
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	noteID := int64(11)
	contact := contacts.Contact{ID: 9, UserID: 1, ContactType: "person", Name: "Alice", DisplayName: "Alice", PrimaryEmailAddress: "alice@example.com", NoteFileID: &noteID, EmailAddresses: []contacts.ContactEmailAddress{{ID: 3, EmailAddress: "alice@example.com", Label: "work", PrimaryAddress: true}}, CreatedAt: now, UpdatedAt: now, Note: &contacts.ContactNote{ContactID: 9, StoredFileID: &noteID, Title: "Alice", Body: "Met at RubyConf."}}

	if err := store.StoreContact(contact, now); err != nil {
		t.Fatal(err)
	}
	cached, err := store.ReadContact(9)
	if err != nil {
		t.Fatal(err)
	}
	if cached.Meta.DisplayName != "Alice" || cached.Meta.PrimaryEmailAddress != "alice@example.com" || cached.Note == nil || cached.Note.Body != "Met at RubyConf." {
		t.Fatalf("cached = %#v", cached)
	}
	contacts, err := store.ListContacts()
	if err != nil {
		t.Fatal(err)
	}
	if len(contacts) != 1 || contacts[0].Meta.RemoteID != 9 {
		t.Fatalf("contacts = %#v", contacts)
	}
}

func TestStoreCommunicationsRoundTrip(t *testing.T) {
	store := New(t.TempDir())
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	if err := store.StoreContact(contacts.Contact{ID: 9, DisplayName: "Alice"}, now); err != nil {
		t.Fatal(err)
	}
	communications := []contacts.ContactCommunication{{ID: 7, ContactID: 9, Kind: "message", CommunicableType: "Message", CommunicableID: 2, OccurredAt: now, Metadata: map[string]any{"direction": "inbound"}, Communication: map[string]any{"subject": "Hi", "from_address": "alice@example.com"}, CreatedAt: now, UpdatedAt: now}}
	if err := store.StoreCommunications(9, communications); err != nil {
		t.Fatal(err)
	}
	cached, err := store.ReadContact(9)
	if err != nil {
		t.Fatal(err)
	}
	if len(cached.Communications) != 1 || cached.Communications[0].Subject != "Hi" || cached.Communications[0].Direction != "inbound" {
		t.Fatalf("communications = %#v", cached.Communications)
	}
}
