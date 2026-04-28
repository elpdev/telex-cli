package screens

import (
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/api"
	"github.com/elpdev/telex-cli/internal/calendarstore"
)

func NewCalendar(store calendarstore.Store, sync CalendarSyncFunc) Calendar {
	return Calendar{store: store, sync: sync, loading: true, keys: DefaultCalendarKeyMap(), calendarList: newCalendarList(nil, 0, 0, 0)}
}

func (c Calendar) WithActions(create CreateCalendarEventFunc, update UpdateCalendarEventFunc, delete DeleteCalendarEventFunc) Calendar {
	c.createEvent = create
	c.updateEvent = update
	c.deleteEvent = delete
	return c
}

func (c Calendar) WithCalendarActions(create CreateCalendarFunc, update UpdateCalendarFunc, delete DeleteCalendarFunc) Calendar {
	c.createCalendar = create
	c.updateCalendar = update
	c.deleteCalendar = delete
	return c
}

func (c Calendar) WithImportICS(importICS ImportICSFunc) Calendar {
	c.importICS = importICS
	return c
}

func (c Calendar) WithInvitationActions(show ShowInvitationFunc, sync SyncInvitationFunc, respond RespondInvitationFunc) Calendar {
	c.showInvite = show
	c.syncInvite = sync
	c.respondInvite = respond
	return c
}

func (c Calendar) Init() tea.Cmd { return c.loadCmd() }

func (c Calendar) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if c.loading {
		return style.Render("Loading local calendar cache...")
	}
	if c.form != nil {
		return style.Render(c.form.WithWidth(max(40, width-4)).WithHeight(max(8, height-3)).View())
	}
	if c.filePickerOpen {
		return style.Render("Calendar / Import ICS\n" + c.status + "\n\n" + c.filePicker.View(width, max(1, height-3)))
	}
	if c.err != nil {
		return style.Render(calendarCacheErrorView(c.err))
	}
	var b strings.Builder
	b.WriteString("Calendar / " + c.modeTitle() + "\n")
	if c.status != "" {
		b.WriteString(c.status + "\n")
	}
	if c.mode == calendarViewAgenda {
		b.WriteString("Range: " + c.rangeLabel() + "\n")
	}
	if cache := c.cacheStatusLine(); cache != "" {
		b.WriteString(cache + "\n")
	}
	if c.syncErr != nil {
		b.WriteString(calendarRemoteErrorStatus(c.syncErr) + "\n")
	}
	if c.syncing {
		b.WriteString("Syncing remote Calendar...\n")
	}
	if c.mode == calendarViewAgenda && c.filtering {
		b.WriteString("Filter: " + c.filterInput + "\n")
		b.WriteString("Hint: calendar:<name|id> status:<value> source:<value> text terms\n")
	} else if c.mode == calendarViewAgenda && c.filter.active() {
		b.WriteString(fmt.Sprintf("Filters: %s (%d/%d)\n", c.filter.summary(), len(c.items), len(c.agendaSourceItems())))
	}
	if c.confirm != "" {
		b.WriteString(c.confirm + " [y/N]\n")
	}
	b.WriteString("\n")
	if c.mode == calendarViewCalendars {
		b.WriteString(c.calendarListView(width, max(1, height-8)))
		return style.Render(b.String())
	}
	if c.detail {
		b.WriteString(c.detailView())
		return style.Render(b.String())
	}
	if len(c.items) == 0 && c.filter.active() {
		b.WriteString("No calendar occurrences match the active filters. Press ctrl+l to clear filters.\n")
		return style.Render(b.String())
	}
	if len(c.items) == 0 {
		b.WriteString(c.emptyAgendaView())
		return style.Render(b.String())
	}
	for i, item := range c.items {
		cursor := listCursor(i == c.index)
		b.WriteString(fmt.Sprintf("%s%s  %s  %s  %s\n", cursor, item.StartsAt.Format("Jan 02 15:04"), c.agendaCalendarMarker(item.CalendarID), item.Title, item.Status))
	}
	return style.Render(b.String())
}

func (c Calendar) Title() string { return "Calendar" }

func (c Calendar) cacheStatusLine() string {
	if c.lastSynced.IsZero() {
		return ""
	}
	label := "Last synced: " + c.lastSynced.Format("2006-01-02 15:04")
	if time.Since(c.lastSynced) > 24*time.Hour {
		label += " (stale; press S to refresh)"
	}
	return label
}

func (c Calendar) emptyAgendaView() string {
	if len(c.calendars) == 0 {
		return "No calendars are cached. Press S to sync remote calendars, or press n to create a calendar.\n"
	}
	start, end := c.activeRange()
	if c.cachedEvents > 0 {
		return fmt.Sprintf("No events in this range (%s to %s). Press [ or ] to change range, t for today, or S to refresh.\n", start.Format("Jan 02, 2006"), end.AddDate(0, 0, -1).Format("Jan 02, 2006"))
	}
	return "Calendars are cached, but no events are cached yet. Press S to sync events for this range, or n to create an event.\n"
}

func calendarCacheErrorView(err error) string {
	return fmt.Sprintf("Calendar cache error: %v\n\nCheck that the local data directory is readable and writable, then press r to reload. Remote sync is available with S after the cache issue is fixed.", err)
}

func calendarRemoteErrorStatus(err error) string {
	if err == nil {
		return ""
	}
	var apiErr *api.Error
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == 401 || apiErr.StatusCode == 403 {
			return fmt.Sprintf("Calendar sync failed: authentication was rejected (%s). Run `telex auth login`, then press S.", emptyDash(apiErr.Error()))
		}
		if apiErr.StatusCode >= 500 {
			return fmt.Sprintf("Calendar sync failed: remote server error (%s). Cached data is still shown; press S to retry later.", emptyDash(apiErr.Error()))
		}
		return fmt.Sprintf("Calendar sync failed: remote API returned %d (%s). Cached data is still shown; press S to retry.", apiErr.StatusCode, emptyDash(apiErr.Error()))
	}
	message := err.Error()
	if strings.Contains(strings.ToLower(message), "config") || strings.Contains(strings.ToLower(message), "base url") || strings.Contains(strings.ToLower(message), "client") || strings.Contains(strings.ToLower(message), "secret") {
		return fmt.Sprintf("Calendar sync failed: configuration problem (%s). Open Settings and verify your Telex instance, then press S.", message)
	}
	return fmt.Sprintf("Calendar sync failed: %v. Cached data is still shown; press S to retry.", err)
}

func (c Calendar) CapturesFocusKey(tea.KeyPressMsg) bool {
	return c.form != nil || c.filePickerOpen || c.filtering
}

func (c Calendar) Selection() CalendarSelection {
	if c.mode == calendarViewCalendars {
		item, ok := c.selectedCalendar()
		if !ok {
			return CalendarSelection{Kind: "calendar", HasItem: false}
		}
		return CalendarSelection{Kind: "calendar", Subject: item.Name, HasItem: true}
	}
	item, ok := c.selected()
	if !ok {
		return CalendarSelection{Kind: "calendar-event", HasItem: false}
	}
	selection := CalendarSelection{Kind: "calendar-event", Subject: item.Title, HasItem: true}
	if c.selectedInvitationMessageID() > 0 {
		selection.HasInvitation = true
	}
	return selection
}
