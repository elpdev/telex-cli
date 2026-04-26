package filepicker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestInitialLoadUsesBubblesFilepickerView(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "file.txt"), "body")

	m := initModel(New("", root, ModeOpenFile))
	view := m.View(80, 10)

	if !strings.Contains(view, "file.txt") {
		t.Fatalf("view = %q", view)
	}
}

func TestEnterSelectsFilesInFileMode(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	mustWrite(t, path, "body")
	m := initModel(New("", root, ModeOpenFile))

	_, action, _ := m.Update(keyPress("enter"))

	if action.Type != ActionSelect || action.Path != path {
		t.Fatalf("action = %#v", action)
	}
}

func TestEnterOpensDirectoriesInFileMode(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "docs")
	mustMkdir(t, dir)
	m := initModel(New("", root, ModeOpenFile))

	updated, action, cmd := m.Update(keyPress("enter"))
	if cmd != nil {
		updated, _, _ = updated.Update(cmd())
	}

	if action.Type != ActionNone || updated.Cwd != dir {
		t.Fatalf("cwd = %q action = %#v", updated.Cwd, action)
	}
}

func TestEscEmitsCancel(t *testing.T) {
	m := initModel(New("", t.TempDir(), ModeOpenFile))

	_, action, _ := m.Update(keyPress("esc"))

	if action.Type != ActionCancel {
		t.Fatalf("action = %#v", action)
	}
}

func initModel(m Model) Model {
	cmd := m.Init()
	if cmd == nil {
		return m
	}
	updated, _, _ := m.Update(cmd())
	return updated
}

func keyPress(value string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: value})
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
}
