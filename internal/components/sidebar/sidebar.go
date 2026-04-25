package sidebar

import (
	"strings"

	"github.com/elpdev/telex-cli/internal/theme"
)

type Item struct {
	ID    string
	Title string
}

type Model struct {
	Items    []Item
	ActiveID string
	Focused  bool
}

func View(m Model, width, height int, t theme.Theme) string {
	frameWidth, frameHeight := t.Sidebar.GetFrameSize()
	innerWidth := max(0, width-frameWidth)
	innerHeight := max(0, height-frameHeight)
	var b strings.Builder
	if m.Focused {
		b.WriteString(t.Title.Render("Navigation"))
	} else {
		b.WriteString(t.Muted.Render("Navigation"))
	}
	b.WriteString("\n\n")
	for _, item := range m.Items {
		if item.ID == m.ActiveID {
			b.WriteString(t.Selected.Render("▸ " + item.Title))
		} else {
			b.WriteString(t.Text.Render("  " + item.Title))
		}
		b.WriteString("\n")
	}
	return t.Sidebar.Width(innerWidth).Height(innerHeight).Render(b.String())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
