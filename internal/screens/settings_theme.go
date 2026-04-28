package screens

import (
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/theme"
)

func (s Settings) themeSelectView(width, height int) string {
	s.themeList.SetSize(width, max(1, height-4))
	var b strings.Builder
	b.WriteString(s.th.Title.Render("Theme"))
	b.WriteString("\n")
	b.WriteString(s.th.Muted.Render("Move to preview · enter selects · esc reverts"))
	b.WriteString("\n\n")
	b.WriteString(s.themeList.View())
	return lipgloss.NewStyle().Width(width).Height(height).Render(b.String())
}

func newSettingsThemeList(themes []theme.Theme, selected, original string, th theme.Theme, width, height int) list.Model {
	items := make([]list.Item, 0, len(themes))
	selectedIdx := 0
	for i, t := range themes {
		items = append(items, settingsThemeItem{name: t.Name, was: t.Name == original})
		if t.Name == selected {
			selectedIdx = i
		}
	}

	return newSimpleList(items, settingsThemeDelegate{th: th}, selectedIdx, width, height)
}

type settingsThemeDelegate struct {
	simpleDelegate
	th theme.Theme
}

func (d settingsThemeDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	themeItem, ok := item.(settingsThemeItem)
	if !ok {
		return
	}
	marker := "  "
	label := themeItem.name
	if themeItem.was {
		label += "  (was)"
	}
	line := padRight(marker+label, m.Width())
	if index == m.Index() {
		line = padRight("▸ "+label, m.Width())
		_, _ = io.WriteString(w, d.th.Selected.Render(line))
		return
	}
	_, _ = io.WriteString(w, d.th.Text.Render(line))
}
