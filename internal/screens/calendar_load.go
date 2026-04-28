package screens

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/calendar"
	"github.com/elpdev/telex-cli/internal/calendarstore"
)

func (c Calendar) loadCmd() tea.Cmd {
	return func() tea.Msg { return c.load() }

}

func (c Calendar) load() calendarLoadedMsg {
	start, end := c.activeRange()
	items, err := c.store.ListOccurrencesRange(start, end)
	if err != nil {
		return calendarLoadedMsg{err: err}
	}
	calendars, err := c.store.ListCalendars()
	if err != nil {
		return calendarLoadedMsg{err: err}
	}
	events, err := c.store.ListEvents(0)
	if err != nil {
		return calendarLoadedMsg{err: err}
	}
	return calendarLoadedMsg{items: items, calendars: calendars, lastSynced: latestCalendarSync(items, calendars, events), cachedEvents: len(events)}
}

func (c Calendar) syncCmd() tea.Cmd {
	from, to := c.rangeDates()
	return func() tea.Msg {
		result, err := c.sync(context.Background(), from, to)
		loaded := c.load()
		return calendarSyncedMsg{result: result, loaded: loaded, syncErr: err}
	}
}

func latestCalendarSync(items []calendarstore.OccurrenceMeta, calendars []calendarstore.CalendarMeta, events []calendarstore.CachedEvent) time.Time {
	var latest time.Time
	for _, item := range items {
		if item.SyncedAt.After(latest) {
			latest = item.SyncedAt
		}
	}
	for _, cal := range calendars {
		if cal.SyncedAt.After(latest) {
			latest = cal.SyncedAt
		}
	}
	for _, event := range events {
		if event.Meta.SyncedAt.After(latest) {
			latest = event.Meta.SyncedAt
		}
	}
	return latest
}

func (c Calendar) deleteCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		err := c.deleteEvent(context.Background(), id)
		loaded := calendarLoadedMsg{}
		if err == nil {
			loaded = c.load()
			err = loaded.err
		}
		return calendarActionFinishedMsg{status: "Deleted event", loaded: loaded, err: err}
	}
}

func (c Calendar) deleteCalendarCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		err := c.deleteCalendar(context.Background(), id)
		loaded := calendarLoadedMsg{}
		if err == nil {
			loaded = c.load()
			err = loaded.err
		}
		return calendarActionFinishedMsg{status: "Deleted calendar", loaded: loaded, err: err}
	}
}

func (c Calendar) importICSCmd(calendarID int64, path string) tea.Cmd {
	return func() tea.Msg {
		result, err := c.importICS(context.Background(), calendarID, path)
		loaded := calendarLoadedMsg{}
		if err == nil {
			loaded = c.load()
			err = loaded.err
		}
		return calendarActionFinishedMsg{status: importICSStatus(result), loaded: loaded, err: err}
	}
}

func (c Calendar) invitationCmd(messageID int64, action, status string) tea.Cmd {
	return func() tea.Msg {
		var invite *calendar.Invitation
		var err error
		switch action {
		case "show":
			invite, err = c.showInvite(context.Background(), messageID)
		case "sync":
			invite, err = c.syncInvite(context.Background(), messageID)
		case "respond":
			invite, err = c.respondInvite(context.Background(), messageID, calendar.InvitationInput{ParticipationStatus: status})
		default:
			err = errors.New("unknown invitation action")
		}
		loaded := calendarLoadedMsg{}
		if err == nil {
			loaded = c.load()
			err = loaded.err
		}
		return calendarActionFinishedMsg{status: invitationActionStatus(action, status, invite), loaded: loaded, invitation: invite, err: err}
	}
}

func invitationStatusFromAction(action string) string {
	switch action {
	case "invitation-accepted":
		return "accepted"
	case "invitation-tentative":
		return "tentative"
	case "invitation-declined":
		return "declined"
	case "invitation-needs-action":
		return "needs_action"
	default:
		return ""
	}
}

func invitationActionStatus(action, status string, invite *calendar.Invitation) string {
	switch action {
	case "show":
		return "Loaded invitation details"
	case "sync":
		return "Synced invitation into Calendar"
	case "respond":
		if status != "" {
			return "Responded " + status
		}
	}
	if invite != nil && invite.CalendarEvent != nil && invite.CalendarEvent.Title != "" {
		return invite.CalendarEvent.Title
	}
	return "Updated invitation"
}

func importICSStatus(result *calendar.ImportResult) string {
	if result == nil {
		return "Imported ICS"
	}
	status := fmt.Sprintf("Imported ICS: created %d, updated %d, skipped %d, failed %d", result.Created, result.Updated, result.Skipped, result.Failed)
	if len(result.Errors) > 0 {
		status += "; errors: " + strings.Join(result.Errors, "; ")
	}
	return status
}
