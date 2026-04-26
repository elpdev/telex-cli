package screens

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/calendar"
	"github.com/elpdev/telex-cli/internal/calendarstore"
)

func TestCalendarBackFromCalendarsReturnsToAgenda(t *testing.T) {
	screen := NewCalendar(calendarstore.New(t.TempDir()), nil)
	updated, cmd := screen.Update(CalendarActionMsg{Action: "view-calendars"})
	if cmd != nil {
		t.Fatal("expected no command")
	}

	updated, cmd = updated.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if cmd != nil {
		t.Fatal("expected no command")
	}

	calendar, ok := updated.(Calendar)
	if !ok {
		t.Fatalf("updated = %T", updated)
	}
	if calendar.mode != calendarViewAgenda {
		t.Fatalf("mode = %v, want agenda", calendar.mode)
	}
	if calendar.status != "Showing agenda" {
		t.Fatalf("status = %q", calendar.status)
	}
}

func TestCalendarAgendaFilterMatchesCalendarStatusSourceAndText(t *testing.T) {
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	screen := Calendar{
		allItems: []calendarstore.OccurrenceMeta{
			{EventID: 1, CalendarID: 10, Title: "Planning", Location: "Room A", StartsAt: startsAt, Status: "confirmed", Source: "telex"},
			{EventID: 2, CalendarID: 20, Title: "Lunch", Location: "Cafe", StartsAt: startsAt.Add(time.Hour), Status: "tentative", Source: "ics"},
			{EventID: 3, CalendarID: 10, Title: "Roadmap", Location: "Room B", StartsAt: startsAt.Add(2 * time.Hour), Status: "cancelled", Source: "telex"},
		},
		calendars: []calendarstore.CalendarMeta{{RemoteID: 10, Name: "Work"}, {RemoteID: 20, Name: "Personal"}},
	}
	screen.filter = parseCalendarAgendaFilter("calendar:work status:conf source:tel planning room")
	screen.applyAgendaFilter()

	if len(screen.items) != 1 || screen.items[0].EventID != 1 {
		t.Fatalf("items = %#v", screen.items)
	}
}

func TestCalendarAgendaFilterSourceFallsBackToCachedEvent(t *testing.T) {
	store := calendarstore.New(t.TempDir())
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	event := calendar.CalendarEvent{ID: 9, CalendarID: 1, Title: "Imported", StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Source: "ics"}
	if err := store.StoreEvent(event, startsAt); err != nil {
		t.Fatal(err)
	}
	screen := Calendar{store: store, allItems: []calendarstore.OccurrenceMeta{{EventID: 9, CalendarID: 1, Title: "Imported", StartsAt: startsAt}}}
	screen.filter = parseCalendarAgendaFilter("source:ics")
	screen.applyAgendaFilter()

	if len(screen.items) != 1 || screen.items[0].EventID != 9 {
		t.Fatalf("items = %#v", screen.items)
	}
}

