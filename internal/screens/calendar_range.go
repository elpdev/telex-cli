package screens

import (
	"fmt"
	"time"
)

func (c Calendar) modeTitle() string {
	switch c.mode {
	case calendarViewCalendars:
		return "Calendars"
	case calendarViewWeek:
		return "Week"
	case calendarViewMonth:
		return "Month"
	default:
		return "Agenda"
	}
}

func (c Calendar) modeStatus() string {
	switch c.mode {
	case calendarViewCalendars:
		return "Showing calendars"
	case calendarViewWeek:
		return "Showing week of " + c.weekStartLabel()
	case calendarViewMonth:
		return "Showing " + c.monthLabel()
	default:
		return "Showing agenda"
	}
}

func (c *Calendar) cycleMode() {
	switch c.mode {
	case calendarViewAgenda:
		c.setMode(calendarViewWeek)
	case calendarViewWeek:
		c.setMode(calendarViewMonth)
	case calendarViewMonth:
		c.setMode(calendarViewCalendars)
	default:
		c.setMode(calendarViewAgenda)
	}
}

func (c *Calendar) setMode(mode calendarViewMode) {
	c.mode = mode
	c.detail = false
	if (mode == calendarViewWeek || mode == calendarViewMonth) && c.selectedDate.IsZero() {
		c.selectedDate = calendarRangeDate(time.Now())
	}
	c.setRangeForMode()
	c.status = c.modeStatus()
}

func (c *Calendar) jumpToToday() {
	today := time.Now().Format("2006-01-02")
	for i, item := range c.items {
		if item.StartsAt.Format("2006-01-02") >= today {
			c.index = i
			return
		}
	}
}

func (c *Calendar) jumpToTodayRange() {
	today := calendarRangeDate(time.Now())
	switch c.mode {
	case calendarViewMonth, calendarViewWeek:
		c.selectedDate = today
		if c.mode == calendarViewWeek {
			c.hourCursor = clampHour(time.Now().Hour(), c.visibleHourFrom, c.visibleHourTo)
		}
		c.setRangeForMode()
	default:
		c.rangeStart = today
		c.rangeEnd = today.AddDate(0, 0, 30)
	}
	c.detail = false
}

func (c *Calendar) shiftRange(direction int) {
	switch c.mode {
	case calendarViewMonth:
		anchor := c.selectedDate
		if anchor.IsZero() {
			anchor = calendarRangeDate(time.Now())
		}
		c.selectedDate = clampDayInMonth(anchor.AddDate(0, direction, 0), anchor.Day())
		c.setRangeForMode()
	case calendarViewWeek:
		anchor := c.selectedDate
		if anchor.IsZero() {
			anchor = calendarRangeDate(time.Now())
		}
		c.selectedDate = anchor.AddDate(0, 0, 7*direction)
		c.setRangeForMode()
	default:
		start, end := c.activeRange()
		duration := end.Sub(start)
		if duration <= 0 {
			duration = 30 * 24 * time.Hour
		}
		shift := time.Duration(direction) * duration
		c.rangeStart = start.Add(shift)
		c.rangeEnd = end.Add(shift)
		c.index = 0
	}
	c.detail = false
}

func (c *Calendar) shiftSelectedDate(days int) {
	anchor := c.selectedDate
	if anchor.IsZero() {
		anchor = calendarRangeDate(time.Now())
	}
	c.selectedDate = anchor.AddDate(0, 0, days)
	c.setRangeForMode()
}

func (c *Calendar) shiftHour(delta int) {
	hour := c.hourCursor + delta
	if hour < c.visibleHourFrom {
		hour = c.visibleHourFrom
	}
	if hour >= c.visibleHourTo {
		hour = c.visibleHourTo - 1
	}
	c.hourCursor = hour
}

func (c *Calendar) setRangeForMode() {
	switch c.mode {
	case calendarViewMonth:
		anchor := c.selectedDate
		if anchor.IsZero() {
			anchor = calendarRangeDate(time.Now())
		}
		first := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, anchor.Location())
		offset := (int(first.Weekday()) - int(c.weekStartsOn) + 7) % 7
		gridStart := first.AddDate(0, 0, -offset)
		gridEnd := gridStart.AddDate(0, 0, 42)
		c.rangeStart = gridStart
		c.rangeEnd = gridEnd
	case calendarViewWeek:
		anchor := c.selectedDate
		if anchor.IsZero() {
			anchor = calendarRangeDate(time.Now())
		}
		ws := startOfWeek(anchor, c.weekStartsOn)
		c.rangeStart = ws
		c.rangeEnd = ws.AddDate(0, 0, 7)
	}
}

func (c Calendar) weekStartLabel() string {
	anchor := c.selectedDate
	if anchor.IsZero() {
		anchor = calendarRangeDate(time.Now())
	}
	ws := startOfWeek(anchor, c.weekStartsOn)
	return ws.Format("Jan 02, 2006")
}

func (c Calendar) monthLabel() string {
	anchor := c.selectedDate
	if anchor.IsZero() {
		anchor = calendarRangeDate(time.Now())
	}
	return anchor.Format("January 2006")
}

func clampHour(hour, from, to int) int {
	if hour < from {
		return from
	}
	if hour >= to {
		return to - 1
	}
	return hour
}

func clampDayInMonth(t time.Time, preferDay int) time.Time {
	year, month, _ := t.Date()
	first := time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
	last := first.AddDate(0, 1, -1).Day()
	day := preferDay
	if day > last {
		day = last
	}
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func (c Calendar) activeRange() (time.Time, time.Time) {
	start := c.rangeStart
	end := c.rangeEnd
	if start.IsZero() {
		start = calendarRangeDate(time.Now())
	}
	if end.IsZero() || !end.After(start) {
		end = start.AddDate(0, 0, 30)
	}
	return start, end
}

func (c Calendar) rangeDates() (string, string) {
	start, end := c.activeRange()
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

func (c Calendar) rangeLabel() string {
	start, end := c.activeRange()
	return fmt.Sprintf("%s to %s", start.Format("Jan 02, 2006"), end.AddDate(0, 0, -1).Format("Jan 02, 2006"))
}

func (c Calendar) rangeStatusLabel() string {
	switch c.mode {
	case calendarViewMonth:
		return "Showing " + c.monthLabel()
	case calendarViewWeek:
		return "Showing week of " + c.weekStartLabel()
	default:
		return "Showing " + c.rangeLabel()
	}
}

func calendarRangeDate(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}
