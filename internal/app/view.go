package app

import (
	"strings"

	"github.com/elpdev/telex-cli/internal/components/footer"
	"github.com/elpdev/telex-cli/internal/components/header"
	"github.com/elpdev/telex-cli/internal/components/modal"
	"github.com/elpdev/telex-cli/internal/components/sidebar"
	"github.com/elpdev/telex-cli/internal/layout"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
)

func (m Model) View() tea.View {
	if m.width <= 0 || m.height <= 0 {
		return tea.NewView("initializing...")
	}

	dims := layout.Calculate(m.width, m.height, m.showSidebar)
	active := m.screens[m.activeScreen]

	head := header.View(header.Model{AppName: "Telex", ScreenTitle: active.Title(), Version: m.meta.Version}, dims.Header.Width, dims.Header.Height, m.theme)
	foot := footer.View(m.keys.ShortHelp(), dims.Footer.Width, dims.Footer.Height, m.theme)
	mainFrameWidth, mainFrameHeight := m.theme.Main.GetFrameSize()
	mainWidth := max(0, dims.Main.Width-mainFrameWidth)
	mainHeight := max(0, dims.Main.Height-mainFrameHeight)
	main := m.theme.Main.Width(mainWidth).Height(mainHeight).Render(active.View(mainWidth, mainHeight))

	body := main
	if m.showSidebar && dims.Sidebar.Width > 0 {
		side := sidebar.View(sidebar.Model{Items: m.sidebarItems(), ActiveID: m.activeScreen, Focused: m.focus == FocusSidebar}, dims.Sidebar.Width, dims.Sidebar.Height, m.theme)
		body = lipgloss.JoinHorizontal(lipgloss.Top, side, main)
	}

	view := lipgloss.JoinVertical(lipgloss.Left, head, body, foot)
	view = lipgloss.NewStyle().Width(m.width).Height(m.height).Background(m.theme.Background).Render(view)

	if m.showHelp {
		view = modal.Overlay(view, m.helpOverlay(), m.width, m.height, m.theme)
	}
	if m.showCommandPalette {
		view = modal.Overlay(view, m.commandPalette.View(m.theme), m.width, m.height, m.theme)
	}

	rendered := tea.NewView(view)
	rendered.AltScreen = true
	rendered.BackgroundColor = m.theme.Background
	return rendered
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m Model) sidebarItems() []sidebar.Item {
	items := make([]sidebar.Item, 0, len(m.screenOrder))
	for _, id := range m.screenOrder {
		items = append(items, sidebar.Item{ID: id, Title: m.screens[id].Title()})
	}
	return items
}

func (m Model) helpOverlay() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Keyboard Help"))
	b.WriteString("\n\nGlobal keys\n")
	for _, group := range m.keys.FullHelp() {
		for _, binding := range group {
			b.WriteString(formatBinding(binding))
		}
	}
	if active := m.screens[m.activeScreen]; active != nil {
		b.WriteString("\nScreen keys\n")
		for _, binding := range active.KeyBindings() {
			b.WriteString(formatBinding(binding))
		}
	}
	b.WriteString("\nPress esc to close.")
	return m.theme.Modal.Width(56).Render(b.String())
}

func formatBinding(binding key.Binding) string {
	return "  " + binding.Help().Key + "  " + binding.Help().Desc + "\n"
}
