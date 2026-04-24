package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type Theme struct {
	Name       string
	Background color.Color

	Text     lipgloss.Style
	Muted    lipgloss.Style
	Title    lipgloss.Style
	Selected lipgloss.Style
	Header   lipgloss.Style
	Sidebar  lipgloss.Style
	Main     lipgloss.Style
	Footer   lipgloss.Style
	Modal    lipgloss.Style
	Border   lipgloss.Style
	Info     lipgloss.Style
	Warn     lipgloss.Style
}
