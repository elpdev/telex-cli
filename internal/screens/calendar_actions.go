package screens

import (
	"fmt"
	"strconv"

	tea "charm.land/bubbletea/v2"
)

func (c Calendar) handleAction(action string) (Screen, tea.Cmd) {
	if c.confirm != "" || c.form != nil {
		return c, nil
	}
	switch action {
	case "filter":
		if c.mode != calendarViewAgenda {
			c.mode = calendarViewAgenda
			c.detail = false
		}
		c.filtering = true
		c.filterInput = c.filter.inputString()
		c.status = "Filter agenda"
		return c, nil
	case "clear-filter":
		c.filtering = false
		c.filterInput = ""
		c.filter = calendarAgendaFilter{}
		c.applyAgendaFilter()
		c.status = "Agenda filters cleared"
		return c, nil
	case "sync":
		if c.sync == nil || c.syncing {
			if c.sync == nil {
				c.status = "Calendar sync is not configured. Open Settings and verify your Telex instance, then run `telex auth login`."
			}
			return c, nil
		}
		c.syncing = true
		c.syncErr = nil
		c.status = ""
		return c, c.syncCmd()
	case "delete":
		if c.mode == calendarViewCalendars {
			if item, ok := c.selectedCalendar(); ok {
				c.confirm = fmt.Sprintf("Delete calendar %s?", strconv.FormatInt(item.RemoteID, 10))
				c.confirmAction = "delete-calendar"
				c.confirmID = item.RemoteID
			}
			return c, nil
		}
		if item, ok := c.selected(); ok {
			c.confirm = fmt.Sprintf("Delete event %s?", strconv.FormatInt(item.EventID, 10))
			c.confirmAction = "delete-event"
			c.confirmID = item.EventID
		}
		return c, nil
	case "new":
		return c.startEventForm(calendarFormEventCreate, nil)
	case "edit":
		item, ok := c.selected()
		if !ok {
			c.status = "Select an event to edit"
			return c, nil
		}
		cached, err := c.store.ReadEvent(item.EventID)
		if err != nil {
			c.status = fmt.Sprintf("Cannot load event: %v", err)
			return c, nil
		}
		return c.startEventForm(calendarFormEventEdit, cached)
	case "today":
		c.jumpToTodayRange()
		c.loading = true
		c.status = c.rangeStatusLabel()
		return c, c.loadCmd()
	case "previous-range":
		c.shiftRange(-1)
		c.loading = true
		c.status = c.rangeStatusLabel()
		return c, c.loadCmd()
	case "next-range":
		c.shiftRange(1)
		c.loading = true
		c.status = c.rangeStatusLabel()
		return c, c.loadCmd()
	case "toggle-view":
		c.cycleMode()
		if c.mode == calendarViewWeek || c.mode == calendarViewMonth {
			c.loading = true
			return c, c.loadCmd()
		}
		return c, nil
	case "view-agenda":
		c.setMode(calendarViewAgenda)
		return c, nil
	case "view-week":
		c.setMode(calendarViewWeek)
		c.loading = true
		return c, c.loadCmd()
	case "view-month":
		c.setMode(calendarViewMonth)
		c.loading = true
		return c, c.loadCmd()
	case "view-calendars":
		c.setMode(calendarViewCalendars)
		return c, nil
	case "new-calendar":
		c.mode = calendarViewCalendars
		return c.startCalendarForm(calendarFormCalendarCreate, nil)
	case "edit-calendar":
		item, ok := c.selectedCalendar()
		if !ok {
			c.status = "Select a calendar to edit"
			return c, nil
		}
		return c.startCalendarForm(calendarFormCalendarEdit, &item)
	case "delete-calendar":
		item, ok := c.selectedCalendar()
		if !ok {
			c.status = "Select a calendar to delete"
			return c, nil
		}
		c.confirm = fmt.Sprintf("Delete calendar %s?", strconv.FormatInt(item.RemoteID, 10))
		c.confirmAction = "delete-calendar"
		c.confirmID = item.RemoteID
		return c, nil
	case "import-ics":
		if c.mode != calendarViewCalendars {
			c.mode = calendarViewCalendars
			c.detail = false
			c.status = "Select a calendar, then import ICS"
			return c, nil
		}
		return c.startImportICS()
	case "invitation-show":
		messageID := c.selectedInvitationMessageID()
		if messageID <= 0 {
			c.status = "Selected event has no linked invitation message"
			return c, nil
		}
		if c.showInvite == nil {
			c.status = "Invitation details are not configured"
			return c, nil
		}
		c.loading = true
		c.detail = true
		return c, c.invitationCmd(messageID, "show", "")
	case "invitation-sync":
		messageID := c.selectedInvitationMessageID()
		if messageID <= 0 {
			c.status = "Selected event has no linked invitation message"
			return c, nil
		}
		if c.syncInvite == nil {
			c.status = "Invitation sync is not configured"
			return c, nil
		}
		c.loading = true
		return c, c.invitationCmd(messageID, "sync", "")
	case "invitation-accepted", "invitation-tentative", "invitation-declined", "invitation-needs-action":
		messageID := c.selectedInvitationMessageID()
		if messageID <= 0 {
			c.status = "Selected event has no linked invitation message"
			return c, nil
		}
		if c.respondInvite == nil {
			c.status = "Invitation responses are not configured"
			return c, nil
		}
		c.loading = true
		return c, c.invitationCmd(messageID, "respond", invitationStatusFromAction(action))
	}
	return c, nil
}
