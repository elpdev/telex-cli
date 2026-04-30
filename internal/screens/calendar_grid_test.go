package screens

import (
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/calendarstore"
)

func TestMonthGridDaysMondayStart(t *testing.T) {
	anchor := time.Date(2026, time.April, 29, 12, 0, 0, 0, time.UTC)
	grid := monthGridDays(anchor, time.Monday)

	if got, want := grid[0][0], time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("grid[0][0] = %s, want %s", got, want)
	}
	if got, want := grid[5][6], time.Date(2026, time.May, 10, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("grid[5][6] = %s, want %s", got, want)
	}
	if grid[0][0].Weekday() != time.Monday {
		t.Errorf("grid[0][0].Weekday() = %s, want Monday", grid[0][0].Weekday())
	}

	prev := grid[0][0]
	for r := 0; r < 6; r++ {
		for d := 0; d < 7; d++ {
			if r == 0 && d == 0 {
				continue
			}
			diff := grid[r][d].Sub(prev)
			if diff != 24*time.Hour {
				t.Fatalf("non-monotonic at [%d][%d]: %s -> %s", r, d, prev, grid[r][d])
			}
			prev = grid[r][d]
		}
	}
}

func TestMonthGridDaysSundayStart(t *testing.T) {
	anchor := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	grid := monthGridDays(anchor, time.Sunday)
	if grid[0][0].Weekday() != time.Sunday {
		t.Errorf("grid[0][0].Weekday() = %s, want Sunday", grid[0][0].Weekday())
	}
	if got, want := grid[0][0], time.Date(2026, time.March, 29, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("grid[0][0] = %s, want %s", got, want)
	}
}

func TestStartOfWeekDSTSafe(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("America/New_York tz not available")
	}
	// 2026-03-08 is the spring-forward Sunday in the US.
	mid := time.Date(2026, time.March, 8, 12, 0, 0, 0, ny)
	got := startOfWeek(mid, time.Sunday)
	want := time.Date(2026, time.March, 8, 0, 0, 0, 0, ny)
	if !got.Equal(want) {
		t.Errorf("startOfWeek = %s, want %s", got, want)
	}
}

func TestPlaceWeekEventsHappyPath(t *testing.T) {
	weekStart := time.Date(2026, time.April, 27, 0, 0, 0, 0, time.UTC) // Monday
	occ := calendarstore.OccurrenceMeta{
		EventID:    1,
		CalendarID: 10,
		Title:      "Standup",
		StartsAt:   time.Date(2026, time.April, 27, 9, 0, 0, 0, time.UTC),
		EndsAt:     time.Date(2026, time.April, 27, 10, 30, 0, 0, time.UTC),
	}
	timed, allDay := placeWeekEvents([]calendarstore.OccurrenceMeta{occ}, weekStart, 2, 6, 22)
	if len(allDay[0]) != 0 {
		t.Errorf("expected no all-day, got %d", len(allDay[0]))
	}
	// 09:00 with fromH=6, slotsPerHour=2: slot index = (9-6)*2 = 6
	if timed[0][6].Title != "Standup" {
		t.Errorf("slot[6].Title = %q, want Standup", timed[0][6].Title)
	}
	if timed[0][6].Continuation {
		t.Errorf("slot[6] should be the head slot, not continuation")
	}
	if !timed[0][7].Continuation || timed[0][7].EventID != 1 {
		t.Errorf("slot[7] should be continuation of event 1, got %+v", timed[0][7])
	}
	if !timed[0][8].Continuation || timed[0][8].EventID != 1 {
		t.Errorf("slot[8] should be continuation of event 1, got %+v", timed[0][8])
	}
	if timed[0][9].EventID != 0 {
		t.Errorf("slot[9] should be empty, got %+v", timed[0][9])
	}
}

func TestPlaceWeekEventsOverlap(t *testing.T) {
	weekStart := time.Date(2026, time.April, 27, 0, 0, 0, 0, time.UTC)
	first := calendarstore.OccurrenceMeta{
		EventID:  1,
		Title:    "First",
		StartsAt: time.Date(2026, time.April, 27, 9, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, time.April, 27, 10, 0, 0, 0, time.UTC),
	}
	second := calendarstore.OccurrenceMeta{
		EventID:  2,
		Title:    "Second",
		StartsAt: time.Date(2026, time.April, 27, 9, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, time.April, 27, 10, 0, 0, 0, time.UTC),
	}
	timed, _ := placeWeekEvents([]calendarstore.OccurrenceMeta{first, second}, weekStart, 2, 6, 22)
	if timed[0][6].EventID != 1 {
		t.Errorf("expected first event kept, got EventID=%d", timed[0][6].EventID)
	}
	if timed[0][6].OverflowCount != 1 {
		t.Errorf("expected overflow=1, got %d", timed[0][6].OverflowCount)
	}
}

