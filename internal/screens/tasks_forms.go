package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/frontmatter"
	"github.com/elpdev/telex-cli/internal/tasks"
)

var taskProjectDocumentFieldOrder = []string{"name"}
var taskCardDocumentFieldOrder = []string{"title", "column"}

type taskCardDocumentInput struct {
	Card   tasks.CardInput
	Column string
}

func editTaskProjectTemplate(name string) (tasks.ProjectInput, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("telex-task-project-%d.md", time.Now().UnixNano()))
	if err := os.WriteFile(path, []byte(renderTaskProjectDocument(name)), 0o600); err != nil {
		return tasks.ProjectInput{}, err
	}
	if err := runTaskEditor(path); err != nil {
		return tasks.ProjectInput{}, err
	}
	content, err := readEditedFile(path)
	if err != nil {
		return tasks.ProjectInput{}, err
	}
	return parseTaskProjectDocument(string(content)), nil
}

func editTaskCardTemplate(title, body, column string) (taskCardDocumentInput, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("telex-task-card-%d.md", time.Now().UnixNano()))
	if err := os.WriteFile(path, []byte(renderTaskCardDocument(title, body, column)), 0o600); err != nil {
		return taskCardDocumentInput{}, err
	}
	if err := runTaskEditor(path); err != nil {
		return taskCardDocumentInput{}, err
	}
	content, err := readEditedFile(path)
	if err != nil {
		return taskCardDocumentInput{}, err
	}
	return parseTaskCardTemplate(string(content)), nil
}

func renderTaskProjectDocument(name string) string {
	return frontmatter.RenderWithOrder(map[string]string{"name": name}, taskProjectDocumentFieldOrder, "")
}

func parseTaskProjectDocument(content string) tasks.ProjectInput {
	if strings.HasPrefix(strings.ReplaceAll(content, "\r\n", "\n"), "---\n") {
		doc, err := frontmatter.Parse(content)
		if err == nil {
			name := strings.TrimSpace(doc.Fields["name"])
			if name == "" {
				name = defaultTitle
			}
			return tasks.ProjectInput{Name: name, Body: content}
		}
	}
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.ToLower(line), "name:") {
			if parsed := strings.TrimSpace(line[len("Name:"):]); parsed != "" {
				return tasks.ProjectInput{Name: parsed, Body: content}
			}
		}
	}
	return tasks.ProjectInput{Name: defaultTitle, Body: content}
}

func renderTaskCardDocument(title, body, column string) string {
	if strings.HasPrefix(strings.ReplaceAll(body, "\r\n", "\n"), "---\n") {
		return body
	}
	fields := map[string]string{"title": title}
	if strings.TrimSpace(column) != "" {
		fields["column"] = strings.TrimSpace(column)
	}
	return frontmatter.RenderWithOrder(fields, taskCardDocumentFieldOrder, body)
}

func parseTaskCardTemplate(content string) taskCardDocumentInput {
	if strings.HasPrefix(strings.ReplaceAll(content, "\r\n", "\n"), "---\n") {
		doc, err := frontmatter.Parse(content)
		if err == nil {
			title := strings.TrimSpace(doc.Fields["title"])
			if title == "" {
				title = defaultTitle
			}
			return taskCardDocumentInput{Card: tasks.CardInput{Title: title, Body: content}, Column: strings.TrimSpace(doc.Fields["column"])}
		}
	}
	lines := strings.Split(content, "\n")
	title := defaultTitle
	if len(lines) > 0 && strings.HasPrefix(strings.ToLower(lines[0]), "title:") {
		if parsed := strings.TrimSpace(lines[0][len("Title:"):]); parsed != "" {
			title = parsed
		}
	}
	return taskCardDocumentInput{Card: tasks.CardInput{Title: title, Body: content}}
}

func runTaskEditor(path string) error {
	editor := tasksEditorCommand()
	if editor == "" {
		_ = os.Remove(path)
		return fmt.Errorf("set TELEX_TASKS_EDITOR, TELEX_NOTES_EDITOR, VISUAL, or EDITOR to edit tasks")
	}
	if err := runEditorCommand(editor, path, "set TELEX_TASKS_EDITOR, TELEX_NOTES_EDITOR, VISUAL, or EDITOR to edit tasks"); err != nil {
		return fmt.Errorf("%w; edited file kept at %s", err, path)
	}
	return nil
}

func tasksEditorCommand() string {
	if editor := strings.TrimSpace(os.Getenv("TELEX_TASKS_EDITOR")); editor != "" {
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
