package screens

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
)

type SettingsState struct {
	ThemeName      string
	SidebarVisible bool
	Version        string
	Commit         string
	Date           string
}

type Settings struct{ state SettingsState }

func NewSettings(state SettingsState) Settings { return Settings{state: state} }

func (s Settings) Init() tea.Cmd { return nil }

func (s Settings) Update(msg tea.Msg) (Screen, tea.Cmd) { return s, nil }

func (s Settings) View(width, height int) string {
	content := strings.Join([]string{
		"Settings",
		"",
		fmt.Sprintf("Theme: %s", s.state.ThemeName),
		fmt.Sprintf("Sidebar visible: %t", s.state.SidebarVisible),
		"",
		"Build",
		fmt.Sprintf("Version: %s", s.state.Version),
		fmt.Sprintf("Commit: %s", s.state.Commit),
		fmt.Sprintf("Date: %s", s.state.Date),
	}, "\n")
	return lipgloss.NewStyle().Width(width).Height(height).Render(content)
}

func (s Settings) Title() string { return "Settings" }

func (s Settings) KeyBindings() []key.Binding { return nil }
