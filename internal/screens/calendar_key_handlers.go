package screens

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (c Calendar) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if c.filtering {
		return c.handleFilterKey(msg)
	}
	if c.confirm != "" {
		if msg.String() == "y" || msg.String() == "Y" {
			action := c.confirmAction
			id := c.confirmID
			c.confirm = ""
			c.confirmAction = ""
			c.confirmID = 0
			if action == "delete-event" && id > 0 && c.deleteEvent != nil {
				c.loading = true
				return c, c.deleteCmd(id)
			}
			if action == "delete-calendar" && id > 0 && c.deleteCalendar != nil {
				c.loading = true
				return c, c.deleteCalendarCmd(id)
			}
		}
		if key.Matches(msg, c.keys.Back) || msg.String() == "n" || msg.String() == "N" {
			c.confirm = ""
			c.confirmAction = ""
			c.confirmID = 0
		}
		return c, nil
	}
	if key.Matches(msg, c.keys.ViewAgenda) {
		return c.handleAction("view-agenda")
	}
	if key.Matches(msg, c.keys.ViewWeek) {
		return c.handleAction("view-week")
	}
	if key.Matches(msg, c.keys.ViewMonth) {
		return c.handleAction("view-month")
	}
	if key.Matches(msg, c.keys.ViewCalendars) {
		return c.handleAction("view-calendars")
	}
	if c.mode == calendarViewCalendars {
		if key.Matches(msg, c.keys.Back) {
			c.setMode(calendarViewAgenda)
			return c, nil
		}
		c.ensureCalendarList()
		updated, cmd := c.calendarList.Update(msg)
		c.calendarList = updated
		c.calendarIndex = c.calendarList.GlobalIndex()
		c.clampCalendarIndex()
		if cmd != nil {
			return c, cmd
		}
	}
	if key.Matches(msg, c.keys.Back) && c.detail {
		c.detail = false
		return c, nil
	}
	switch c.mode {
	case calendarViewAgenda:
		if key.Matches(msg, c.keys.Up) && c.index > 0 {
			c.index--
			return c, nil
		}
		if key.Matches(msg, c.keys.Down) && c.index < len(c.items)-1 {
			c.index++
			return c, nil
		}
		if key.Matches(msg, c.keys.Open) && len(c.items) > 0 {
			c.detail = true
			return c, nil
		}
	case calendarViewMonth:
		if key.Matches(msg, c.keys.Up) {
			return c.handleGridShift(-7)
		}
		if key.Matches(msg, c.keys.Down) {
			return c.handleGridShift(7)
		}
		if key.Matches(msg, c.keys.Left) {
			return c.handleGridShift(-1)
		}
		if key.Matches(msg, c.keys.Right) {
			return c.handleGridShift(1)
		}
		if key.Matches(msg, c.keys.Open) {
			return c.openSelectedDay()
		}
	case calendarViewWeek:
		if key.Matches(msg, c.keys.Up) {
			c.shiftHour(-1)
			return c, nil
		}
		if key.Matches(msg, c.keys.Down) {
			c.shiftHour(1)
			return c, nil
		}
		if key.Matches(msg, c.keys.Left) {
			return c.handleGridShift(-1)
		}
		if key.Matches(msg, c.keys.Right) {
			return c.handleGridShift(1)
		}
		if key.Matches(msg, c.keys.Open) {
			return c.openSelectedSlot()
		}
	}
	if key.Matches(msg, c.keys.Refresh) {
		c.loading = true
		return c, c.loadCmd()
	}
	if key.Matches(msg, c.keys.Sync) {
		return c.handleAction("sync")
	}
	if key.Matches(msg, c.keys.Today) {
		return c.handleAction("today")
	}
	if key.Matches(msg, c.keys.Prev) && c.modeAllowsRangeShift() && !c.detail {
		return c.handleAction("previous-range")
	}
	if key.Matches(msg, c.keys.Next) && c.modeAllowsRangeShift() && !c.detail {
		return c.handleAction("next-range")
	}
	if key.Matches(msg, c.keys.View) {
		return c.handleAction("toggle-view")
	}
	if key.Matches(msg, c.keys.New) {
		if c.mode == calendarViewCalendars {
			return c.handleAction("new-calendar")
		}
		return c.handleAction("new")
	}
	if key.Matches(msg, c.keys.Edit) {
		if c.mode == calendarViewCalendars {
			return c.handleAction("edit-calendar")
		}
		return c.handleAction("edit")
	}
	if key.Matches(msg, c.keys.Delete) {
		return c.handleAction("delete")
	}
	if key.Matches(msg, c.keys.Import) && c.mode == calendarViewCalendars {
		return c.handleAction("import-ics")
	}
	if key.Matches(msg, c.keys.Filter) && c.modeAllowsFilter() && !c.detail {
		return c.handleAction("filter")
	}
	if key.Matches(msg, c.keys.Clear) && c.modeAllowsFilter() && c.filter.active() {
		return c.handleAction("clear-filter")
	}
	return c, nil
}

func (c Calendar) handleFilterKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		c.filtering = false
		c.filterInput = ""
		c.status = "Filter cancelled"
		return c, nil
	case "enter":
		c.filtering = false
		c.filter = parseCalendarAgendaFilter(c.filterInput)
		c.filterInput = ""
		c.applyAgendaFilter()
		if c.filter.active() {
			c.status = fmt.Sprintf("Filtered agenda: %d occurrence(s)", len(c.items))
		} else {
			c.status = "Agenda filters cleared"
		}
		return c, nil
	case "backspace":
		if len(c.filterInput) > 0 {
			c.filterInput = c.filterInput[:len(c.filterInput)-1]
		}
		return c, nil
	case "ctrl+u":
		c.filterInput = ""
		return c, nil
	}
	if msg.Text != "" {
		c.filterInput += msg.Text
	}
	return c, nil
}
