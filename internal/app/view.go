package app

import (
	"strings"

	helpbubble "charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/components/footer"
	"github.com/elpdev/telex-cli/internal/components/header"
	"github.com/elpdev/telex-cli/internal/components/modal"
	"github.com/elpdev/telex-cli/internal/components/sidebar"
	"github.com/elpdev/telex-cli/internal/layout"
	"github.com/elpdev/telex-cli/internal/theme"
)

func (m Model) View() tea.View {
	if m.width <= 0 || m.height <= 0 {
		return tea.NewView("initializing...")
	}

	dims := layout.Calculate(m.width, m.height, m.showSidebar)
	active := m.screens[m.activeScreen]

	head := header.View(header.Model{AppName: "Telex", ScreenTitle: active.Title(), Version: m.meta.Version, Instance: m.instance}, dims.Header.Width, dims.Header.Height, m.theme)
	foot := footer.View(m.help, m.keys.ShortHelp(), dims.Footer.Width, dims.Footer.Height, m.theme)
	mainFrameWidth, mainFrameHeight := m.theme.Main.GetFrameSize()
	mainWidth := max(0, dims.Main.Width-mainFrameWidth)
	mainHeight := max(0, dims.Main.Height-mainFrameHeight)
	main := m.theme.Main.Width(dims.Main.Width).Height(dims.Main.Height).Render(active.View(mainWidth, mainHeight))

	body := main
	if m.showSidebar && dims.Sidebar.Width > 0 {
		side := sidebar.View(sidebar.Model{Items: m.sidebarItems(), ActiveID: m.activeScreen, CursorID: m.currentSidebarCursorID(), Focused: m.focus == FocusSidebar}, dims.Sidebar.Width, dims.Sidebar.Height, m.theme)
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
	ids := m.sidebarScreenIDs()
	items := make([]sidebar.Item, 0, len(ids))
	for _, id := range ids {
		title := m.screens[id].Title()
		if isMailSection(m.activeScreen) && id == "mail" {
			title = "Mailboxes"
		}
		items = append(items, sidebar.Item{ID: id, Title: title})
	}
	return items
}

func (m Model) sidebarScreenIDs() []string {
	if isMailSection(m.activeScreen) {
		ids := []string{"home", "mail-unread", "mail-inbox", "mail-sent", "mail-drafts", "mail-outbox", "mail-junk", "mail-archive", "mail-trash", "mail", "mail-admin"}
		out := make([]string, 0, len(ids))
		for _, id := range ids {
			if _, ok := m.screens[id]; ok {
				out = append(out, id)
			}
		}
		return out
	}
	return append([]string(nil), m.screenOrder...)
}

func (m Model) currentSidebarCursorID() string {
	ids := m.sidebarScreenIDs()
	if containsScreenID(ids, m.sidebarCursor) {
		return m.sidebarCursor
	}
	if containsScreenID(ids, m.activeScreen) {
		return m.activeScreen
	}
	if len(ids) > 0 {
		return ids[0]
	}
	return ""
}

func containsScreenID(ids []string, id string) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

func isMailSection(id string) bool {
	return id == "mail-admin" || isMailScreen(id)
}

func (m Model) helpOverlay() string {
	help := m.help
	help.ShowAll = true
	help.SetWidth(52)
	help.Styles = helpStyles(m.theme)

	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Keyboard Help"))
	b.WriteString("\n\nGlobal keys\n")
	b.WriteString(help.FullHelpView(m.keys.FullHelp()))
	if active := m.screens[m.activeScreen]; active != nil {
		b.WriteString("\nScreen keys\n")
		b.WriteString(help.FullHelpView([][]key.Binding{active.KeyBindings()}))
	}
	b.WriteString("\nPress esc to close.")
	return m.theme.Modal.Width(56).Render(b.String())
}

func helpStyles(t theme.Theme) helpbubble.Styles {
	styles := helpbubble.DefaultStyles(false)
	styles.ShortKey = t.Footer.Bold(true)
	styles.ShortDesc = t.Footer
	styles.ShortSeparator = t.Footer
	styles.FullKey = t.Selected
	styles.FullDesc = t.Text
	styles.FullSeparator = t.Muted
	styles.Ellipsis = t.Muted
	return styles
}
