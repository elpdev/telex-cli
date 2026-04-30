package screens

import (
	tea "charm.land/bubbletea/v2"
)

func (c Calendar) handleGridShift(days int) (Screen, tea.Cmd) {
	prevStart := c.rangeStart
	prevEnd := c.rangeEnd
	c.shiftSelectedDate(days)
	if !c.rangeStart.Equal(prevStart) || !c.rangeEnd.Equal(prevEnd) {
		c.loading = true
		c.status = c.modeStatus()
		return c, c.loadCmd()
	}
	return c, nil
}

func (c Calendar) openSelectedDay() (Screen, tea.Cmd) {
	if c.selectedDate.IsZero() {
		return c, nil
	}
	idx := firstAgendaIndexForDate(c.items, c.selectedDate)
	if idx < 0 {
		c.status = "No events on " + c.selectedDate.Format("Jan 02, 2006")
		return c, nil
	}
	c.index = idx
	c.detail = true
	return c, nil
}

func (c Calendar) openSelectedSlot() (Screen, tea.Cmd) {
	if c.selectedDate.IsZero() {
		return c, nil
	}
	idx := firstAgendaIndexForSlot(c.items, c.selectedDate, c.hourCursor)
	if idx < 0 {
		idx = firstAgendaIndexForDate(c.items, c.selectedDate)
	}
	if idx < 0 {
		c.status = "No events on " + c.selectedDate.Format("Jan 02, 2006")
		return c, nil
	}
	c.index = idx
	c.detail = true
	return c, nil
}
