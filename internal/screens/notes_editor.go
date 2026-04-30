package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/frontmatter"
	"github.com/elpdev/telex-cli/internal/notes"
)

var noteDocumentFieldOrder = []string{"title", "folder_id"}

func editNoteTemplate(title, body string, folderID int64) (notes.NoteInput, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("telex-note-%d.md", time.Now().UnixNano()))
	if err := os.WriteFile(path, []byte(renderNoteDocument(title, body, folderID)), 0o600); err != nil {
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
	return parseNoteTemplate(string(content), folderID), nil
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

func renderNoteDocument(title, body string, folderID int64) string {
	if strings.HasPrefix(strings.ReplaceAll(body, "\r\n", "\n"), "---\n") {
		return body
	}
	fields := map[string]string{"title": title}
	if folderID > 0 {
		fields["folder_id"] = strconv.FormatInt(folderID, 10)
	}
	return frontmatter.RenderWithOrder(fields, noteDocumentFieldOrder, body)
}

func parseNoteTemplate(content string, fallbackFolderID int64) notes.NoteInput {
	if strings.HasPrefix(strings.ReplaceAll(content, "\r\n", "\n"), "---\n") {
		return parseNoteDocument(content, fallbackFolderID)
	}
	lines := strings.Split(content, "\n")
	title := defaultTitle
	if len(lines) > 0 && strings.HasPrefix(strings.ToLower(lines[0]), "title:") {
		if parsed := strings.TrimSpace(lines[0][len("Title:"):]); parsed != "" {
			title = parsed
		}
	}
	return notes.NoteInput{Title: title, Body: content, FolderID: noteFolderIDPtr(fallbackFolderID)}
}

func parseNoteDocument(content string, fallbackFolderID int64) notes.NoteInput {
	doc, err := frontmatter.Parse(content)
	if err != nil {
		return notes.NoteInput{Title: defaultTitle, Body: content, FolderID: noteFolderIDPtr(fallbackFolderID)}
	}
	title := strings.TrimSpace(doc.Fields["title"])
	if title == "" {
		title = defaultTitle
	}
	folderID := fallbackFolderID
	if raw := strings.TrimSpace(doc.Fields["folder_id"]); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			folderID = parsed
		}
	}
	return notes.NoteInput{Title: title, Body: content, FolderID: noteFolderIDPtr(folderID)}
}

func noteFolderIDPtr(folderID int64) *int64 {
	if folderID <= 0 {
		return nil
	}
	return &folderID
}
