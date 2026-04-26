package screens

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Help struct{ bindings [][]key.Binding }

func NewHelp(bindings [][]key.Binding) Help { return Help{bindings: bindings} }

func (h Help) Init() tea.Cmd { return nil }

func (h Help) Update(msg tea.Msg) (Screen, tea.Cmd) { return h, nil }

func (h Help) View(width, height int) string {
	var b strings.Builder
	b.WriteString("Help\n\nGlobal keys\n")
	for _, group := range h.bindings {
		for _, binding := range group {
			help := binding.Help()
			b.WriteString("  " + help.Key + "  " + help.Desc + "\n")
		}
	}
	b.WriteString("\nNavigation keys\n")
	b.WriteString("  up/k  move up in sidebar\n")
	b.WriteString("  down/j  move down in sidebar\n")
	b.WriteString("  enter  open selected sidebar item\n")
	b.WriteString("\nCommand palette keys\n")
	b.WriteString("  ctrl+k  open palette\n")
	b.WriteString("  type  filter commands\n")
	b.WriteString("  enter  run selected command\n")
	b.WriteString("  esc  close palette\n")
	return lipgloss.NewStyle().Width(width).Height(height).Render(b.String())
}

func (h Help) Title() string { return "Help" }

func (h Help) KeyBindings() []key.Binding { return nil }
