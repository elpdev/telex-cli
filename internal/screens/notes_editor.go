package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/notes"
)

func editNoteTemplate(title, body string) (notes.NoteInput, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("telex-note-%d.md", time.Now().UnixNano()))
	if err := os.WriteFile(path, []byte(fmt.Sprintf("Title: %s\n\n%s", title, body)), 0o600); err != nil {
		return notes.NoteInput{}, err
	}
	editor := notesEditorCommand()
	if editor == "" {
		_ = os.Remove(path)
		return notes.NoteInput{}, fmt.Errorf("set TELEX_NOTES_EDITOR, VISUAL, or EDITOR to edit notes")
	}
	if err := runEditorCommand(editor, path, "set TELEX_NOTES_EDITOR, VISUAL, or EDITOR to edit notes"); err != nil {
		return notes.NoteInput{}, fmt.Errorf("%w; edited file kept at %s", err, path)
	}
	content, err := readEditedFile(path)
	if err != nil {
		return notes.NoteInput{}, err
	}
	return parseNoteTemplate(string(content)), nil
}

func notesEditorCommand() string {
	if editor := strings.TrimSpace(os.Getenv("TELEX_NOTES_EDITOR")); editor != "" {
		return editor
	}
	if editor := strings.TrimSpace(os.Getenv("VISUAL")); editor != "" {
		return editor
	}
	return strings.TrimSpace(os.Getenv("EDITOR"))
}

func parseNoteTemplate(content string) notes.NoteInput {
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
	return notes.NoteInput{Title: title, Body: strings.Join(lines[start:], "\n")}
}
