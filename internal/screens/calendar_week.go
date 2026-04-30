package screens

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/calendarstore"
)

func (c Calendar) weekView(width, height int) string {
	axisW := 5
	if width < axisW+7*4 {
		return "Terminal too narrow for week grid; press v or 1 to switch to agenda."
	}
	anchor := c.selectedDate
	if anchor.IsZero() {
		anchor = calendarRangeDate(time.Now())
	}
	weekStart := startOfWeek(anchor, c.weekStartsOn)
	dayW, extraW := weekColW(width, axisW)
	if dayW < 1 {
		return "Terminal too narrow for week grid; press v or 1 to switch to agenda."
	}

	cols := make([]time.Time, 7)
	for i := range cols {
		cols[i] = weekStart.AddDate(0, 0, i)
	}
	timed, allDay := placeWeekEvents(c.items, weekStart, c.slotsPerHour, c.visibleHourFrom, c.visibleHourTo)

	header := c.weekHeader(cols, axisW, dayW, extraW)
	allDayBlock := c.weekAllDayStrip(allDay, axisW, dayW, extraW)
	used := lipgloss.Height(header) + lipgloss.Height(allDayBlock) + 1
	bodyH := max(1, height-used)
	body := c.weekTimedBody(timed, axisW, dayW, extraW, bodyH)

	return lipgloss.JoinVertical(lipgloss.Left, header, weekHorizontalRule(width), allDayBlock, body)
}

func (c Calendar) weekHeader(cols []time.Time, axisW, dayW, extraW int) string {
	today := calendarRangeDate(time.Now())
	cells := make([]string, 0, 1+7+6)
	cells = append(cells, padRight("", axisW))
	cells = append(cells, " ")
	for i, day := range cols {
		cw := dayW
		if i < extraW {
			cw++
		}
		label := fmt.Sprintf("%s %d", day.Weekday().String()[:3], day.Day())
		style := lipgloss.NewStyle().Bold(true)
		if day.Equal(today) {
			style = style.Underline(true)
		}
		if !c.selectedDate.IsZero() && calendarRangeDate(c.selectedDate).Equal(day) {
			style = style.Reverse(true)
		}
		cells = append(cells, style.Render(centerText(label, cw)))
		if i < 6 {
			cells = append(cells, " ")
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

func (c Calendar) weekAllDayStrip(allDay [7][]calendarstore.OccurrenceMeta, axisW, dayW, extraW int) string {
	maxLines := 1
	for _, evs := range allDay {
		if len(evs) > maxLines {
			maxLines = len(evs)
		}
	}
	if maxLines > 3 {
		maxLines = 3
	}

	rows := make([]string, 0, maxLines)
	for line := 0; line < maxLines; line++ {
		cells := make([]string, 0, 1+7+6)
		axisLabel := ""
		if line == 0 {
			axisLabel = "all"
		}
		cells = append(cells, lipgloss.NewStyle().Faint(true).Render(padRight(axisLabel, axisW)))
		cells = append(cells, " ")
		for d := 0; d < 7; d++ {
			cw := dayW
			if d < extraW {
				cw++
			}
			text := ""
			if line < len(allDay[d]) {
				ev := allDay[d][line]
				if line == 2 && len(allDay[d]) > 3 {
					text = fmt.Sprintf("+%d more", len(allDay[d])-2)
				} else {
					bullet := c.coloredCalendarBullet(ev.CalendarID)
					text = bullet + " " + truncate(ev.Title, max(0, cw-2))
				}
			}
			cells = append(cells, padRight(text, cw))
			if d < 6 {
				cells = append(cells, " ")
			}
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}
	if len(rows) == 0 {
		return ""
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (c Calendar) weekTimedBody(timed [7][]weekSlot, axisW, dayW, extraW, bodyH int) string {
	slotsTotal := (c.visibleHourTo - c.visibleHourFrom) * c.slotsPerHour
	if slotsTotal < 1 {
		return ""
	}
	if bodyH < slotsTotal {
		// scroll window centered on hourCursor
		// keep simple: clamp top to a window of size bodyH containing hourCursor
	}
	visibleSlots := slotsTotal
	if bodyH < visibleSlots {
		visibleSlots = bodyH
	}
	cursorSlot := slotIndexForHour(c.hourCursor, c.visibleHourFrom, c.slotsPerHour)
	topSlot := 0
	if visibleSlots < slotsTotal {
		topSlot = cursorSlot - visibleSlots/2
		if topSlot < 0 {
			topSlot = 0
		}
		if topSlot+visibleSlots > slotsTotal {
			topSlot = slotsTotal - visibleSlots
		}
	}

	selectedCol := -1
	if !c.selectedDate.IsZero() {
		anchor := c.selectedDate
		ws := startOfWeek(anchor, c.weekStartsOn)
		selectedCol = dayDiff(ws, calendarRangeDate(anchor))
	}

	rows := make([]string, 0, visibleSlots)
	axisStyle := lipgloss.NewStyle().Faint(true)
	for s := topSlot; s < topSlot+visibleSlots; s++ {
		cells := make([]string, 0, 1+7+6)
		minutesPerSlot := 60 / c.slotsPerHour
		minutesFromStart := s * minutesPerSlot
		hour := c.visibleHourFrom + minutesFromStart/60
		minute := minutesFromStart % 60
		axisLabel := "     "
		if minute == 0 {
			axisLabel = fmt.Sprintf("%02d:00", hour)
		}
		cells = append(cells, axisStyle.Render(axisLabel))
		cells = append(cells, axisStyle.Render("│"))
		for d := 0; d < 7; d++ {
			cw := dayW
			if d < extraW {
				cw++
			}
			cells = append(cells, c.renderWeekCell(timed[d][s], d, s, selectedCol, cursorSlot, cw))
			if d < 6 {
				cells = append(cells, " ")
			}
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (c Calendar) renderWeekCell(slot weekSlot, col, slotIdx, selectedCol, cursorSlot, cw int) string {
	isCursor := col == selectedCol && slotIdx == cursorSlot
	style := lipgloss.NewStyle().Width(cw)
	if slot.EventID == 0 {
		bg := " "
		if isCursor {
			return style.Reverse(true).Render(padRight(bg, cw))
		}
		return style.Render(padRight(bg, cw))
	}
	color := c.calendarColorString(slot.CalendarID)
	bullet := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("▌")
	content := bullet
	if !slot.Continuation && slot.Title != "" {
		title := truncate(slot.Title, max(0, cw-2))
		if slot.OverflowCount > 0 && lipgloss.Width(title)+lipgloss.Width(fmt.Sprintf(" +%d", slot.OverflowCount)) <= cw-1 {
			title = title + fmt.Sprintf(" +%d", slot.OverflowCount)
		}
		content = bullet + truncate(title, max(0, cw-1))
	} else if slot.Continuation {
		content = bullet + strings.Repeat(" ", max(0, cw-1))
	}
	rendered := padRight(content, cw)
	if isCursor {
		return style.Reverse(true).Render(rendered)
	}
	return style.Render(rendered)
}

func slotIndexForHour(hour, fromHour, slotsPerHour int) int {
	if hour < fromHour {
		return 0
	}
	return (hour - fromHour) * slotsPerHour
}

func weekHorizontalRule(width int) string {
	return lipgloss.NewStyle().Faint(true).Render(strings.Repeat("─", width))
}