func TestCalendarAgendaFilterModeAppliesAndRendersActiveFilters(t *testing.T) {
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	screen := Calendar{
		allItems:  []calendarstore.OccurrenceMeta{{EventID: 1, CalendarID: 10, Title: "Planning", Location: "Room A", StartsAt: startsAt, Status: "confirmed"}, {EventID: 2, CalendarID: 20, Title: "Lunch", Location: "Cafe", StartsAt: startsAt.Add(time.Hour), Status: "tentative"}},
		calendars: []calendarstore.CalendarMeta{{RemoteID: 10, Name: "Work"}, {RemoteID: 20, Name: "Personal"}},
		keys:      DefaultCalendarKeyMap(),
	}
	screen.applyAgendaFilter()
	updated, _ := screen.Update(tea.KeyPressMsg(tea.Key{Text: "/", Code: '/'}))
	for _, r := range "calendar:work planning" {
		updated, _ = updated.Update(tea.KeyPressMsg(tea.Key{Text: string(r), Code: r}))
	}
	updated, _ = updated.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Calendar)

	if len(screen.items) != 1 || screen.items[0].EventID != 1 {
		t.Fatalf("items = %#v", screen.items)
	}
	view := screen.View(80, 20)
	for _, want := range []string{"Filters: calendar=work text=\"planning\" (1/2)", "Planning"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func TestCalendarAgendaRowsShowCalendarMarkerAndFallback(t *testing.T) {
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	screen := Calendar{
		items: []calendarstore.OccurrenceMeta{
			{EventID: 1, CalendarID: 10, Title: "Planning", StartsAt: startsAt, Status: "confirmed"},
			{EventID: 2, CalendarID: 20, Title: "Lunch", StartsAt: startsAt.Add(time.Hour), Status: "tentative"},
		},
		calendars: []calendarstore.CalendarMeta{{RemoteID: 10, Name: "Work", Color: "#22c55e"}},
		keys:      DefaultCalendarKeyMap(),
	}

	view := screen.View(100, 20)
	for _, want := range []string{"Work", "Planning", "## calendar:20", "Lunch"} {
		if !strings.Contains(view, want) {
			t.Fatalf("agenda view missing %q:\n%s", want, view)
		}
	}
}

func TestCalendarRangeNavigationReloadsCachedWindow(t *testing.T) {
	store := calendarstore.New(t.TempDir())
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	events := []calendar.CalendarOccurrence{
		{StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Event: calendar.CalendarEvent{ID: 1, CalendarID: 10, Title: "Current"}},
		{StartsAt: startsAt.AddDate(0, 0, 30), EndsAt: startsAt.AddDate(0, 0, 30).Add(time.Hour), Event: calendar.CalendarEvent{ID: 2, CalendarID: 10, Title: "Next"}},
	}
	if err := store.StoreOccurrences(events, startsAt); err != nil {
		t.Fatal(err)
	}
	screen := Calendar{store: store, rangeStart: time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC), rangeEnd: time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC), keys: DefaultCalendarKeyMap()}

	updated, cmd := screen.Update(CalendarActionMsg{Action: "next-range"})
	if cmd == nil {
		t.Fatal("expected range change to reload cache")
	}
	updated, _ = updated.Update(cmd())
	screen = updated.(Calendar)
	if len(screen.items) != 1 || screen.items[0].Title != "Next" {
		t.Fatalf("items = %#v", screen.items)
	}
	if got := screen.rangeLabel(); got != "May 25, 2026 to Jun 23, 2026" {
		t.Fatalf("range label = %q", got)
	}
}

func TestCalendarSyncUsesActiveRange(t *testing.T) {
	var gotFrom, gotTo string
	screen := Calendar{
		sync: func(_ context.Context, from, to string) (CalendarSyncResult, error) {
			gotFrom = from
			gotTo = to
			return CalendarSyncResult{}, nil
		},
		rangeStart: time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC),
		rangeEnd:   time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC),
		keys:       DefaultCalendarKeyMap(),
	}

	updated, cmd := screen.Update(CalendarActionMsg{Action: "sync"})
	if cmd == nil {
		t.Fatal("expected sync command")
	}
	updated, _ = updated.Update(cmd())
	_ = updated.(Calendar)
	if gotFrom != "2026-04-25" || gotTo != "2026-05-25" {
		t.Fatalf("range = %q to %q", gotFrom, gotTo)
	}
}

func TestCalendarAgendaClearFilterRestoresItems(t *testing.T) {
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	screen := Calendar{allItems: []calendarstore.OccurrenceMeta{{EventID: 1, Title: "Planning", StartsAt: startsAt}, {EventID: 2, Title: "Lunch", StartsAt: startsAt.Add(time.Hour)}}, keys: DefaultCalendarKeyMap()}
	screen.filter = parseCalendarAgendaFilter("planning")
	screen.applyAgendaFilter()
	updated, _ := screen.Update(CalendarActionMsg{Action: "clear-filter"})
	screen = updated.(Calendar)

	if screen.filter.active() || len(screen.items) != 2 {
		t.Fatalf("filter = %#v items = %#v", screen.filter, screen.items)
	}
}

