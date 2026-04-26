package footer

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/theme"
)

func View(model help.Model, bindings []key.Binding, width, height int, t theme.Theme) string {
	frameWidth, frameHeight := t.Footer.GetFrameSize()
	contentWidth := max(0, width-frameWidth)
	model.SetWidth(contentWidth)
	model.Styles = helpStyles(t)
	return t.Footer.Width(contentWidth).Height(max(0, height-frameHeight)).Render(model.ShortHelpView(bindings))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func helpStyles(t theme.Theme) help.Styles {
	styles := help.DefaultStyles(false)
	styles.ShortKey = t.Footer.Bold(true)
	styles.ShortDesc = t.Footer
	styles.ShortSeparator = t.Footer
	styles.FullKey = lipgloss.NewStyle()
	styles.FullDesc = lipgloss.NewStyle()
	styles.FullSeparator = lipgloss.NewStyle()
	styles.Ellipsis = t.Footer
	return styles
}
