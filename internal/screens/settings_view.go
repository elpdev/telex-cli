package screens

import (
	"charm.land/lipgloss/v2"
	"strings"
)

func (s Settings) View(width, height int) string {
	if s.mode == settingsModeThemes {
		return s.themeSelectView(width, height)
	}
	var b strings.Builder
	focusedRowIdx := -1
	if s.cursor >= 0 && s.cursor < len(focusableSettingsRowIdx) {
		focusedRowIdx = focusableSettingsRowIdx[s.cursor]
	}
	for i, row := range settingsRows {
		switch row.kind {
		case rowSection:
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(s.th.Title.Render(row.label))
			b.WriteString("\n")
		default:
			line := s.formatRow(row, width)
			if i == focusedRowIdx {
				line = s.th.Selected.Render(line)
			} else {
				line = s.th.Text.Render(line)
			}
			b.WriteString(line + "\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(s.th.Muted.Render("↑/↓ move · enter activate · esc back"))
	return lipgloss.NewStyle().Width(width).Height(height).Render(b.String())
}

func (s Settings) formatRow(row settingsRowDef, width int) string {
	const indent = "  "
	const labelCol = 20

	var line string
	switch row.kind {
	case rowAction:
		text := "› " + row.label
		if s.confirming == row.id {
			text += "   press enter again to confirm"
		}
		line = indent + text
	default:
		label := padRight(row.label, labelCol)
		value := s.rowValue(row)
		switch row.kind {
		case rowSelect:
			value = padRight(value, 16) + " ›"
		case rowToggle:
			if s.toggleValue(row.id) {
				value = padRight(value, 16) + " ●"
			} else {
				value = padRight(value, 16) + " ○"
			}
		}
		line = indent + label + "  " + value
	}
	return padRight(line, width)
}

func (s Settings) rowValue(row settingsRowDef) string {
	switch row.id {
	case "theme":
		return valueOrDash(s.state.ThemeName)
	case "sidebar-visible":
		if s.state.SidebarVisible {
			return "on"
		}
		return "off"
	case "instance":
		return valueOrDash(s.state.Instance)
	case "auth-status":
		return valueOrDash(s.state.AuthStatus)
	case "mail-admin":
		return "Manage domains and inboxes"
	case "data-dir":
		return valueOrDash(s.state.DataDir)
	case "cache-size":
		if s.state.CacheSize <= 0 {
			return "0 B"
		}
		return formatBytes(s.state.CacheSize)
	case "drive-sync":
		return valueOrDash(s.state.DriveSyncMode)
	case "version":
		return valueOrDash(s.state.Version)
	case "commit":
		return valueOrDash(s.state.Commit)
	case "date":
		return valueOrDash(s.state.Date)
	}
	return ""
}

func (s Settings) toggleValue(id string) bool {
	switch id {
	case "sidebar-visible":
		return s.state.SidebarVisible
	}
	return false
}

func valueOrDash(value string) string {
	if value == "" {
		return "—"
	}
	return value
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
