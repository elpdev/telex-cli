package commands

import (
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

func newThemePage(themes []theme.Theme, original string) tuipalette.SelectPage {
	items := make([]tuipalette.SelectItem, 0, len(themes))
	selected := 0
	var cancelData any
	for i, candidate := range themes {
		current := candidate.Name == original
		if current {
			selected = i
			cancelData = candidate
		}
		items = append(items, tuipalette.SelectItem{Label: candidate.Name, Current: current, Value: candidate})
	}
	return tuipalette.NewSelectPage(tuipalette.SelectPageOptions{
		Title:       "Telex / Themes",
		Subtitle:    "Move to preview, enter to select, esc to go back.",
		Items:       items,
		Selected:    selected,
		PreviewPage: "theme-preview",
		ConfirmPage: "theme-confirm",
		CancelPage:  "theme-cancel",
		CancelData:  cancelData,
	})
}
