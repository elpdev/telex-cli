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
	for _, want := range []string{"Planning", "Description:", "Discuss roadmap", "Confirm launch date", "Organizer: Alex <alex@example.com>", "Recurrence:", "Summary: Weekly on Friday", "Rule: FREQ=WEEKLY;BYDAY=FR", "Attendees: 1", "Leo <leo@example.com> | role:required | status:accepted", "Links: 1", "message:42 | uid:uid-1 | method:REQUEST | sequence:3", "Messages: 1", "Planning invite", "Please RSVP"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail view missing %q:\n%s", want, view)
		}
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
