package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/telex-cli/internal/commands"
	"github.com/elpdev/telex-cli/internal/screens"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case routeMsg:
		m.switchScreen(msg.ScreenID)
		m.showCommandPalette = false
		m.updateDerivedScreens()
		return m, m.screens[m.activeScreen].Init()
	case toggleSidebarMsg:
		m.showSidebar = !m.showSidebar
		m.logs.Info(fmt.Sprintf("Sidebar toggled: %t", m.showSidebar))
		m.updateDerivedScreens()
		return m, nil
	case quitMsg:
		m.logs.Info("Command executed: Quit")
		return m, tea.Quit
	case commandsExecutedMsg:
		m.logs.Info(fmt.Sprintf("Command executed: %s", msg.Title))
		return m, msg.Cmd
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	if m.showCommandPalette {
		palette, cmd := m.commandPalette.Update(msg)
		m.commandPalette = palette
		return m, cmd
	}

	active := m.screens[m.activeScreen]
	updated, cmd := active.Update(msg)
	m.screens[m.activeScreen] = updated
	return m, cmd
}

type commandsExecutedMsg struct {
	Title string
	Cmd   tea.Cmd
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.ForceQuit) {
		return m, tea.Quit
	}

	if m.showCommandPalette {
		palette, cmd := m.commandPalette.Update(msg)
		m.commandPalette = palette
		if action := m.commandPalette.Action(); action.Type != commands.PaletteActionNone {
			return m.handlePaletteAction(action)
		}
		if executed := m.commandPalette.ExecutedCommand(); executed != nil {
			m.showCommandPalette = false
			m.commandPalette.Reset(m.theme.Name)
			return m, func() tea.Msg { return commandsExecutedMsg{Title: executed.Title, Cmd: executed.Run()} }
		}
		return m, cmd
	}

	if m.showHelp {
		if key.Matches(msg, m.keys.Cancel) || key.Matches(msg, m.keys.Help) {
			m.showHelp = false
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Commands):
		m.showCommandPalette = true
		m.commandPalette.Reset(m.theme.Name)
		return m, nil
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil
	case key.Matches(msg, m.keys.Cancel):
		active := m.screens[m.activeScreen]
		updated, cmd := active.Update(msg)
		m.screens[m.activeScreen] = updated
		return m, cmd
	case key.Matches(msg, m.keys.Focus):
		if m.focus == FocusMain && m.showSidebar {
			m.focus = FocusSidebar
		} else {
			m.focus = FocusMain
		}
		return m, nil
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}

	if m.focus == FocusSidebar && m.showSidebar {
		return m.handleSidebarKey(msg)
	}

	active := m.screens[m.activeScreen]
	updated, cmd := active.Update(msg)
	m.screens[m.activeScreen] = updated
	return m, cmd
}

func (m Model) handlePaletteAction(action commands.PaletteAction) (tea.Model, tea.Cmd) {
	m.commandPalette.ClearAction()
	switch action.Type {
	case commands.PaletteActionClose:
		m.showCommandPalette = false
		m.commandPalette.Reset(m.theme.Name)
		return m, nil
	case commands.PaletteActionExecute:
		m.showCommandPalette = false
		m.commandPalette.Reset(m.theme.Name)
		return m, func() tea.Msg { return commandsExecutedMsg{Title: action.Command.Title, Cmd: action.Command.Run()} }
	case commands.PaletteActionPreviewTheme:
		m.theme = *action.Theme
		m.updateDerivedScreens()
		return m, nil
	case commands.PaletteActionConfirmTheme:
		m.theme = *action.Theme
		m.logs.Info(fmt.Sprintf("Theme selected: %s", m.theme.Name))
		m.updateDerivedScreens()
		m.showCommandPalette = false
		m.commandPalette.Reset(m.theme.Name)
		return m, nil
	case commands.PaletteActionCancelTheme:
		m.theme = *action.Theme
		m.updateDerivedScreens()
		return m, nil
	}
	return m, nil
}

func (m Model) handleSidebarKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	idx := 0
	for i, id := range m.screenOrder {
		if id == m.activeScreen {
			idx = i
			break
		}
	}
	if key.Matches(msg, m.keys.Up) && idx > 0 {
		idx--
	} else if key.Matches(msg, m.keys.Down) && idx < len(m.screenOrder)-1 {
		idx++
	} else if !key.Matches(msg, m.keys.Enter) {
		return m, nil
	}
	m.switchScreen(m.screenOrder[idx])
	m.updateDerivedScreens()
	return m, m.screens[m.activeScreen].Init()
}

func (m *Model) updateDerivedScreens() {
	m.screens["settings"] = screens.NewSettings(screens.SettingsState{
		ThemeName:      m.theme.Name,
		SidebarVisible: m.showSidebar,
		Version:        m.meta.Version,
		Commit:         m.meta.Commit,
		Date:           m.meta.Date,
	})
	m.screens["logs"] = screens.NewLogs(m.logs)
}
