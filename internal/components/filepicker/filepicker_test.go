package filepicker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestInitialLoadListsVisibleFilesAndDirectories(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "file.txt"), "body")
	mustMkdir(t, filepath.Join(root, "docs"))

	m := New(root, root, ModeOpenFile)

	if len(m.Entries) != 2 {
		t.Fatalf("entries = %#v", m.Entries)
	}
	if m.Entries[0].Name != "docs" || !m.Entries[0].IsDir || m.Entries[1].Name != "file.txt" {
		t.Fatalf("entries = %#v", m.Entries)
	}
}

func TestHiddenFilesAreHiddenByDefault(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".env"), "secret")
	mustWrite(t, filepath.Join(root, "visible.txt"), "body")

	m := New(root, root, ModeOpenFile)

	if names(m.Entries) != "visible.txt" {
		t.Fatalf("entries = %#v", m.Entries)
	}
}

func TestDotTogglesHiddenFiles(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".env"), "secret")
	mustWrite(t, filepath.Join(root, "visible.txt"), "body")
	m := New(root, root, ModeOpenFile)

	m, _ = m.Update(key("."))

	if names(m.Entries) != ".env,visible.txt" {
		t.Fatalf("entries = %#v", m.Entries)
	}
}

func TestNavigationChangesSelectedIndex(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.txt"), "a")
	mustWrite(t, filepath.Join(root, "b.txt"), "b")
	m := New(root, root, ModeOpenFile)

	m, _ = m.Update(key("down"))
	if m.Selected != 1 {
		t.Fatalf("selected = %d", m.Selected)
	}
	m, _ = m.Update(key("up"))
	if m.Selected != 0 {
		t.Fatalf("selected = %d", m.Selected)
	}
}

func TestEnterOpensDirectoriesInFileMode(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "docs")
	mustMkdir(t, dir)
	m := New(root, root, ModeOpenFile)

	m, action := m.Update(key("enter"))

	if action.Type != ActionNone || m.Cwd != dir {
		t.Fatalf("cwd = %q action = %#v", m.Cwd, action)
	}
}

func TestEnterSelectsFilesInFileMode(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	mustWrite(t, path, "body")
	m := New(root, root, ModeOpenFile)

	_, action := m.Update(key("enter"))

	if action.Type != ActionSelect || action.Path != path {
		t.Fatalf("action = %#v", action)
	}
}

func TestEscEmitsCancel(t *testing.T) {
	m := New(t.TempDir(), "", ModeOpenFile)

	_, action := m.Update(key("esc"))

	if action.Type != ActionCancel {
		t.Fatalf("action = %#v", action)
	}
}

func TestFilterNarrowsEntries(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "apple.txt"), "a")
	mustWrite(t, filepath.Join(root, "banana.txt"), "b")
	m := New(root, root, ModeOpenFile)

	m, _ = m.Update(key("/"))
	m, _ = m.Update(textKey("p"))

	if names(m.Entries) != "apple.txt" {
		t.Fatalf("entries = %#v", m.Entries)
	}
}

func TestParentNavigationCannotEscapeRoot(t *testing.T) {
	root := t.TempDir()
	m := New(root, root, ModeOpenFile)

	m, _ = m.Update(key("backspace"))

	if m.Cwd != filepath.Clean(root) {
		t.Fatalf("cwd = %q", m.Cwd)
	}
}

func TestHomeNavigationWorks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	m := New("", t.TempDir(), ModeOpenFile)

	m, _ = m.Update(key("~"))

	if m.Cwd != filepath.Clean(home) {
		t.Fatalf("cwd = %q", m.Cwd)
	}
}

func key(value string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: value})
}

func textKey(value string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: value, Code: []rune(value)[0]})
}

func names(entries []Entry) string {
	values := make([]string, 0, len(entries))
	for _, entry := range entries {
		values = append(values, entry.Name)
	}
	return strings.Join(values, ",")
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
