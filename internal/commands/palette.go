package commands

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/theme"
	"github.com/elpdev/tuipalette"
)

const pageThemes = "themes"

type PaletteModel struct {
	registry *Registry
	themes   []theme.Theme
	inner    tuipalette.PaletteModel
	ctx      Context
	original string
	executed *Command
	action   PaletteAction
}

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
	m := PaletteModel{registry: registry, themes: themes}
	m.rebuildInner()
	return m
}

func (m PaletteModel) Update(msg tea.Msg) (PaletteModel, tea.Cmd) {
	m.executed = nil
	m.action = PaletteAction{}
	inner, cmd := m.inner.Update(msg)
	m.inner = inner
	m.translateAction(m.inner.Action())
	return m, cmd
}

func (m PaletteModel) View(t theme.Theme) string {
	m.inner.SetStyles(stylesFromTheme(t))
	return m.inner.View()
}

func (m *PaletteModel) Reset(currentTheme string, ctx Context) {
	m.ctx = ctx
	m.original = currentTheme
	m.executed = nil
	m.action = PaletteAction{}
	m.rebuildInner()
	m.inner.Reset(tuipalette.Context{ActiveScreen: ctx.ActiveScreen})
}

func (m PaletteModel) ExecutedCommand() *Command { return m.executed }

func (m PaletteModel) Action() PaletteAction { return m.action }

func (m *PaletteModel) ClearAction() {
	m.action = PaletteAction{}
	m.inner.ClearAction()
}

func (m *PaletteModel) rebuildInner() {
	registry := tuipalette.NewRegistry()
	registry.SetDefaultModule(ModuleGlobal)
	for _, cmd := range m.registry.List() {
		command := cmd
		registry.Register(tuipalette.Command{
			ID:          command.ID,
			Module:      command.Module,
			Group:       command.Group,
			Title:       command.Title,
			Description: command.Description,
			Shortcut:    command.Shortcut,
			Keywords:    command.Keywords,
			OpensPage:   command.OpensPage,
			Run:         command.Run,
			Available: func(tuipalette.Context) bool {
				return command.IsAvailable(m.ctx)
			},
			Describe: func(tuipalette.Context) string {
				return command.DescriptionFor(m.ctx)
			},
		})
	}

	m.inner = tuipalette.NewPaletteModel(registry, tuipalette.Options{
		Title:              "Telex",
		Placeholder:        "type a command, or 'mail ' to scope...",
		Modules:            []string{ModuleMail, ModuleCalendar, ModuleDrive, ModuleNotes, ModuleHackerNews, ModuleSettings, ModuleGlobal},
		Groups:             []string{GroupDrafts, GroupMessages, GroupOutbox, GroupInbox},
		ReservedNamespaces: []string{ModuleCalendar, ModuleDrive, ModuleNotes},
		Pages: map[string]tuipalette.Page{
			pageThemes: newThemePage(m.themes, m.original),
		},
	})
}

func (m *PaletteModel) translateAction(action tuipalette.PaletteAction) {
	switch action.Type {
	case tuipalette.PaletteActionClose:
		m.action = PaletteAction{Type: PaletteActionClose}
	case tuipalette.PaletteActionExecute:
		if action.Command == nil {
			return
		}
		command, ok := m.registry.Find(action.Command.ID)
		if !ok {
			return
		}
		m.executed = &command
		m.action = PaletteAction{Type: PaletteActionExecute, Command: &command}
	case tuipalette.PaletteActionBack:
		if action.Page == "theme-cancel" {
			if selected, ok := action.Data.(theme.Theme); ok {
				m.action = PaletteAction{Type: PaletteActionCancelTheme, Theme: &selected}
			}
		}
	case tuipalette.PaletteActionPage:
		selected, ok := action.Data.(theme.Theme)
		if !ok {
			return
		}
		switch action.Page {
		case "theme-preview":
			m.action = PaletteAction{Type: PaletteActionPreviewTheme, Theme: &selected}
		case "theme-confirm":
			m.action = PaletteAction{Type: PaletteActionConfirmTheme, Theme: &selected}
		}
	}
}

func stylesFromTheme(t theme.Theme) tuipalette.Styles {
	return tuipalette.Styles{
		Modal:    t.Modal,
		Title:    t.Title,
		Text:     t.Text,
		Muted:    t.Muted,
		Selected: t.Selected,
		Accent:   t.PaletteAccent,
	}
}

type themePage struct {
	themes   []theme.Theme
	original string
	selected int
}

func newThemePage(themes []theme.Theme, original string) themePage {
	page := themePage{themes: append([]theme.Theme(nil), themes...), original: original}
	page.selected = page.themeIndex(original)
	return page
}

func (p themePage) Update(msg tea.KeyPressMsg) (tuipalette.Page, tuipalette.PaletteAction) {
	switch msg.String() {
	case "esc", "backspace", "ctrl+h":
		if original, ok := p.themeByName(p.original); ok {
			return p, tuipalette.PaletteAction{Type: tuipalette.PaletteActionBack, Page: "theme-cancel", Data: original}
		}
		return p, tuipalette.PaletteAction{Type: tuipalette.PaletteActionBack, Page: "theme-cancel"}
	case "up", "ctrl+p":
		if p.selected > 0 {
			p.selected--
			return p, p.previewSelectedTheme()
		}
	case "down", "ctrl+n":
		if p.selected < len(p.themes)-1 {
			p.selected++
			return p, p.previewSelectedTheme()
		}
	case "enter":
		if len(p.themes) > 0 {
			selected := p.themes[p.selected]
			return p, tuipalette.PaletteAction{Type: tuipalette.PaletteActionPage, Page: "theme-confirm", Data: selected}
		}
	}
	return p, tuipalette.PaletteAction{}
}

func (p themePage) View(styles tuipalette.Styles, width int) string {
	innerWidth := width - styles.Modal.GetHorizontalFrameSize()
	var b strings.Builder
	b.WriteString(renderPaletteHeader(styles, "Telex / Themes", "Move to preview, enter to select, esc to go back.", innerWidth))
	b.WriteString("\n\n")

	for i, candidate := range p.themes {
		line := candidate.Name
		if candidate.Name == p.original {
			line += "  current"
		}
		if i == p.selected {
			line = styles.Selected.Render("> " + line)
		} else {
			line = styles.Text.Render("  " + line)
		}
		b.WriteString(line + "\n")
	}

	return styles.Modal.Width(width).Render(b.String())
}

func (p themePage) Reset() {
	p.selected = p.themeIndex(p.original)
}

func (p themePage) previewSelectedTheme() tuipalette.PaletteAction {
	if len(p.themes) == 0 {
		return tuipalette.PaletteAction{}
	}
	selected := p.themes[p.selected]
	return tuipalette.PaletteAction{Type: tuipalette.PaletteActionPage, Page: "theme-preview", Data: selected}
}

func (p themePage) themeIndex(name string) int {
	for i, candidate := range p.themes {
		if candidate.Name == name {
			return i
		}
	}
	return 0
}

func (p themePage) themeByName(name string) (theme.Theme, bool) {
	for _, candidate := range p.themes {
		if candidate.Name == name {
			return candidate, true
		}
	}
	return theme.Theme{}, false
}

func renderPaletteHeader(styles tuipalette.Styles, title, subtitle string, width int) string {
	rendered := styles.Title.Render(title)
	slashCount := max(3, width-len(title)-1)
	slashes := styles.Accent.Render(strings.Repeat("/", slashCount))
	out := rendered + " " + slashes
	if subtitle != "" {
		out += "\n" + styles.Muted.Width(width).Render(subtitle)
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
