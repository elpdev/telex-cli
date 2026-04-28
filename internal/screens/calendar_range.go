package screens

import (
	"fmt"
	"time"
)

func (c Calendar) modeTitle() string {
	if c.mode == calendarViewCalendars {
		return "Calendars"
	}
	return "Agenda"
}

func (c *Calendar) toggleMode() {
	c.detail = false
	if c.mode == calendarViewCalendars {
		c.mode = calendarViewAgenda
		c.status = "Showing agenda"
		return
	}
	c.mode = calendarViewCalendars
	c.status = "Showing calendars"
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
	c.rangeStart = today
	c.rangeEnd = today.AddDate(0, 0, 30)
	c.detail = false
}

func (c *Calendar) shiftRange(direction int) {
	start, end := c.activeRange()
	duration := end.Sub(start)
	if duration <= 0 {
		duration = 30 * 24 * time.Hour
	}
	shift := time.Duration(direction) * duration
	c.rangeStart = start.Add(shift)
	c.rangeEnd = end.Add(shift)
	c.index = 0
	c.detail = false
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

func calendarRangeDate(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}
