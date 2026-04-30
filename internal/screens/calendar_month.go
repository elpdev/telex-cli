package screens

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/calendarstore"
)

func (c Calendar) monthView(width, height int) string {
	if width < 42 {
		return "Terminal too narrow for month view; press v or 1 to switch to agenda."
	}
	anchor := c.selectedDate
	if anchor.IsZero() {
		anchor = calendarRangeDate(time.Now())
	}
	grid := monthGridDays(anchor, c.weekStartsOn)
	bucket := bucketByDay(c.items)
	today := calendarRangeDate(time.Now())
	cellW, cellH, extraW, extraH := monthCellDims(width, height)

	rows := make([]string, 0, 14)
	rows = append(rows, monthWeekdayHeader(c.weekStartsOn, cellW, extraW))
	rows = append(rows, monthHorizontalRule(width))

	for r := 0; r < 6; r++ {
		rowH := cellH
		if r < extraH {
			rowH++
		}
		cells := make([]string, 0, 13)
		for d := 0; d < 7; d++ {
			cw := cellW
			if d < extraW {
				cw++
			}
			cells = append(cells, c.renderMonthCell(grid[r][d], anchor.Month(), today, c.selectedDate, bucket[dayKey(grid[r][d])], cw, rowH))
			if d < 6 {
				cells = append(cells, monthVerticalSep(rowH))
			}
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
		if r < 5 {
			rows = append(rows, monthHorizontalRule(width))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func monthWeekdayHeader(weekStartsOn time.Weekday, cellW, extraW int) string {
	names := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	if weekStartsOn == time.Sunday {
		names = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	}
	cells := make([]string, 0, 13)
	headerStyle := lipgloss.NewStyle().Bold(true)
	for d := 0; d < 7; d++ {
		cw := cellW
		if d < extraW {
			cw++
		}
		cells = append(cells, headerStyle.Render(centerText(names[d], cw)))
		if d < 6 {
			cells = append(cells, " ")
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

func monthHorizontalRule(width int) string {
	return lipgloss.NewStyle().Faint(true).Render(strings.Repeat("─", width))
}

func monthVerticalSep(rowH int) string {
	style := lipgloss.NewStyle().Faint(true)
	lines := make([]string, rowH)
	for i := range lines {
		lines[i] = style.Render("│")
	}
	return strings.Join(lines, "\n")
}

func (c Calendar) renderMonthCell(day time.Time, anchorMonth time.Month, today, selected time.Time, evs []calendarstore.OccurrenceMeta, w, h int) string {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	style := lipgloss.NewStyle().Width(w).Height(h)
	inMonth := day.Month() == anchorMonth
	dayStyle := lipgloss.NewStyle()
	if !inMonth {
		dayStyle = dayStyle.Faint(true)
	}
	isToday := day.Equal(today)
	isSelected := !selected.IsZero() && calendarRangeDate(selected).Equal(day)
	if isToday {
		dayStyle = dayStyle.Bold(true).Underline(true)
	}
	if isSelected {
		dayStyle = dayStyle.Reverse(true)
	}

	dayLabel := fmt.Sprintf("%d", day.Day())
	if isSelected {
		dayLabel = "[" + dayLabel + "]"
	}
	headerLine := alignRight(dayStyle.Render(dayLabel), w)

	if w < 9 {
		count := len(evs)
		if count > 0 {
			countLine := alignRight(fmt.Sprintf("•%d", count), w)
			return style.Render(headerLine + "\n" + countLine)
		}
		return style.Render(headerLine)
	}

	capacity := h - 1
	lines := []string{headerLine}
	if capacity > 0 && len(evs) > 0 {
		shown := 0
		for i, ev := range evs {
			remaining := capacity - shown
			if remaining <= 0 {
				break
			}
			if remaining == 1 && len(evs)-i > 1 {
				lines = append(lines, alignLeft(fmt.Sprintf("+%d more", len(evs)-i), w))
				break
			}
			bullet := c.coloredCalendarBullet(ev.CalendarID)
			titleW := w - 2
			title := truncate(ev.Title, titleW)
			lineStyle := lipgloss.NewStyle()
			if !inMonth {
				lineStyle = lineStyle.Faint(true)
			}
			lines = append(lines, lineStyle.Render(bullet+" "+padRight(title, max(0, titleW))))
			shown++
		}
	}
	for len(lines) < h {
		lines = append(lines, padRight("", w))
	}
	return style.Render(strings.Join(lines, "\n"))
}

func centerText(s string, w int) string {
	sw := lipgloss.Width(s)
	if sw >= w {
		return s
	}
	pad := w - sw
	left := pad / 2
	right := pad - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func alignRight(s string, w int) string {
	sw := lipgloss.Width(s)
	if sw >= w {
		return s
	}
	return strings.Repeat(" ", w-sw) + s
}

func alignLeft(s string, w int) string {
	return padRight(s, w)
}
