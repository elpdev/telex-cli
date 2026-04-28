package screens

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

func (c Calendar) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if c.filePickerOpen {
		return c.handleImportFileMsg(msg)
	}
	if c.form != nil {
		return c.updateForm(msg)
	}

	switch msg := msg.(type) {
	case calendarLoadedMsg:
		c.loading = false
		c.err = msg.err
		if msg.err == nil {
			c.allItems = msg.items
			c.calendars = msg.calendars
			c.syncCalendarList()
			c.lastSynced = msg.lastSynced
			c.cachedEvents = msg.cachedEvents
			c.applyAgendaFilter()
		}
		return c, nil
	case calendarSyncedMsg:
		c.syncing = false
		c.syncErr = msg.syncErr
		if msg.loaded.err == nil {
			c.err = nil
			c.allItems = msg.loaded.items
			c.calendars = msg.loaded.calendars
			c.syncCalendarList()
			c.lastSynced = msg.loaded.lastSynced
			c.cachedEvents = msg.loaded.cachedEvents
			c.applyAgendaFilter()
		} else if msg.syncErr == nil {
			c.err = msg.loaded.err
			c.status = ""
			return c, nil
		}
		if msg.syncErr != nil {
			if msg.loaded.err != nil {
				c.err = msg.loaded.err
			}
			c.status = ""
			return c, nil
		}
		c.status = fmt.Sprintf("Synced %d calendar(s), %d event(s), %d occurrence(s)", msg.result.Calendars, msg.result.Events, msg.result.Occurrences)
		return c, nil
	case calendarActionFinishedMsg:
		c.loading = false
		c.err = msg.err
		if msg.err != nil {
			c.status = fmt.Sprintf("Calendar action failed: %v", msg.err)
			return c, nil
		}
		c.status = msg.status
		c.allItems = msg.loaded.items
		c.calendars = msg.loaded.calendars
		c.syncCalendarList()
		c.invitation = msg.invitation
		if msg.invitation != nil {
			c.detail = true
		} else {
			c.detail = false
		}
		c.form = nil
		c.formKind = calendarFormNone
		c.filePickerOpen = false
		c.importCalendar = 0
		c.confirm = ""
		c.confirmAction = ""
		c.confirmID = 0
		c.applyAgendaFilter()
		return c, nil
	case CalendarActionMsg:
		c.syncErr = nil
		return c.handleAction(msg.Action)
	case tea.KeyPressMsg:
		return c.handleKey(msg)
	}
	return c, nil
}