func TestCalendarDetailShowsLinkedMessages(t *testing.T) {
	store := calendarstore.New(t.TempDir())
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	event := calendar.CalendarEvent{
		ID:         9,
		CalendarID: 1,
		Title:      "Planning",
		StartsAt:   startsAt,
		EndsAt:     startsAt.Add(time.Hour),
		Messages: []calendar.MessageSummary{{
			ID:            42,
			InboxID:       7,
			Subject:       "Re: Planning",
			SenderDisplay: "Alex <alex@example.com>",
			PreviewText:   "Agenda attached",
			ReceivedAt:    startsAt.Add(-time.Hour),
			SystemState:   "inbox",
		}},
	}
	if err := store.StoreEvent(event, startsAt); err != nil {
		t.Fatal(err)
	}
	if err := store.StoreOccurrences([]calendar.CalendarOccurrence{{StartsAt: event.StartsAt, EndsAt: event.EndsAt, Event: event}}, startsAt); err != nil {
		t.Fatal(err)
	}

	screen := Calendar{store: store, items: []calendarstore.OccurrenceMeta{{EventID: event.ID, CalendarID: event.CalendarID, Title: event.Title, StartsAt: event.StartsAt, EndsAt: event.EndsAt}}}
	view := screen.detailView()
	for _, want := range []string{"Messages: 1", "Re: Planning", "Alex <alex@example.com>", "2026-04-25 13:00", "inbox:7", "inbox", "Agenda attached"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail view missing %q:\n%s", want, view)
		}
	}
}

func TestCalendarDetailShowsCalendarMetadata(t *testing.T) {
	store := calendarstore.New(t.TempDir())
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	cal := calendar.Calendar{ID: 10, Name: "Work", Color: "#22c55e", TimeZone: "America/New_York"}
	event := calendar.CalendarEvent{ID: 9, CalendarID: cal.ID, Title: "Planning", StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), TimeZone: "UTC"}
	if err := store.StoreCalendar(cal, startsAt); err != nil {
		t.Fatal(err)
	}
	if err := store.StoreEvent(event, startsAt); err != nil {
		t.Fatal(err)
	}
	calendars, err := store.ListCalendars()
	if err != nil {
		t.Fatal(err)
	}

	screen := Calendar{store: store, calendars: calendars, items: []calendarstore.OccurrenceMeta{{EventID: event.ID, CalendarID: event.CalendarID, Title: event.Title, StartsAt: event.StartsAt, EndsAt: event.EndsAt}}}
	view := screen.detailView()
	for _, want := range []string{"Calendar: Work", "Calendar color: #22c55e", "Calendar time zone: America/New_York", "Event time zone: UTC"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail view missing %q:\n%s", want, view)
		}
	}
}

func TestCalendarDetailCalendarMetadataFallbacks(t *testing.T) {
	store := calendarstore.New(t.TempDir())
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	event := calendar.CalendarEvent{ID: 9, CalendarID: 10, Title: "Planning", StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), TimeZone: "UTC"}
	if err := store.StoreEvent(event, startsAt); err != nil {
		t.Fatal(err)
	}

	screen := Calendar{store: store, items: []calendarstore.OccurrenceMeta{{EventID: event.ID, CalendarID: event.CalendarID, Title: event.Title, StartsAt: event.StartsAt, EndsAt: event.EndsAt}}}
	view := screen.detailView()
	for _, want := range []string{"Calendar: #10", "Calendar color: -", "Calendar time zone: UTC"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail view missing %q:\n%s", want, view)
		}
	}
}

