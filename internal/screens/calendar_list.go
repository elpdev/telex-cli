package screens

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/calendarstore"
)

func (c Calendar) calendarListView(width, height int) string {
	if len(c.calendars) == 0 {
		return "No calendars are cached. Press S to sync remote calendars, or press n to create one. If sync fails, run `telex auth login` and verify Settings.\n"
	}
	c.ensureCalendarList()
	c.calendarList.SetSize(width, height)
	var b strings.Builder
	b.WriteString(c.calendarList.View())
	if item, ok := c.selectedCalendar(); ok {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Calendar ID: %d\nName: %s\nColor: %s\nTime zone: %s\nPosition: %d\nSource: %s\n", item.RemoteID, item.Name, item.Color, item.TimeZone, item.Position, item.Source))
	}
	return b.String()
}

func (c *Calendar) ensureCalendarList() {
	if len(c.calendarList.Items()) == len(c.calendars) {
		c.calendarList.Select(c.clampedCalendarIndex())
		return
	}
	c.syncCalendarList()
}

func (c *Calendar) syncCalendarList() {
	c.clampCalendarIndex()
	c.calendarList = newCalendarList(c.calendars, c.calendarIndex, c.calendarList.Width(), c.calendarList.Height())
}

func (c *Calendar) clampCalendarIndex() {
	c.calendarIndex = c.clampedCalendarIndex()
}

func (c Calendar) clampedCalendarIndex() int {
	if c.calendarIndex < 0 || len(c.calendars) == 0 {
		return 0
	}
	if c.calendarIndex >= len(c.calendars) {
		return len(c.calendars) - 1
	}
	return c.calendarIndex
}

func newCalendarList(calendars []calendarstore.CalendarMeta, selected, width, height int) list.Model {
	items := make([]list.Item, 0, len(calendars))
	for _, cal := range calendars {
		items = append(items, calendarListItem{meta: cal})
	}
	return newSimpleList(items, calendarListDelegate{}, selected, width, height)
}

type calendarListDelegate struct{ simpleDelegate }

func (d calendarListDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	calendarItem, ok := item.(calendarListItem)
	if !ok {
		return
	}
	cal := calendarItem.meta
	cursor := listCursor(index == m.Index())
	line := fmt.Sprintf("%s%s  %s  %s  pos:%d  %s", cursor, cal.Name, cal.Color, cal.TimeZone, cal.Position, cal.Source)
	_, _ = io.WriteString(w, padRight(line, m.Width()))
}

func (c Calendar) agendaCalendarMarker(calendarID int64) string {
	cal, ok := c.calendarByID(calendarID)
	label := calendarRowLabel(calendarID, cal, ok)
	color := strings.TrimSpace(cal.Color)
	marker := "##"
	if color != "" {
		marker = lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(marker)
	}
	return marker + " " + label
}

func (c Calendar) calendarByID(calendarID int64) (calendarstore.CalendarMeta, bool) {
	for _, cal := range c.calendars {
		if cal.RemoteID == calendarID {
			return cal, true
		}
	}
	return calendarstore.CalendarMeta{}, false
}

func calendarRowLabel(calendarID int64, cal calendarstore.CalendarMeta, ok bool) string {
	name := ""
	if ok {
		name = strings.TrimSpace(cal.Name)
	}
	if name != "" {
		return name
	}
	if calendarID > 0 {
		return "calendar:" + strconv.FormatInt(calendarID, 10)
	}
	return "calendar:-"
}

func calendarDetailName(calendarID int64, cal calendarstore.CalendarMeta, ok bool) string {
	name := ""
	if ok {
		name = strings.TrimSpace(cal.Name)
	}
	if name != "" {
		return name
	}
	if calendarID > 0 {
		return "#" + strconv.FormatInt(calendarID, 10)
	}
	return "-"
}

func calendarDetailColor(cal calendarstore.CalendarMeta, ok bool) string {
	if !ok {
		return "-"
	}
	return emptyDash(cal.Color)
}

func calendarDetailTimeZone(cal calendarstore.CalendarMeta, ok bool, fallback string) string {
	if ok && strings.TrimSpace(cal.TimeZone) != "" {
		return strings.TrimSpace(cal.TimeZone)
	}
	return emptyDash(fallback)
}
