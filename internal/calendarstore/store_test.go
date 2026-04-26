package calendarstore

import (
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/calendar"
)

func TestStoreCalendarEventAndOccurrencesUnderCalendarRoot(t *testing.T) {
	store := New(t.TempDir())
	syncedAt := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	cal := calendar.Calendar{ID: 1, UserID: 2, Name: "Work", Color: "cyan", TimeZone: "UTC", Position: 1, Source: "local", CreatedAt: syncedAt, UpdatedAt: syncedAt}
	if err := store.StoreCalendar(cal, syncedAt); err != nil {
		t.Fatal(err)
	}
	cals, err := store.ListCalendars()
	if err != nil {
		t.Fatal(err)
	}
	if len(cals) != 1 || cals[0].RemoteID != 1 || cals[0].Name != "Work" {
		t.Fatalf("calendars = %#v", cals)
	}

	event := calendar.CalendarEvent{ID: 9, CalendarID: 1, Title: "Standup", Description: "Daily sync", StartsAt: syncedAt, EndsAt: syncedAt.Add(30 * time.Minute), Status: "confirmed", RecurrenceRule: "FREQ=DAILY", RecurrenceSummary: "Daily", RecurrenceExceptions: []string{"2026-04-26"}, NextOccurrences: []time.Time{syncedAt.Add(24 * time.Hour)}, Messages: []calendar.MessageSummary{{ID: 42, InboxID: 7, Subject: "Re: Standup", SenderDisplay: "Alex", ReceivedAt: syncedAt.Add(-time.Hour), SystemState: "inbox"}}}
	if err := store.StoreEvent(event, syncedAt); err != nil {
		t.Fatal(err)
	}
	events, err := store.ListEvents(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Meta.RemoteID != 9 || events[0].Description != "Daily sync" {
		t.Fatalf("events = %#v", events)
	}
	if len(events[0].Meta.Messages) != 1 || events[0].Meta.Messages[0].ID != 42 || events[0].Meta.Messages[0].Subject != "Re: Standup" {
		t.Fatalf("messages = %#v", events[0].Meta.Messages)
	}
	if events[0].Meta.RecurrenceRule != "FREQ=DAILY" || events[0].Meta.RecurrenceSummary != "Daily" || len(events[0].Meta.RecurrenceExceptions) != 1 || len(events[0].Meta.NextOccurrences) != 1 {
		t.Fatalf("recurrence = %#v", events[0].Meta)
	}
	if want := store.CalendarRoot(); events[0].Path[:len(want)] != want {
		t.Fatalf("event path = %q, want under %q", events[0].Path, want)
	}

	occurrences := []calendar.CalendarOccurrence{{StartsAt: event.StartsAt, EndsAt: event.EndsAt, Event: event}}
	if err := store.StoreOccurrences(occurrences, syncedAt); err != nil {
		t.Fatal(err)
	}
	cached, err := store.ListOccurrences()
	if err != nil {
		t.Fatal(err)
	}
	if len(cached) != 1 || cached[0].EventID != 9 || cached[0].CalendarID != 1 {
		t.Fatalf("occurrences = %#v", cached)
	}
}

func TestDeleteCalendarRemovesCalendarAndCachedEvents(t *testing.T) {
	store := New(t.TempDir())
	syncedAt := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	cal := calendar.Calendar{ID: 1, Name: "Work", TimeZone: "UTC"}
	if err := store.StoreCalendar(cal, syncedAt); err != nil {
		t.Fatal(err)
	}
	if err := store.StoreEvent(calendar.CalendarEvent{ID: 9, CalendarID: 1, Title: "Standup"}, syncedAt); err != nil {
		t.Fatal(err)
	}
	if err := store.StoreEvent(calendar.CalendarEvent{ID: 10, CalendarID: 2, Title: "Other"}, syncedAt); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteCalendar(1); err != nil {
		t.Fatal(err)
	}
	calendars, err := store.ListCalendars()
	if err != nil {
		t.Fatal(err)
	}
	if len(calendars) != 0 {
		t.Fatalf("calendars = %#v", calendars)
	}
	events, err := store.ListEvents(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Meta.RemoteID != 10 {
		t.Fatalf("events = %#v", events)
	}
}

func TestListOccurrencesRangeFiltersCachedOccurrences(t *testing.T) {
	store := New(t.TempDir())
	syncedAt := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	event := calendar.CalendarEvent{ID: 9, CalendarID: 1, Title: "Standup"}
	occurrences := []calendar.CalendarOccurrence{
		{StartsAt: syncedAt.AddDate(0, 0, -1), EndsAt: syncedAt.AddDate(0, 0, -1).Add(time.Hour), Event: event},
		{StartsAt: syncedAt, EndsAt: syncedAt.Add(time.Hour), Event: event},
		{StartsAt: syncedAt.AddDate(0, 0, 7), EndsAt: syncedAt.AddDate(0, 0, 7).Add(time.Hour), Event: event},
	}
	if err := store.StoreOccurrences(occurrences, syncedAt); err != nil {
		t.Fatal(err)
	}

	cached, err := store.ListOccurrencesRange(syncedAt, syncedAt.AddDate(0, 0, 7))
	if err != nil {
		t.Fatal(err)
	}
	if len(cached) != 1 || !cached[0].StartsAt.Equal(syncedAt) {
		t.Fatalf("occurrences = %#v", cached)
	}
}