func TestCalendarDetailUsesCachedFullEvent(t *testing.T) {
	store := calendarstore.New(t.TempDir())
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	event := calendar.CalendarEvent{
		ID:                9,
		CalendarID:        1,
		Title:             "Planning",
		Description:       "Discuss roadmap\nConfirm launch date",
		Location:          "Room A",
		StartsAt:          startsAt,
		EndsAt:            startsAt.Add(time.Hour),
		Status:            "confirmed",
		OrganizerName:     "Alex",
		OrganizerEmail:    "alex@example.com",
		RecurrenceSummary: "Weekly on Friday",
		RecurrenceRule:    "FREQ=WEEKLY;BYDAY=FR",
		RecurrenceExceptions: []string{
			"2026-05-01",
		},
		NextOccurrences: []time.Time{
			startsAt.Add(7 * 24 * time.Hour),
			startsAt.Add(14 * 24 * time.Hour),
		},
		Attendees: []calendar.CalendarEventAttendee{{
			Email:               "leo@example.com",
			Name:                "Leo",
			Role:                "required",
			ParticipationStatus: "accepted",
			ResponseRequested:   true,
		}},
		Links: []calendar.CalendarEventLink{{MessageID: 42, ICalUID: "uid-1", ICalMethod: "REQUEST", SequenceNumber: 3}},
		Messages: []calendar.MessageSummary{{
			ID:            42,
			InboxID:       7,
			Subject:       "Planning invite",
			SenderDisplay: "Alex <alex@example.com>",
			PreviewText:   "Please RSVP",
			ReceivedAt:    startsAt.Add(-time.Hour),
			SystemState:   "inbox",
		}},
	}
	if err := store.StoreEvent(event, startsAt); err != nil {
		t.Fatal(err)
	}

	screen := Calendar{store: store, items: []calendarstore.OccurrenceMeta{{EventID: event.ID, CalendarID: event.CalendarID, Title: "Occurrence title", StartsAt: event.StartsAt, EndsAt: event.EndsAt}}}
	view := screen.detailView()
	for _, want := range []string{"Planning", "Description:", "Discuss roadmap", "Confirm launch date", "Organizer: Alex <alex@example.com>", "Recurrence:", "Summary: Weekly on Friday", "Rule: FREQ=WEEKLY;BYDAY=FR", "Next occurrences: 2", "2026-05-02 14:00", "2026-05-09 14:00", "Exceptions: 1", "2026-05-01", "Attendees: 1", "Leo <leo@example.com> | role:required | status:accepted", "Links: 1", "message:42 | uid:uid-1 | method:REQUEST | sequence:3", "Messages: 1", "Planning invite", "Please RSVP"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail view missing %q:\n%s", want, view)
		}
	}
}

func TestCalendarDetailOmitsRecurrenceForNonRecurringEvent(t *testing.T) {
	store := calendarstore.New(t.TempDir())
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	event := calendar.CalendarEvent{ID: 9, CalendarID: 1, Title: "Planning", StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour)}
	if err := store.StoreEvent(event, startsAt); err != nil {
		t.Fatal(err)
	}

	screen := Calendar{store: store, items: []calendarstore.OccurrenceMeta{{EventID: event.ID, CalendarID: event.CalendarID, Title: event.Title, StartsAt: event.StartsAt, EndsAt: event.EndsAt}}}
	view := screen.detailView()
	if strings.Contains(view, "Recurrence:") {
		t.Fatalf("detail view should omit empty recurrence section:\n%s", view)
	}
}

