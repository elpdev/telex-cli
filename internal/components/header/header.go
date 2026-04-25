package header

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/theme"
)

type Model struct {
	AppName     string
	ScreenTitle string
	Version     string
	Instance    string
}

var telexBanner = [3]string{
	"### ### #   ### # #",
	" #  ##  #   ##   # ",
	" #  ### ### ### # #",
}

const (
	bannerLetterCells = 19
	bannerMinWidth    = 48
	bannerMinHeight   = 4
)

func View(m Model, width, height int, t theme.Theme) string {
	frameWidth, frameHeight := t.Header.GetFrameSize()
	contentWidth := max(0, width-frameWidth)
	contentHeight := max(0, height-frameHeight)

	var content string
	if contentHeight >= 3 && contentWidth >= bannerMinWidth {
		content = renderBanner(m, contentWidth, t)
	} else {
		content = renderCompact(m, contentWidth, t)
	}

	return t.Header.Width(width).Height(height).Render(content)
}

func renderCompact(m Model, contentWidth int, t theme.Theme) string {
	brandRaw := m.AppName
	if m.Instance != "" {
		brandRaw = m.AppName + "  " + m.Instance
	}
	rightRaw := fmt.Sprintf("%s  %s", m.ScreenTitle, m.Version)
	leftW := lipgloss.Width(brandRaw)
	rightW := lipgloss.Width(rightRaw)

	brand := t.Title.Render(m.AppName)
	if m.Instance != "" {
		brand += "  " + t.Muted.Render(m.Instance)
	}
	right := t.Text.Render(rightRaw)

	gap := contentWidth - leftW - rightW - 2
	middle := buildMiddle(gap, contentWidth, leftW, rightW, t)
	return brand + middle + right
}

func renderBanner(m Model, contentWidth int, t theme.Theme) string {
	metaRaw := [3]string{m.Instance, m.ScreenTitle, m.Version}
	meta := [3]string{}
	if m.Instance != "" {
		meta[0] = t.Muted.Render(m.Instance)
	}
	meta[1] = t.Text.Render(m.ScreenTitle)
	meta[2] = t.Muted.Render(m.Version)

	rows := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		letter := t.Title.Render(telexBanner[i])
		metaW := lipgloss.Width(metaRaw[i])
		gap := contentWidth - bannerLetterCells - metaW - 2
		middle := buildMiddle(gap, contentWidth, bannerLetterCells, metaW, t)
		rows = append(rows, letter+middle+meta[i])
	}
	return strings.Join(rows, "\n")
}

func buildMiddle(gap, contentWidth, leftW, rightW int, t theme.Theme) string {
	switch {
	case gap >= 3:
		return " " + t.HeaderAccent.Render(strings.Repeat("/", gap)) + " "
	case contentWidth > leftW+rightW:
		return strings.Repeat(" ", contentWidth-leftW-rightW)
	default:
		return " "
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
