package screens

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/notes"
)

func editNoteTemplate(title, body string) (notes.NoteInput, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("telex-note-%d.md", time.Now().UnixNano()))
	defer os.Remove(path)
	if err := os.WriteFile(path, []byte(fmt.Sprintf("Title: %s\n\n%s", title, body)), 0o600); err != nil {
		return notes.NoteInput{}, err
	}
	editor := notesEditorCommand()
	if editor == "" {
		return notes.NoteInput{}, fmt.Errorf("set TELEX_NOTES_EDITOR, VISUAL, or EDITOR to edit notes")
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return notes.NoteInput{}, fmt.Errorf("set TELEX_NOTES_EDITOR, VISUAL, or EDITOR to edit notes")
	}
	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return notes.NoteInput{}, err
	}
	content, err := os.ReadFile(path)
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