func TestCalendarDetailShowsInvitationMetadata(t *testing.T) {
	store := calendarstore.New(t.TempDir())
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	event := calendar.CalendarEvent{
		ID:             9,
		CalendarID:     1,
		Title:          "Planning",
		StartsAt:       startsAt,
		EndsAt:         startsAt.Add(time.Hour),
		Invitation:     true,
		OrganizerName:  "Alex",
		OrganizerEmail: "alex@example.com",
		Attendees: []calendar.CalendarEventAttendee{{
			Email:               "leo@example.com",
			Name:                "Leo",
			ParticipationStatus: "tentative",
			ResponseRequested:   true,
		}},
		Links: []calendar.CalendarEventLink{{MessageID: 42, ICalMethod: "REQUEST", SequenceNumber: 3}},
	}
	if err := store.StoreEvent(event, startsAt); err != nil {
		t.Fatal(err)
	}
	if err := store.StoreOccurrences([]calendar.CalendarOccurrence{{StartsAt: event.StartsAt, EndsAt: event.EndsAt, Event: event}}, startsAt); err != nil {
		t.Fatal(err)
	}

	screen := Calendar{store: store, items: []calendarstore.OccurrenceMeta{{EventID: event.ID, CalendarID: event.CalendarID, Title: event.Title, StartsAt: event.StartsAt, EndsAt: event.EndsAt}}}
	view := screen.detailView()
	for _, want := range []string{"Invitation: true", "Organizer: Alex <alex@example.com>", "Attendees: 1", "Leo <leo@example.com> | role:- | status:tentative", "Links: 1", "message:42 | uid:- | method:REQUEST | sequence:3"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail view missing %q:\n%s", want, view)
		}
	}
}

func TestCalendarInvitationShowLoadsDetails(t *testing.T) {
	store := calendarstore.New(t.TempDir())
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	event := calendar.CalendarEvent{ID: 9, CalendarID: 1, Title: "Planning", StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Links: []calendar.CalendarEventLink{{MessageID: 42}}}
	if err := store.StoreEvent(event, startsAt); err != nil {
		t.Fatal(err)
	}
	if err := store.StoreOccurrences([]calendar.CalendarOccurrence{{StartsAt: event.StartsAt, EndsAt: event.EndsAt, Event: event}}, startsAt); err != nil {
		t.Fatal(err)
	}
	var gotMessageID int64
	screen := Calendar{
		store: store,
		items: []calendarstore.OccurrenceMeta{{EventID: event.ID, CalendarID: event.CalendarID, Title: event.Title, StartsAt: event.StartsAt, EndsAt: event.EndsAt}},
		showInvite: func(_ context.Context, messageID int64) (*calendar.Invitation, error) {
			gotMessageID = messageID
			return &calendar.Invitation{MessageID: messageID, Available: true, CalendarEvent: &event, CurrentUserAttendee: &calendar.CalendarEventAttendee{ParticipationStatus: "accepted"}}, nil
		},
	}

	updated, cmd := screen.Update(CalendarActionMsg{Action: "invitation-show"})
	if cmd == nil {
		t.Fatal("expected command")
	}
	updated, _ = updated.Update(cmd())
	screen = updated.(Calendar)
	if gotMessageID != 42 {
		t.Fatalf("message id = %d", gotMessageID)
	}
	if screen.invitation == nil || screen.invitation.MessageID != 42 || !screen.detail {
		t.Fatalf("screen = %#v", screen)
	}
	view := screen.detailView()
	for _, want := range []string{"Invitation details:", "Message ID: 42", "Current response: accepted"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail view missing %q:\n%s", want, view)
		}
	}
}

func TestCalendarInvitationRespondUsesParticipationStatus(t *testing.T) {
	store := calendarstore.New(t.TempDir())
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	event := calendar.CalendarEvent{ID: 9, CalendarID: 1, Title: "Planning", StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Links: []calendar.CalendarEventLink{{MessageID: 42}}}
	if err := store.StoreEvent(event, startsAt); err != nil {
		t.Fatal(err)
	}
	if err := store.StoreOccurrences([]calendar.CalendarOccurrence{{StartsAt: event.StartsAt, EndsAt: event.EndsAt, Event: event}}, startsAt); err != nil {
		t.Fatal(err)
	}
	var gotStatus string
	screen := Calendar{
		store: store,
		items: []calendarstore.OccurrenceMeta{{EventID: event.ID, CalendarID: event.CalendarID, Title: event.Title, StartsAt: event.StartsAt, EndsAt: event.EndsAt}},
		respondInvite: func(_ context.Context, _ int64, input calendar.InvitationInput) (*calendar.Invitation, error) {
			gotStatus = input.ParticipationStatus
			return &calendar.Invitation{MessageID: 42, Available: true, CalendarEvent: &event, CurrentUserAttendee: &calendar.CalendarEventAttendee{ParticipationStatus: input.ParticipationStatus}}, nil
		},
	}

	updated, cmd := screen.Update(CalendarActionMsg{Action: "invitation-declined"})
	if cmd == nil {
		t.Fatal("expected command")
	}
	updated, _ = updated.Update(cmd())
	screen = updated.(Calendar)
	if gotStatus != "declined" || screen.status != "Responded declined" {
		t.Fatalf("status = %q screen status = %q", gotStatus, screen.status)
	}
}

