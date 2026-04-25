package commands

import (
	"fmt"
	"strings"

	"github.com/elpdev/telex-cli/internal/theme"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const paletteWidth = 62

func renderPaletteHeader(t theme.Theme, title, subtitle string, width int) string {
	rendered := t.Title.Render(title)
	slashCount := max(3, width-lipgloss.Width(rendered)-1)
	slashes := t.PaletteAccent.Render(strings.Repeat("/", slashCount))
	out := rendered + " " + slashes
	if subtitle != "" {
		out += "\n" + t.Muted.Width(width).Render(subtitle)
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type PaletteModel struct {
	registry *Registry
	themes   []theme.Theme
	query    string
	selected int
	executed *Command
	action   PaletteAction
	page     palettePage
	original string
}

type palettePage int

const (
	paletteRoot palettePage = iota
	paletteThemes
)

type PaletteAction struct {
	Type    PaletteActionType
	Command *Command
	Theme   *theme.Theme
}

type PaletteActionType int

const (
	PaletteActionNone PaletteActionType = iota
	PaletteActionClose
	PaletteActionExecute
	PaletteActionPreviewTheme
	PaletteActionConfirmTheme
	PaletteActionCancelTheme
)

func NewPaletteModel(registry *Registry, themes []theme.Theme) PaletteModel {
	return PaletteModel{registry: registry, themes: themes}
}

func (m PaletteModel) Update(msg tea.Msg) (PaletteModel, tea.Cmd) {
	m.executed = nil
	m.action = PaletteAction{}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.page == paletteThemes {
			return m.updateThemes(msg), nil
		}
		switch msg.String() {
		case "esc":
			m.action = PaletteAction{Type: PaletteActionClose}
			return m, nil
		case "up", "ctrl+p":
			if m.selected > 0 {
				m.selected--
			}
			return m, nil
		case "down", "ctrl+n":
			if m.selected < len(m.matches())-1 {
				m.selected++
			}
			return m, nil
		case "enter":
			matches := m.matches()
			if len(matches) == 0 {
				return m, nil
			}
			command := matches[m.selected]
			if command.ID == "themes" {
				m.openThemes()
				return m, nil
			}
			m.executed = &command
			m.action = PaletteAction{Type: PaletteActionExecute, Command: &command}
			return m, nil
		case "backspace", "ctrl+h":
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.selected = 0
			}
			return m, nil
		case "space":
			m.query += " "
			m.selected = 0
			return m, nil
		}
		if len(msg.String()) == 1 {
			m.query += msg.String()
			m.selected = 0
			return m, nil
		}
	}
	if m.selected >= len(m.matches()) {
		m.selected = 0
	}
	return m, nil
}

func (m PaletteModel) updateThemes(msg tea.KeyPressMsg) PaletteModel {
	switch msg.String() {
	case "esc", "backspace", "ctrl+h":
		if original, ok := m.themeByName(m.original); ok {
			m.action = PaletteAction{Type: PaletteActionCancelTheme, Theme: &original}
		}
		m.page = paletteRoot
		m.query = ""
		m.selected = 0
		return m
	case "up", "ctrl+p":
		if m.selected > 0 {
			m.selected--
			m.previewSelectedTheme()
		}
		return m
	case "down", "ctrl+n":
		if m.selected < len(m.themes)-1 {
			m.selected++
			m.previewSelectedTheme()
		}
		return m
	case "enter":
		if len(m.themes) == 0 {
			return m
		}
		selected := m.themes[m.selected]
		m.action = PaletteAction{Type: PaletteActionConfirmTheme, Theme: &selected}
		return m
	}
	return m
}

func (m PaletteModel) View(t theme.Theme) string {
	if m.page == paletteThemes {
		return m.themeView(t)
	}

	matches := m.matches()
	innerWidth := paletteWidth - t.Modal.GetHorizontalFrameSize()
	var b strings.Builder
	b.WriteString(renderPaletteHeader(t, "Telex", "", innerWidth))
	b.WriteString("\n")
	query := m.query
	if query == "" {
		query = t.Muted.Render("type a command...")
	}
	b.WriteString("> " + query)
	b.WriteString("\n\n")

	if len(matches) == 0 {
		b.WriteString(t.Muted.Render("No commands found"))
	} else {
		for i, command := range matches {
			line := fmt.Sprintf("%-18s %s", command.Title, command.Description)
			if i == m.selected {
				line = t.Selected.Render(line)
			} else {
				line = t.Text.Render(line)
			}
			b.WriteString(line + "\n")
		}
	}

	return t.Modal.Width(paletteWidth).Render(b.String())
}

func (m PaletteModel) themeView(t theme.Theme) string {
	innerWidth := paletteWidth - t.Modal.GetHorizontalFrameSize()
	var b strings.Builder
	b.WriteString(renderPaletteHeader(t, "Telex / Themes", "Move to preview, enter to select, esc to go back.", innerWidth))
	b.WriteString("\n\n")

	for i, candidate := range m.themes {
		line := candidate.Name
		if candidate.Name == t.Name {
			line += "  current"
		}
		if i == m.selected {
			line = t.Selected.Render("▸ " + line)
		} else {
			line = t.Text.Render("  " + line)
		}
		b.WriteString(line + "\n")
	}

	return t.Modal.Width(paletteWidth).Render(b.String())
}

func (m *PaletteModel) Reset(currentTheme string) {
	m.query = ""
	m.selected = 0
	m.executed = nil
	m.action = PaletteAction{}
	m.page = paletteRoot
	m.original = currentTheme
}

func (m PaletteModel) ExecutedCommand() *Command { return m.executed }

func (m PaletteModel) Action() PaletteAction { return m.action }

func (m *PaletteModel) ClearAction() { m.action = PaletteAction{} }

func (m PaletteModel) matches() []Command { return m.registry.Filter(m.query) }

func (m *PaletteModel) openThemes() {
	m.page = paletteThemes
	m.query = ""
	m.selected = m.themeIndex(m.original)
}

func (m *PaletteModel) previewSelectedTheme() {
	if len(m.themes) == 0 {
		return
	}
	selected := m.themes[m.selected]
	m.action = PaletteAction{Type: PaletteActionPreviewTheme, Theme: &selected}
}

func (m PaletteModel) themeIndex(name string) int {
	for i, candidate := range m.themes {
		if candidate.Name == name {
			return i
		}
	}
	return 0
}

func (m PaletteModel) themeByName(name string) (theme.Theme, bool) {
	for _, candidate := range m.themes {
		if candidate.Name == name {
			return candidate, true
		}
	}
	return theme.Theme{}, false
}
