package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/tasks"
)

func editTaskProjectTemplate(name string) (tasks.ProjectInput, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("telex-task-project-%d.md", time.Now().UnixNano()))
	if err := os.WriteFile(path, []byte("Name: "+name+"\n"), 0o600); err != nil {
		return tasks.ProjectInput{}, err
	}
	if err := runTaskEditor(path); err != nil {
		return tasks.ProjectInput{}, err
	}
	content, err := readEditedFile(path)
	if err != nil {
		return tasks.ProjectInput{}, err
	}
	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(strings.ToLower(line), "name:") {
			if parsed := strings.TrimSpace(line[len("Name:"):]); parsed != "" {
				return tasks.ProjectInput{Name: parsed}, nil
			}
		}
	}
	return tasks.ProjectInput{Name: defaultTitle}, nil
}

func editTaskCardTemplate(title, body string) (tasks.CardInput, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("telex-task-card-%d.md", time.Now().UnixNano()))
	if err := os.WriteFile(path, []byte(fmt.Sprintf("Title: %s\n\n%s", title, body)), 0o600); err != nil {
		return tasks.CardInput{}, err
	}
	if err := runTaskEditor(path); err != nil {
		return tasks.CardInput{}, err
	}
	content, err := readEditedFile(path)
	if err != nil {
		return tasks.CardInput{}, err
	}
	return parseTaskCardTemplate(string(content)), nil
}

func parseTaskCardTemplate(content string) tasks.CardInput {
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
	return tasks.CardInput{Title: title, Body: strings.Join(lines[start:], "\n")}
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