func TestCalendarDetailShowsEmptyLinkedMessageState(t *testing.T) {
	store := calendarstore.New(t.TempDir())
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	event := calendar.CalendarEvent{ID: 9, CalendarID: 1, Title: "Planning", StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour)}
	if err := store.StoreEvent(event, startsAt); err != nil {
		t.Fatal(err)
	}

	screen := Calendar{store: store, items: []calendarstore.OccurrenceMeta{{EventID: event.ID, CalendarID: event.CalendarID, Title: event.Title, StartsAt: event.StartsAt, EndsAt: event.EndsAt}}}
	view := screen.detailView()
	if !strings.Contains(view, "Messages: none") {
		t.Fatalf("detail view missing empty state:\n%s", view)
	}
}

func TestCalendarDetailGracefullyFallsBackWhenCachedEventMissing(t *testing.T) {
	startsAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	screen := Calendar{store: calendarstore.New(t.TempDir()), items: []calendarstore.OccurrenceMeta{{EventID: 99, CalendarID: 1, Title: "Occurrence Planning", StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Status: "confirmed"}}}
	view := screen.detailView()
	for _, want := range []string{"Occurrence Planning", "Event ID: 99", "Cached event details: unavailable"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail view missing %q:\n%s", want, view)
		}
	}
}

func TestCalendarImportICSUsesSelectedCalendarAndShowsResult(t *testing.T) {
	store := calendarstore.New(t.TempDir())
	var gotCalendarID int64
	var gotPath string
	screen := Calendar{
		store:     store,
		mode:      calendarViewCalendars,
		calendars: []calendarstore.CalendarMeta{{RemoteID: 12, Name: "Work"}},
		importICS: func(_ context.Context, calendarID int64, path string) (*calendar.ImportResult, error) {
			gotCalendarID = calendarID
			gotPath = path
			return &calendar.ImportResult{Created: 1, Updated: 2, Skipped: 3, Failed: 4, Errors: []string{"bad event"}}, nil
		},
	}

	updated, cmd := screen.Update(CalendarActionMsg{Action: "import-ics"})
	if cmd != nil {
		t.Fatal("expected no command")
	}
	screen = updated.(Calendar)
	if !screen.filePickerOpen || screen.importCalendar != 12 {
		t.Fatalf("screen = %#v", screen)
	}

	path := "/tmp/import.ics"
	updated, cmd = screen.importSelectedICS(path)
	if cmd == nil {
		t.Fatal("expected import command")
	}
	updated, _ = updated.Update(cmd())
	screen = updated.(Calendar)
	if gotCalendarID != 12 || gotPath != path {
		t.Fatalf("import = %d %q", gotCalendarID, gotPath)
	}
	for _, want := range []string{"created 1", "updated 2", "skipped 3", "failed 4", "bad event"} {
		if !strings.Contains(screen.status, want) {
			t.Fatalf("status missing %q: %q", want, screen.status)
		}
	}
}