func TestPlaceWeekEventsAllDay(t *testing.T) {
	weekStart := time.Date(2026, time.April, 27, 0, 0, 0, 0, time.UTC)
	occ := calendarstore.OccurrenceMeta{
		EventID:  3,
		Title:    "Holiday",
		AllDay:   true,
		StartsAt: time.Date(2026, time.April, 29, 0, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC),
	}
	timed, allDay := placeWeekEvents([]calendarstore.OccurrenceMeta{occ}, weekStart, 2, 6, 22)
	if len(allDay[2]) != 1 {
		t.Errorf("expected 1 all-day event on column 2, got %d", len(allDay[2]))
	}
	for s, slot := range timed[2] {
		if slot.EventID != 0 {
			t.Errorf("all-day should not occupy timed slot[%d]", s)
		}
	}
}

func TestPlaceWeekEventsClampToVisibleHours(t *testing.T) {
	weekStart := time.Date(2026, time.April, 27, 0, 0, 0, 0, time.UTC)
	// 23:00 is past 22:00 visible window
	occ := calendarstore.OccurrenceMeta{
		EventID:  4,
		Title:    "Late",
		StartsAt: time.Date(2026, time.April, 27, 23, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, time.April, 28, 0, 0, 0, 0, time.UTC),
	}
	timed, _ := placeWeekEvents([]calendarstore.OccurrenceMeta{occ}, weekStart, 2, 6, 22)
	for s, slot := range timed[0] {
		if slot.EventID != 0 {
			t.Errorf("event past visible hours should not appear, slot[%d]=%+v", s, slot)
		}
	}
}

func TestFirstAgendaIndexForDate(t *testing.T) {
	items := []calendarstore.OccurrenceMeta{
		{EventID: 1, StartsAt: time.Date(2026, time.April, 28, 10, 0, 0, 0, time.UTC)},
		{EventID: 2, StartsAt: time.Date(2026, time.April, 29, 9, 0, 0, 0, time.UTC)},
		{EventID: 3, StartsAt: time.Date(2026, time.April, 29, 14, 0, 0, 0, time.UTC)},
	}
	got := firstAgendaIndexForDate(items, time.Date(2026, time.April, 29, 0, 0, 0, 0, time.UTC))
	if got != 1 {
		t.Errorf("got %d, want 1", got)
	}
	miss := firstAgendaIndexForDate(items, time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC))
	if miss != -1 {
		t.Errorf("got %d for missing date, want -1", miss)
	}
}

func TestCalendarMonthViewRendersSelectedDay(t *testing.T) {
	c := Calendar{
		mode:            calendarViewMonth,
		selectedDate:    time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
		weekStartsOn:    time.Monday,
		visibleHourFrom: 6,
		visibleHourTo:   22,
		slotsPerHour:    2,
		items: []calendarstore.OccurrenceMeta{
			{EventID: 99, CalendarID: 1, Title: "TeamSync", StartsAt: time.Date(2026, time.April, 15, 10, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, time.April, 15, 11, 0, 0, 0, time.UTC)},
		},
		calendars: []calendarstore.CalendarMeta{
			{RemoteID: 1, Name: "Work", Color: "#22c55e"},
		},
	}
	out := c.monthView(120, 24)
	if !contains(out, "15") {
		t.Errorf("month view did not contain day number 15:\n%s", out)
	}
	if !contains(out, "TeamSync") {
		t.Errorf("month view did not contain event title:\n%s", out)
	}
}

func TestCalendarWeekViewRendersHourAxis(t *testing.T) {
	c := Calendar{
		mode:            calendarViewWeek,
		selectedDate:    time.Date(2026, time.April, 29, 0, 0, 0, 0, time.UTC),
		weekStartsOn:    time.Monday,
		visibleHourFrom: 6,
		visibleHourTo:   22,
		slotsPerHour:    2,
		hourCursor:      9,
	}
	out := c.weekView(140, 40)
	if !contains(out, "06:00") {
		t.Errorf("week view should include 06:00 axis label:\n%s", out)
	}
	if !contains(out, "Wed") {
		t.Errorf("week view should include weekday header:\n%s", out)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (indexOf(haystack, needle) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
