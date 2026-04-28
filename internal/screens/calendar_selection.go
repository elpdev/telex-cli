package screens

import "github.com/elpdev/telex-cli/internal/calendarstore"

func (c Calendar) selected() (calendarstore.OccurrenceMeta, bool) {
	if c.index < 0 || c.index >= len(c.items) {
		return calendarstore.OccurrenceMeta{}, false
	}
	return c.items[c.index], true
}

func (c Calendar) selectedCalendar() (calendarstore.CalendarMeta, bool) {
	if len(c.calendars) == 0 {
		return calendarstore.CalendarMeta{}, false
	}
	return c.calendars[c.clampedCalendarIndex()], true
}

func (c Calendar) selectedInvitationMessageID() int64 {
	item, ok := c.selected()
	if !ok {
		return 0
	}
	event, err := c.store.ReadEvent(item.EventID)
	if err != nil {
		return 0
	}
	return firstEventMessageID(event.Meta)
}

func firstEventMessageID(event calendarstore.EventMeta) int64 {
	for _, link := range event.Links {
		if link.MessageID > 0 {
			return link.MessageID
		}
	}
	for _, message := range event.Messages {
		if message.ID > 0 {
			return message.ID
		}
	}
	return 0
}

func (c *Calendar) clampIndex() {
	if c.index < 0 {
		c.index = 0
	}
	if c.index >= len(c.items) && len(c.items) > 0 {
		c.index = len(c.items) - 1
	}
	if len(c.items) == 0 {
		c.index = 0
		if c.invitation == nil {
			c.detail = false
		}
	}
	if c.calendarIndex < 0 {
		c.calendarIndex = 0
	}
	if c.calendarIndex >= len(c.calendars) && len(c.calendars) > 0 {
		c.calendarIndex = len(c.calendars) - 1
	}
	if len(c.calendars) == 0 {
		c.calendarIndex = 0
	}
}