func TestCalendarImportICSRejectsNonICSFile(t *testing.T) {
	screen := Calendar{filePickerOpen: true, importCalendar: 12}
	updated, cmd := screen.importSelectedICS("/tmp/import.txt")
	if cmd != nil {
		t.Fatal("expected no command")
	}
	screen = updated.(Calendar)
	if !screen.filePickerOpen || screen.status != "Select an .ics file" {
		t.Fatalf("screen = %#v", screen)
	}
}

func TestCalendarEventInputFromFormValidatesRequiredFields(t *testing.T) {
	_, err := calendarEventInputFromForm(calendarEventFormData{CalendarID: "1", Title: "", StartDate: "2026-04-25", EndDate: "2026-04-25", StartTime: "09:00", EndTime: "10:00"})
	if err == nil {
		t.Fatal("expected missing title error")
	}

	_, err = calendarEventInputFromForm(calendarEventFormData{CalendarID: "1", Title: "Standup", StartDate: "2026-04-25", EndDate: "2026-04-25"})
	if err == nil {
		t.Fatal("expected missing time error")
	}

	_, err = calendarEventInputFromForm(calendarEventFormData{CalendarID: "1", Title: "Standup", StartDate: "2026-04-25", EndDate: "2026-04-25", StartTime: "11:00", EndTime: "10:00"})
	if err == nil {
		t.Fatal("expected invalid range error")
	}
}

func TestCalendarEventInputFromFormBuildsInput(t *testing.T) {
	input, err := calendarEventInputFromForm(calendarEventFormData{CalendarID: "42", Title: "Standup", Description: "Daily sync", Location: "Room A", StartDate: "2026-04-25", EndDate: "2026-04-25", StartTime: "09:00", EndTime: "10:00", TimeZone: "UTC", Status: "confirmed"})
	if err != nil {
		t.Fatal(err)
	}
	if input.CalendarID != 42 || input.Title != "Standup" || input.Description != "Daily sync" || input.Location != "Room A" {
		t.Fatalf("input = %#v", input)
	}
	if input.AllDay == nil || *input.AllDay {
		t.Fatalf("all day = %#v", input.AllDay)
	}
	if input.StartDate != "2026-04-25" || input.StartTime != "09:00" || input.EndDate != "2026-04-25" || input.EndTime != "10:00" {
		t.Fatalf("input = %#v", input)
	}
}

func TestCalendarEventInputFromFormAllowsAllDayWithoutTimes(t *testing.T) {
	input, err := calendarEventInputFromForm(calendarEventFormData{CalendarID: "42", Title: "Holiday", AllDay: true, StartDate: "2026-04-25", EndDate: "2026-04-26"})
	if err != nil {
		t.Fatal(err)
	}
	if input.AllDay == nil || !*input.AllDay {
		t.Fatalf("all day = %#v", input.AllDay)
	}
	if input.StartTime != "" || input.EndTime != "" {
		t.Fatalf("input = %#v", input)
	}
}

func TestCalendarInputFromFormValidatesRequiredFields(t *testing.T) {
	if _, err := calendarInputFromForm(calendarFormData{Name: ""}); err == nil {
		t.Fatal("expected missing name error")
	}
	if _, err := calendarInputFromForm(calendarFormData{Name: "Work", TimeZone: "Not/AZone"}); err == nil {
		t.Fatal("expected invalid time zone error")
	}
	if _, err := calendarInputFromForm(calendarFormData{Name: "Work", Position: "zero"}); err == nil {
		t.Fatal("expected invalid position error")
	}
}

func TestCalendarInputFromFormBuildsInput(t *testing.T) {
	input, err := calendarInputFromForm(calendarFormData{Name: " Work ", Color: " green ", TimeZone: "UTC", Position: "2"})
	if err != nil {
		t.Fatal(err)
	}
	if input.Name != "Work" || input.Color != "green" || input.TimeZone != "UTC" {
		t.Fatalf("input = %#v", input)
	}
	if input.Position == nil || *input.Position != 2 {
		t.Fatalf("position = %#v", input.Position)
	}
}
