package screens

import (
	"strings"
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/contactstore"
)

func TestRenderContactDocument(t *testing.T) {
	now := time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC)
	doc := renderContactDocument(contactstore.CachedContact{
		Meta: contactstore.ContactMeta{ContactType: "person", Name: "Alice", DisplayName: "Alice", CompanyName: "Acme", Title: "Founder", Phone: "+1 555: 123", Website: "https://example.com"},
		Note: &contactstore.CachedContactNote{Meta: contactstore.ContactNoteMeta{Title: "Alice Dossier", SyncedAt: now}, Body: "Met at RubyConf."},
	})
	for _, want := range []string{"note_title: Alice Dossier", "contact_type: person", "name: Alice", "company_name: Acme", "title: Founder", "phone: \"+1 555: 123\"", "website: \"https://example.com\"", "Met at RubyConf."} {
		if !strings.Contains(doc, want) {
			t.Fatalf("document missing %q in %q", want, doc)
		}
	}
}

func TestParseContactDocument(t *testing.T) {
	input, err := parseContactDocument("---\nnote_title: Alice Dossier\ncontact_type: business\nname: Alice\ncompany_name: Acme\ntitle: Founder\nphone: +1555123\nwebsite: https://example.com\n---\n\nMet at RubyConf.", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	if !input.UpdateContact || !input.UpdateNote {
		t.Fatalf("input = %#v", input)
	}
	if input.Contact.ContactType != "business" || input.Contact.Name != "Alice" || input.Contact.CompanyName != "Acme" || input.Contact.Title != "Founder" || input.Contact.Phone != "+1555123" || input.Contact.Website != "https://example.com" {
		t.Fatalf("contact = %#v", input.Contact)
	}
	if input.Note.Title != "Alice Dossier" || input.Note.Body != "Met at RubyConf." {
		t.Fatalf("note = %#v", input.Note)
	}
}

func TestParseContactDocumentRejectsInvalidType(t *testing.T) {
	_, err := parseContactDocument("---\ncontact_type: vendor\n---\n\nBody", "Alice")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseContactDocumentFallsBackToTitleAndOmitsBlankFields(t *testing.T) {
	input, err := parseContactDocument("---\nname: \nphone: \n---\n\nBody", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	if input.UpdateContact {
		t.Fatalf("expected no contact update: %#v", input.Contact)
	}
	if input.Note.Title != "Alice" || input.Note.Body != "Body" {
		t.Fatalf("note = %#v", input.Note)
	}
}
