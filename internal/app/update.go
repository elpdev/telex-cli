package app

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/key"
	hnscreens "github.com/elpdev/hackernews/pkg/screens"
	"github.com/elpdev/telex-cli/internal/commands"
	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/screens"
	"github.com/elpdev/tuimod"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case routeMsg:
		if msg.ScreenID == "hn-search" {
			m = m.refreshHackerNewsSearchScreen()
		}
		if msg.ScreenID == "hn-doctor" {
			return m.openHackerNewsDoctor()
		}
		m.switchScreen(msg.ScreenID)
		m.showCommandPalette = false
		m.updateDerivedScreens()
		return m, m.initScreen(m.activeScreen)
	case hnscreens.NavigateMsg:
		screenID := hackerNewsScreenID(msg.ScreenID)
		m.switchScreen(screenID)
		return m, m.initScreen(m.activeScreen)
	case hnscreens.OpenCommentsMsg:
		if existing, ok := m.screens["hn-comments"].(hnscreens.Comments); ok {
			updated, cmd := existing.Open(msg.Story, msg.ReturnTo)
			m.screens["hn-comments"] = updated
			m.switchScreen("hn-comments")
			return m, cmd
		}
		m.logs.Warn("Hacker News comments screen unavailable")
		return m, nil
	case hnscreens.HideReadToggledMsg:
		settings := m.loadHackerNewsSettings()
		settings.HideRead = msg.HideRead
		return m.applyHackerNewsSettings(settings), nil
	case hnscreens.SortModeChangedMsg:
		settings := m.loadHackerNewsSettings()
		settings.SortMode = msg.Mode
		return m.applyHackerNewsSettings(settings), nil
	case hnscreens.SettingsChangedMsg:
		return m.applyHackerNewsSettings(msg.Settings), nil
	case toggleSidebarMsg:
		m.showSidebar = !m.showSidebar
		m.logs.Info(fmt.Sprintf("Sidebar toggled: %t", m.showSidebar))
		m.updateDerivedScreens()
		return m, nil
	case screens.SettingsThemePreviewMsg:
		if t, ok := themeByName(msg.Name); ok {
			m.theme = t
			m.updateDerivedScreens()
		}
		return m, nil
	case screens.SettingsThemeChangedMsg:
		if t, ok := themeByName(msg.Name); ok {
			m.theme = t
			m.logs.Info(fmt.Sprintf("Theme selected: %s", m.theme.Name))
			m.saveUIPrefs()
			m.updateDerivedScreens()
		}
		return m, nil
	case screens.SettingsThemeCancelMsg:
		if t, ok := themeByName(msg.Name); ok {
			m.theme = t
			m.updateDerivedScreens()
		}
		return m, nil
	case screens.SettingsSidebarChangedMsg:
		m.showSidebar = msg.Visible
		m.logs.Info(fmt.Sprintf("Sidebar default set: %t", m.showSidebar))
		m.saveUIPrefs()
		m.updateDerivedScreens()
		return m, nil
	case screens.SettingsDriveSyncChangedMsg:
		if err := m.saveDriveSyncMode(msg.Mode); err != nil {
			m.logs.Warn(fmt.Sprintf("Saving drive sync mode: %v", err))
			return m, nil
		}
		m.logs.Info(fmt.Sprintf("Drive sync mode set: %s", msg.Mode))
		m.updateDerivedScreens()
		return m, nil
	case settingsSignOutMsg:
		if err := m.signOut(); err != nil {
			m.logs.Warn(fmt.Sprintf("Sign out: %v", err))
			return m, nil
		}
		m.logs.Info("Signed out")
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

	if targeted, ok := msg.(tuimod.TargetedMsg); ok {
		if id := targeted.TargetScreenID(); id != "" {
			prefixed := hackerNewsScreenID(id)
			if _, exists := m.screens[prefixed]; exists {
				return m.updateScreen(prefixed, msg)
			}
		}
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
			m.commandPalette.Reset(m.theme.Name, m.paletteContext())
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

	active := m.screens[m.activeScreen]
	if capturer, ok := active.(tuimod.KeyCapturer); ok && capturer.CapturesKey(msg) {
		updated, cmd := active.Update(msg)
		m.screens[m.activeScreen] = updated
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.keys.Commands):
		m.showCommandPalette = true
		m.commandPalette.Reset(m.theme.Name, m.paletteContext())
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
		active := m.screens[m.activeScreen]
		if capture, ok := active.(interface{ CapturesFocusKey(tea.KeyPressMsg) bool }); ok && capture.CapturesFocusKey(msg) {
			updated, cmd := active.Update(msg)
			m.screens[m.activeScreen] = updated
			return m, cmd
		}
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

	active = m.screens[m.activeScreen]
	updated, cmd := active.Update(msg)
	m.screens[m.activeScreen] = updated
	return m, cmd
}

func (m Model) updateScreen(id string, msg tea.Msg) (tea.Model, tea.Cmd) {
	active, ok := m.screens[id]
	if !ok {
		m.logs.Warn(fmt.Sprintf("Message targeted unknown screen: %s", id))
		return m, nil
	}
	updated, cmd := active.Update(msg)
	m.screens[id] = updated
	return m, cmd
}

func (m Model) handlePaletteAction(action commands.PaletteAction) (tea.Model, tea.Cmd) {
	m.commandPalette.ClearAction()
	switch action.Type {
	case commands.PaletteActionClose:
		m.showCommandPalette = false
		m.commandPalette.Reset(m.theme.Name, m.paletteContext())
		return m, nil
	case commands.PaletteActionExecute:
		m.showCommandPalette = false
		m.commandPalette.Reset(m.theme.Name, m.paletteContext())
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
		m.commandPalette.Reset(m.theme.Name, m.paletteContext())
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
	} else if key.Matches(msg, m.keys.Enter) {
		m.focus = FocusMain
	} else {
		return m, nil
	}
	m.switchScreen(m.screenOrder[idx])
	m.updateDerivedScreens()
	return m, m.initScreen(m.activeScreen)
}

func (m *Model) updateDerivedScreens() {
	m.screens["home"] = m.buildHome()
	m.screens["settings"] = m.buildSettings()
	if m.devBuild() {
		m.screens["logs"] = screens.NewLogs(m.logs)
	}
}

func (m *Model) saveUIPrefs() {
	if m.prefsPath == "" {
		return
	}
	prefs, err := config.LoadPrefs(m.prefsPath)
	if err != nil {
		m.logs.Warn(fmt.Sprintf("Loading UI prefs: %v", err))
		prefs = &config.UIPrefs{}
	}
	prefs.Theme = m.theme.Name
	visible := m.showSidebar
	prefs.SidebarVisible = &visible
	if err := prefs.SaveTo(m.prefsPath); err != nil {
		m.logs.Warn(fmt.Sprintf("Saving UI prefs: %v", err))
	}
}

func (m *Model) saveDriveSyncMode(mode string) error {
	configFile, _ := config.Paths(m.configPath)
	cfg, err := config.LoadFrom(configFile)
	if err != nil {
		return err
	}
	cfg.Drive.SyncMode = mode
	return cfg.SaveTo(configFile)
}

func (m *Model) signOut() error {
	_, tokenFile := config.Paths(m.configPath)
	if err := os.Remove(tokenFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	m.client = nil
	return nil
}
