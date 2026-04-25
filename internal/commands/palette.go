package commands

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/theme"
)

const paletteWidth = 62

const (
	pageDraftActions   = "draft-actions"
	pageMessageActions = "message-actions"
	pageThemes         = "themes"
)

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
	subPage  string
	original string
	ctx      Context
}

type palettePage int

const (
	paletteRoot palettePage = iota
	paletteThemes
	paletteSub
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
			if m.page == paletteSub {
				m.page = paletteRoot
				m.subPage = ""
				m.query = ""
				m.selected = 0
				return m, nil
			}
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
			if command.OpensPage != "" {
				m.openPage(command.OpensPage)
				return m, nil
			}
			m.executed = &command
			m.action = PaletteAction{Type: PaletteActionExecute, Command: &command}
			return m, nil
		case "backspace", "ctrl+h":
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.selected = 0
			} else if m.page == paletteSub {
				m.page = paletteRoot
				m.subPage = ""
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

	innerWidth := paletteWidth - t.Modal.GetHorizontalFrameSize()
	var b strings.Builder
	b.WriteString(renderPaletteHeader(t, m.headerTitle(), m.headerSubtitle(), innerWidth))
	b.WriteString("\n")
	query := m.query
	if query == "" {
		query = t.Muted.Render(m.placeholder())
	}
	b.WriteString("> " + query)
	b.WriteString("\n\n")

	if m.query == "" && m.page == paletteRoot {
		b.WriteString(m.renderSections(t, innerWidth))
	} else {
		matches := m.matches()
		if len(matches) == 0 {
			b.WriteString(t.Muted.Render("No commands found"))
		} else {
			b.WriteString(m.renderList(t, matches, innerWidth))
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

func (m *PaletteModel) Reset(currentTheme string, ctx Context) {
	m.query = ""
	m.selected = 0
	m.executed = nil
	m.action = PaletteAction{}
	m.page = paletteRoot
	m.subPage = ""
	m.original = currentTheme
	m.ctx = ctx
}

func (m PaletteModel) ExecutedCommand() *Command { return m.executed }

func (m PaletteModel) Action() PaletteAction { return m.action }

func (m *PaletteModel) ClearAction() { m.action = PaletteAction{} }

func (m PaletteModel) matches() []Command {
	if m.page == paletteSub {
		return m.subPageCommands()
	}
	if m.query == "" && m.page == paletteRoot {
		var out []Command
		for _, group := range m.registry.GroupByModule(m.ctx) {
			out = append(out, group.Commands...)
		}
		return out
	}
	return m.registry.Filter(m.query, m.ctx)
}

func (m PaletteModel) subPageCommands() []Command {
	var module, group string
	switch m.subPage {
	case pageDraftActions:
		module, group = ModuleMail, GroupDrafts
	case pageMessageActions:
		module, group = ModuleMail, GroupMessages
	default:
		return nil
	}
	out := make([]Command, 0)
	for _, cmd := range m.registry.List() {
		if cmd.Module != module || cmd.Group != group {
			continue
		}
		if cmd.OpensPage != "" {
			continue
		}
		if !cmd.IsAvailable(m.ctx) {
			continue
		}
		if m.query != "" && !textMatch(cmd, m.query) {
			continue
		}
		out = append(out, cmd)
	}
	return out
}

func (m *PaletteModel) openPage(name string) {
	switch name {
	case pageThemes:
		m.page = paletteThemes
		m.query = ""
		m.selected = m.themeIndex(m.original)
	case pageDraftActions, pageMessageActions:
		m.page = paletteSub
		m.subPage = name
		m.query = ""
		m.selected = 0
	}
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

func (m PaletteModel) headerTitle() string {
	if m.page == paletteSub {
		switch m.subPage {
		case pageDraftActions:
			return "Telex / Drafts"
		case pageMessageActions:
			return "Telex / Messages"
		}
	}
	return "Telex"
}

func (m PaletteModel) headerSubtitle() string {
	if m.page == paletteSub && m.ctx.Selection != nil && m.ctx.Selection.Subject != "" {
		return m.ctx.Selection.Subject
	}
	return ""
}

func (m PaletteModel) placeholder() string {
	if m.page == paletteSub {
		return "filter actions..."
	}
	return "type a command, or 'mail '/'drafts ' to scope..."
}

func (m PaletteModel) renderList(t theme.Theme, cmds []Command, width int) string {
	scope, _ := ParseScope(m.query)
	var b strings.Builder
	for i, cmd := range cmds {
		line := formatRow(t, cmd, scope, m.ctx, width)
		if i == m.selected {
			line = t.Selected.Render(line)
		} else {
			line = t.Text.Render(line)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m PaletteModel) renderSections(t theme.Theme, width int) string {
	groups := m.registry.GroupByModule(m.ctx)
	var b strings.Builder
	idx := 0
	for _, group := range groups {
		if len(group.Commands) == 0 {
			if isReservedNamespace(group.Module) {
				b.WriteString(t.Muted.Render(strings.ToUpper(group.Module) + "  coming soon"))
				b.WriteString("\n\n")
			}
			continue
		}
		b.WriteString(t.Muted.Render(strings.ToUpper(group.Module)))
		b.WriteString("\n")
		for _, cmd := range group.Commands {
			line := formatRow(t, cmd, Scope{Module: cmd.Module}, m.ctx, width)
			if idx == m.selected {
				line = t.Selected.Render(line)
			} else {
				line = t.Text.Render(line)
			}
			b.WriteString(line + "\n")
			idx++
		}
		b.WriteString("\n")
	}
	return b.String()
}

func isReservedNamespace(module string) bool {
	return module == ModuleCalendar || module == ModuleDrive || module == ModuleNotes
}

func formatRow(t theme.Theme, cmd Command, scope Scope, ctx Context, width int) string {
	label := renderLabel(cmd, scope)
	desc := cmd.DescriptionFor(ctx)
	shortcut := cmd.Shortcut

	labelCol := 28
	descCol := width - labelCol - len(shortcut) - 2
	if descCol < 0 {
		descCol = 0
	}
	if len(desc) > descCol && descCol > 1 {
		desc = desc[:descCol-1] + "…"
	}
	row := fmt.Sprintf("%-*s %-*s", labelCol, truncate(label, labelCol), descCol, desc)
	if shortcut != "" {
		row += " " + t.PaletteAccent.Render(shortcut)
	}
	return row
}

func renderLabel(cmd Command, scope Scope) string {
	parts := make([]string, 0, 3)
	if cmd.Module != "" && cmd.Module != ModuleGlobal && scope.Module == "" {
		parts = append(parts, titleCase(cmd.Module))
	}
	if cmd.Group != "" && scope.Group == "" {
		parts = append(parts, titleCase(cmd.Group))
	}
	parts = append(parts, cmd.Title)
	return strings.Join(parts, ": ")
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
