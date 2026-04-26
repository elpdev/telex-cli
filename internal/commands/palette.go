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
	width    int
	height   int
}

const (
	paletteMinWidth  = 70
	paletteMaxWidth  = 120
	paletteMargin    = 8
	paletteMinHeight = 12
	paletteVMargin   = 4
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
	m.inner.Reset(tuipalette.Context{ActiveScreen: activeModule(ctx)})
}

func (m *PaletteModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.inner.SetSize(m.paletteWidth(), m.paletteHeight())
}

func (m PaletteModel) paletteWidth() int {
	if m.width <= 0 {
		return 0
	}
	w := m.width - paletteMargin
	if w > paletteMaxWidth {
		w = paletteMaxWidth
	}
	if w < paletteMinWidth {
		w = paletteMinWidth
	}
	return w
}

func (m PaletteModel) paletteHeight() int {
	if m.height <= 0 {
		return 0
	}
	h := m.height - paletteVMargin
	if h < paletteMinHeight {
		h = paletteMinHeight
	}
	return h
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
				if !command.HasCustomAvailability() && !defaultAvailableInPalette(command, m.ctx) {
					return false
				}
				return command.IsAvailable(m.ctx)
			},
			Describe: func(tuipalette.Context) string {
				return command.DescriptionFor(m.ctx)
			},
		})
	}

	m.inner = tuipalette.NewPaletteModel(registry, tuipalette.Options{
		Title:              "Telex",
		Placeholder:        "type a command, or 'mail ' / 'mail messages ' to scope...",
		Width:              m.paletteWidth(),
		Height:             m.paletteHeight(),
		Modules:            Modules(),
		Groups:             Groups(),
		ReservedNamespaces: ScopedModules(),
		Pages: map[string]tuipalette.Page{
			pageThemes: newThemePage(m.themes, m.original),
		},
	})
}

func defaultAvailableInPalette(command Command, ctx Context) bool {
	if command.Pinned {
		return true
	}
	if command.Module == "" || command.Module == ModuleGlobal {
		return true
	}
	if ctx.ActiveModule == "" {
		return true
	}
	return command.Module == ctx.ActiveModule
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
