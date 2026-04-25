package filepicker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Mode int

const (
	ModeOpenFile Mode = iota
	ModeOpenDirectory
)

type ActionType int

const (
	ActionNone ActionType = iota
	ActionSelect
	ActionCancel
)

type Action struct {
	Type ActionType
	Path string
}

type Entry struct {
	Name    string
	Path    string
	IsDir   bool
	Hidden  bool
	Symlink bool
	Size    int64
}

type Model struct {
	Root       string
	Cwd        string
	Mode       Mode
	Entries    []Entry
	Selected   int
	Filter     string
	Filtering  bool
	ShowHidden bool
	Err        error
	action     Action
}

func New(root, cwd string, mode Mode) Model {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if root != "" {
		root = cleanAbs(root)
	}
	cwd = cleanAbs(cwd)
	if root != "" && !insideRoot(root, cwd) {
		cwd = root
	}
	m := Model{Root: root, Cwd: cwd, Mode: mode}
	_ = m.Reload()
	return m
}

func (m *Model) Reload() error {
	entries, err := readEntries(m.Cwd, m.ShowHidden, m.Filter)
	if err != nil {
		m.Err = err
		return err
	}
	m.Err = nil
	m.Entries = entries
	if m.Selected >= len(m.Entries) {
		m.Selected = maxIndex(len(m.Entries))
	}
	return nil
}

func (m Model) Update(msg tea.KeyPressMsg) (Model, Action) {
	m.action = Action{}
	if m.Filtering {
		return m.updateFilter(msg)
	}
	switch keyText(msg) {
	case "up", "k":
		if m.Selected > 0 {
			m.Selected--
		}
	case "down", "j":
		if m.Selected < len(m.Entries)-1 {
			m.Selected++
		}
	case "enter":
		m = m.enter()
	case " ":
		if m.Mode == ModeOpenDirectory {
			m.action = Action{Type: ActionSelect, Path: m.Cwd}
		}
	case "esc":
		m.action = Action{Type: ActionCancel}
	case "backspace", "h":
		m = m.parent()
	case "/":
		m.Filtering = true
		m.Filter = ""
		_ = m.Reload()
	case ".":
		m.ShowHidden = !m.ShowHidden
		_ = m.Reload()
	case "~":
		if home, err := os.UserHomeDir(); err == nil {
			m = m.chdir(home)
		}
	case "ctrl+r":
		_ = m.Reload()
	}
	return m, m.action
}

func (m Model) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Files: %s\n", m.Cwd))
	if m.Filtering {
		b.WriteString("Filter: " + m.Filter + "\n")
	}
	if m.Err != nil {
		b.WriteString("Error: " + m.Err.Error() + "\n")
	}
	b.WriteString("\n")
	if len(m.Entries) == 0 {
		b.WriteString("No matching files.\n")
		return style.Render(b.String())
	}
	limit := height - 4
	if limit < 1 || limit > len(m.Entries) {
		limit = len(m.Entries)
	}
	start := 0
	if m.Selected >= limit {
		start = m.Selected - limit + 1
	}
	for i := start; i < len(m.Entries) && i < start+limit; i++ {
		entry := m.Entries[i]
		cursor := "  "
		if i == m.Selected {
			cursor = "> "
		}
		kind := "file"
		if entry.IsDir {
			kind = "dir "
		}
		name := entry.Name
		if entry.Symlink {
			name += "@"
		}
		b.WriteString(fmt.Sprintf("%s%s  %s\n", cursor, kind, name))
	}
	return style.Render(b.String())
}

func (m Model) Action() Action { return m.action }

func (m Model) updateFilter(msg tea.KeyPressMsg) (Model, Action) {
	switch keyText(msg) {
	case "esc":
		m.Filtering = false
		m.Filter = ""
		_ = m.Reload()
	case "enter":
		m.Filtering = false
	case "backspace":
		if len(m.Filter) > 0 {
			m.Filter = m.Filter[:len(m.Filter)-1]
			_ = m.Reload()
		}
	default:
		if msg.Text != "" && len([]rune(msg.Text)) == 1 {
			m.Filter += msg.Text
			_ = m.Reload()
		}
	}
	return m, m.action
}

func (m Model) enter() Model {
	if len(m.Entries) == 0 {
		if m.Mode == ModeOpenDirectory {
			m.action = Action{Type: ActionSelect, Path: m.Cwd}
		}
		return m
	}
	entry := m.Entries[m.Selected]
	if entry.IsDir {
		m = m.chdir(entry.Path)
		return m
	}
	if m.Mode == ModeOpenFile {
		m.action = Action{Type: ActionSelect, Path: entry.Path}
	}
	return m
}

func (m Model) parent() Model {
	parent := filepath.Dir(m.Cwd)
	if parent == m.Cwd || (m.Root != "" && !insideRoot(m.Root, parent)) {
		return m
	}
	return m.chdir(parent)
}

func (m Model) chdir(path string) Model {
	path = cleanAbs(path)
	if m.Root != "" && !insideRoot(m.Root, path) {
		path = m.Root
	}
	old := m.Cwd
	m.Cwd = path
	m.Selected = 0
	if err := m.Reload(); err != nil {
		m.Cwd = old
		_ = m.Reload()
		m.Err = err
	}
	return m
}

func readEntries(cwd string, showHidden bool, filter string) ([]Entry, error) {
	dirs, err := os.ReadDir(cwd)
	if err != nil {
		return nil, err
	}
	filter = strings.ToLower(strings.TrimSpace(filter))
	out := make([]Entry, 0, len(dirs))
	for _, dir := range dirs {
		name := dir.Name()
		hidden := strings.HasPrefix(name, ".")
		if hidden && !showHidden {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToLower(name), filter) {
			continue
		}
		info, err := dir.Info()
		if err != nil {
			continue
		}
		mode := info.Mode()
		out = append(out, Entry{Name: name, Path: filepath.Join(cwd, name), IsDir: dir.IsDir(), Hidden: hidden, Symlink: mode&os.ModeSymlink != 0, Size: info.Size()})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func keyText(msg tea.KeyPressMsg) string {
	if msg.Text != "" {
		return msg.Text
	}
	return msg.String()
}

func cleanAbs(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

func insideRoot(root, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func maxIndex(length int) int {
	if length <= 0 {
		return 0
	}
	return length - 1
}
