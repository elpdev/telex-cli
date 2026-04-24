package screens

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
)

type Home struct{}

func NewHome() Home { return Home{} }

func (h Home) Init() tea.Cmd { return nil }

func (h Home) Update(msg tea.Msg) (Screen, tea.Cmd) { return h, nil }

func (h Home) View(width, height int) string {
	content := strings.Join([]string{
		"Telex",
		"",
		"An opinionated Bubble Tea starter kit for building polished Go TUIs.",
		"",
		"Press ctrl+k to open command palette.",
		"Press ? for help.",
		"Use the sidebar to navigate between starter screens.",
	}, "\n")
	return lipgloss.NewStyle().Width(width).Height(height).Render(content)
}

func (h Home) Title() string { return "Home" }

func (h Home) KeyBindings() []key.Binding { return nil }
