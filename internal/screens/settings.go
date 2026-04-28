package screens

import (
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/theme"
)

func NewSettings(state SettingsState, th theme.Theme, themes []theme.Theme, actions SettingsActions) Settings {
	return Settings{
		state:     state,
		th:        th,
		themes:    themes,
		actions:   actions,
		themeList: newSettingsThemeList(themes, state.ThemeName, state.ThemeName, th, 0, 0),
		keys:      defaultSettingsKeyMap(),
	}
}

func (s Settings) Reconfigure(state SettingsState, th theme.Theme, themes []theme.Theme, actions SettingsActions) Settings {
	s.state = state
	s.th = th
	s.themes = themes
	s.actions = actions
	if s.cursor < 0 {
		s.cursor = 0
	}
	if s.cursor >= len(focusableSettingsRowIdx) {
		s.cursor = len(focusableSettingsRowIdx) - 1
	}
	selected := state.ThemeName
	if s.mode == settingsModeThemes {
		if name, ok := s.selectedThemeName(); ok {
			selected = name
		}
	}
	s.themeList = newSettingsThemeList(s.themes, selected, s.preTheme, s.th, s.themeList.Width(), s.themeList.Height())
	return s
}

func (s Settings) Init() tea.Cmd { return nil }

func (s Settings) Title() string { return "Settings" }
