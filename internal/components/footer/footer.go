package footer

import (
	"strings"

	"github.com/elpdev/telex-cli/internal/theme"
	"github.com/charmbracelet/bubbles/key"
)

func View(bindings []key.Binding, width, height int, t theme.Theme) string {
	parts := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		help := binding.Help()
		parts = append(parts, help.Key+" "+help.Desc)
	}
	frameWidth, frameHeight := t.Footer.GetFrameSize()
	return t.Footer.Width(max(0, width-frameWidth)).Height(max(0, height-frameHeight)).Render(strings.Join(parts, "   "))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
