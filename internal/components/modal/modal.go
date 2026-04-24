package modal

import (
	"github.com/elpdev/telex-cli/internal/theme"
	"charm.land/lipgloss/v2"
)

func Overlay(base, content string, width, height int, _ theme.Theme) string {
	cw, ch := lipgloss.Width(content), lipgloss.Height(content)
	x := max(0, (width-cw)/2)
	y := max(0, (height-ch)/2)

	baseLayer := lipgloss.NewLayer(base).Z(0)
	contentLayer := lipgloss.NewLayer(content).X(x).Y(y).Z(1)

	canvas := lipgloss.NewCanvas(width, height)
	canvas.Compose(lipgloss.NewCompositor(baseLayer, contentLayer))
	return canvas.Render()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
