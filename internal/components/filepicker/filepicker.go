package filepicker

import (
	"os"
	"path/filepath"
	"strings"

	bubblefilepicker "charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
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

type Model struct {
	Mode      Mode
	Cwd       string
	Err       error
	inner     bubblefilepicker.Model
	lastWidth int
}

func New(_ string, cwd string, mode Mode) Model {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if cwd == "" {
		cwd = "."
	}
	if abs, err := filepath.Abs(cwd); err == nil {
		cwd = abs
	}

	inner := bubblefilepicker.New()
	inner.CurrentDirectory = filepath.Clean(cwd)
	inner.ShowPermissions = false
	inner.ShowSize = true
	inner.ShowHidden = true
	inner.FileAllowed = mode == ModeOpenFile
	inner.DirAllowed = mode == ModeOpenDirectory
	inner.AutoHeight = false
	inner.KeyMap.Back = key.NewBinding(key.WithKeys("h", "backspace", "left"), key.WithHelp("h", "back"))

	return Model{Mode: mode, Cwd: inner.CurrentDirectory, inner: inner}
}

func (m Model) Init() tea.Cmd {
	return m.inner.Init()
}

func (m Model) Update(msg tea.Msg) (Model, Action, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "esc" {
		return m, Action{Type: ActionCancel}, nil
	}

	inner, cmd := m.inner.Update(msg)
	m.inner = inner
	m.Cwd = m.inner.CurrentDirectory

	if selected, path := m.inner.DidSelectFile(msg); selected {
		return m, Action{Type: ActionSelect, Path: path}, cmd
	}
	return m, Action{}, cmd
}

func (m Model) View(width, height int) string {
	if height < 1 {
		height = 1
	}
	header := []string{
		"Files: " + m.Cwd,
		"enter select/open  h/left/backspace parent  esc cancel",
		"",
	}
	pickerHeight := height - len(header)
	if pickerHeight < 1 {
		pickerHeight = 1
	}
	m.inner.SetHeight(pickerHeight)
	inner, _ := m.inner.Update(tea.WindowSizeMsg{Width: width, Height: pickerHeight})
	m.inner = inner
	m.lastWidth = width
	view := strings.Join(header, "\n") + m.inner.View()
	if width <= 0 {
		return view
	}
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	for i, line := range lines {
		if len(line) > width {
			lines[i] = line[:width]
		}
	}
	return strings.Join(lines, "\n")
}
