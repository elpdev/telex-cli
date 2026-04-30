package screens

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/contacts"
	"github.com/elpdev/telex-cli/internal/contactstore"
	"github.com/elpdev/telex-cli/internal/frontmatter"
)

var errInvalidContactType = errors.New("contact_type must be person or business")

var contactDocumentFieldOrder = []string{"note_title", "contact_type", "name", "company_name", "title", "phone", "website"}

type contactDocumentInput struct {
	Contact       contacts.ContactInput
	Note          contacts.ContactNoteInput
	UpdateContact bool
	UpdateNote    bool
}

func editContactDocumentTemplate(contact contactstore.CachedContact) (contactDocumentInput, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("telex-contact-note-%d.md", time.Now().UnixNano()))
	if err := os.WriteFile(path, []byte(renderContactDocument(contact)), 0o600); err != nil {
		return contactDocumentInput{}, err
	}
	editor := contactsEditorCommand()
	if editor == "" {
		_ = os.Remove(path)
		return contactDocumentInput{}, fmt.Errorf("set TELEX_CONTACTS_EDITOR, TELEX_NOTES_EDITOR, VISUAL, or EDITOR to edit contacts")
	}
	if err := runEditorCommand(editor, path, "set TELEX_CONTACTS_EDITOR, TELEX_NOTES_EDITOR, VISUAL, or EDITOR to edit contacts"); err != nil {
		return contactDocumentInput{}, fmt.Errorf("%w; edited file kept at %s", err, path)
	}
	content, err := readEditedFile(path)
	if err != nil {
		return contactDocumentInput{}, err
	}
	return parseContactDocument(string(content), contact.Meta.DisplayName)
}

func contactsEditorCommand() string {
	if editor := strings.TrimSpace(os.Getenv("TELEX_CONTACTS_EDITOR")); editor != "" {
		return editor
	}
	if editor := strings.TrimSpace(os.Getenv("TELEX_NOTES_EDITOR")); editor != "" {
		return editor
	}
	if editor := strings.TrimSpace(os.Getenv("VISUAL")); editor != "" {
		return editor
	}
	return strings.TrimSpace(os.Getenv("EDITOR"))
}

func parseContactNoteTemplate(content string) contacts.ContactNoteInput {
	if !strings.HasPrefix(strings.ReplaceAll(content, "\r\n", "\n"), "---\n") {
		lines := strings.Split(content, "\n")
		title := defaultTitle
		start := 0
		if len(lines) > 0 && strings.HasPrefix(strings.ToLower(lines[0]), "title:") {
			if parsed := strings.TrimSpace(lines[0][len("Title:"):]); parsed != "" {
				title = parsed
			}
			start = 1
			if len(lines) > 1 && strings.TrimSpace(lines[1]) == "" {
				start = 2
			}
		}
		return contacts.ContactNoteInput{Title: title, Body: strings.Join(lines[start:], "\n")}
	}
	input, err := parseContactDocument(content, defaultTitle)
	if err != nil {
		return contacts.ContactNoteInput{Title: defaultTitle, Body: content}
	}
	return input.Note
}

func renderContactDocument(contact contactstore.CachedContact) string {
	noteTitle := contact.Meta.DisplayName
	body := ""
	if contact.Note != nil {
		noteTitle = contact.Note.Meta.Title
		body = contact.Note.Body
	}
	fields := map[string]string{
		"note_title":   noteTitle,
		"contact_type": contact.Meta.ContactType,
		"name":         contact.Meta.Name,
		"company_name": contact.Meta.CompanyName,
		"title":        contact.Meta.Title,
		"phone":        contact.Meta.Phone,
		"website":      contact.Meta.Website,
	}
	return frontmatter.RenderWithOrder(fields, contactDocumentFieldOrder, body)
}

func parseContactDocument(content, fallbackTitle string) (contactDocumentInput, error) {
	doc, err := frontmatter.Parse(content)
	if err != nil {
		return contactDocumentInput{}, err
	}
	noteTitle := strings.TrimSpace(doc.Fields["note_title"])
	if noteTitle == "" {
		noteTitle = strings.TrimSpace(fallbackTitle)
	}
	if noteTitle == "" {
		noteTitle = defaultTitle
	}
	contactType := strings.TrimSpace(doc.Fields["contact_type"])
	if contactType != "" && contactType != "person" && contactType != "business" {
		return contactDocumentInput{}, errInvalidContactType
	}
	contactInput := contacts.ContactInput{
		ContactType: strings.TrimSpace(doc.Fields["contact_type"]),
		Name:        strings.TrimSpace(doc.Fields["name"]),
		CompanyName: strings.TrimSpace(doc.Fields["company_name"]),
		Title:       strings.TrimSpace(doc.Fields["title"]),
		Phone:       strings.TrimSpace(doc.Fields["phone"]),
		Website:     strings.TrimSpace(doc.Fields["website"]),
	}
	return contactDocumentInput{
		Contact:       contactInput,
		Note:          contacts.ContactNoteInput{Title: noteTitle, Body: doc.Body},
		UpdateContact: contactInput.ContactType != "" || contactInput.Name != "" || contactInput.CompanyName != "" || contactInput.Title != "" || contactInput.Phone != "" || contactInput.Website != "",
		UpdateNote:    true,
	}, nil
}
