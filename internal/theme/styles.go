package theme

import (
	"charm.land/lipgloss/v2"
	"github.com/elpdev/tuitheme"
)

func BuiltIns() []Theme {
	return []Theme{Phosphor(), MutedDark(), Miami()}
}

func Next(current string) Theme {
	return tuitheme.Next(BuiltIns(), current)
}

func MutedDark() Theme {
	primary := lipgloss.Color("#A78BFA")
	muted := lipgloss.Color("#9CA3AF")
	border := lipgloss.Color("#374151")
	bg := lipgloss.Color("#111827")
	return Theme{
		Name:          "Muted Dark",
		Background:    bg,
		Text:          lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Background(bg),
		Muted:         lipgloss.NewStyle().Foreground(muted).Background(bg),
		Title:         lipgloss.NewStyle().Bold(true).Foreground(primary).Background(bg),
		Selected:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#111827")).Background(primary),
		Header:        lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Background(bg).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(border).Padding(0, 1),
		HeaderAccent:  lipgloss.NewStyle().Foreground(primary).Background(bg).Bold(true),
		Sidebar:       lipgloss.NewStyle().Background(bg).Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(border).Padding(1, 1),
		Main:          lipgloss.NewStyle().Background(bg).Padding(1, 2),
		Footer:        lipgloss.NewStyle().Foreground(muted).Background(bg).Border(lipgloss.NormalBorder(), true, false, false, false).BorderForeground(border).Padding(0, 1),
		Modal:         lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Background(lipgloss.Color("#1F2937")).Border(lipgloss.RoundedBorder()).BorderForeground(primary).Padding(1, 2),
		PaletteAccent: lipgloss.NewStyle().Foreground(primary).Background(lipgloss.Color("#1F2937")).Bold(true),
		Border:        lipgloss.NewStyle().Foreground(border).Background(bg),
		Info:          lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")).Background(bg),
		Warn:          lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Background(bg),
	}
}

func Phosphor() Theme {
	return tuitheme.Phosphor()
}

func Miami() Theme {
	bright := lipgloss.Color("#F0E6FF")
	muted := lipgloss.Color("#8B7BBF")
	subtle := lipgloss.Color("#A888C9")
	bg := lipgloss.Color("#1A0B2E")
	selected := lipgloss.Color("#102A55")
	divider := lipgloss.Color("#164B7A")
	pink := lipgloss.Color("#FF2E88")
	cyan := lipgloss.Color("#00E5FF")
	blue := lipgloss.Color("#2D7DFF")
	orange := lipgloss.Color("#FF8C42")

	return Theme{
		Name:          "Miami",
		Background:    bg,
		Text:          lipgloss.NewStyle().Foreground(bright).Background(bg),
		Muted:         lipgloss.NewStyle().Foreground(muted).Background(bg),
		Title:         lipgloss.NewStyle().Bold(true).Foreground(pink).Background(selected),
		Selected:      lipgloss.NewStyle().Bold(true).Foreground(cyan).Background(selected),
		Header:        lipgloss.NewStyle().Foreground(bright).Background(bg).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(cyan).Padding(0, 1),
		HeaderAccent:  lipgloss.NewStyle().Foreground(pink).Background(bg).Bold(true),
		Sidebar:       lipgloss.NewStyle().Background(bg).Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(blue).Padding(1, 1),
		Main:          lipgloss.NewStyle().Foreground(bright).Background(bg).Padding(1, 2),
		Footer:        lipgloss.NewStyle().Foreground(subtle).Background(bg).Border(lipgloss.NormalBorder(), true, false, false, false).BorderForeground(blue).Padding(0, 1),
		Modal:         lipgloss.NewStyle().Foreground(bright).Background(bg).Border(lipgloss.RoundedBorder()).BorderForeground(pink).Padding(1, 2),
		PaletteAccent: lipgloss.NewStyle().Foreground(pink).Background(bg).Bold(true),
		Border:        lipgloss.NewStyle().Foreground(divider).Background(bg),
		Info:          lipgloss.NewStyle().Foreground(cyan).Background(bg),
		Warn:          lipgloss.NewStyle().Foreground(orange).Background(bg),
	}
}
