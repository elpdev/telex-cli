package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/contacts"
)

func editContactNoteTemplate(title, body string) (contacts.ContactNoteInput, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("telex-contact-note-%d.md", time.Now().UnixNano()))
	if err := os.WriteFile(path, []byte(fmt.Sprintf("Title: %s\n\n%s", title, body)), 0o600); err != nil {
		return contacts.ContactNoteInput{}, err
	}
	editor := contactsEditorCommand()
	if editor == "" {
		_ = os.Remove(path)
		return contacts.ContactNoteInput{}, fmt.Errorf("set TELEX_CONTACTS_EDITOR, TELEX_NOTES_EDITOR, VISUAL, or EDITOR to edit contact notes")
	}
	if err := runEditorCommand(editor, path, "set TELEX_CONTACTS_EDITOR, TELEX_NOTES_EDITOR, VISUAL, or EDITOR to edit contact notes"); err != nil {
		return contacts.ContactNoteInput{}, fmt.Errorf("%w; edited file kept at %s", err, path)
	}
	content, err := readEditedFile(path)
	if err != nil {
		return contacts.ContactNoteInput{}, err
	}
	return parseContactNoteTemplate(string(content)), nil
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
