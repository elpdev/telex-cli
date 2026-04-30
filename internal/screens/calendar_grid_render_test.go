package screens

import (
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/calendarstore"
)

func TestCalendarMonthViewVisualSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	c := Calendar{
		mode:            calendarViewMonth,
		selectedDate:    time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
		weekStartsOn:    time.Monday,
		visibleHourFrom: 6,
		visibleHourTo:   22,
		slotsPerHour:    2,
		items: []calendarstore.OccurrenceMeta{
			{EventID: 1, CalendarID: 1, Title: "Standup", StartsAt: time.Date(2026, time.April, 6, 9, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, time.April, 6, 9, 30, 0, 0, time.UTC)},
			{EventID: 2, CalendarID: 1, Title: "Design review", StartsAt: time.Date(2026, time.April, 15, 10, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, time.April, 15, 11, 0, 0, 0, time.UTC)},
			{EventID: 3, CalendarID: 2, Title: "Lunch w/ Alex", StartsAt: time.Date(2026, time.April, 15, 12, 30, 0, 0, time.UTC), EndsAt: time.Date(2026, time.April, 15, 13, 30, 0, 0, time.UTC)},
			{EventID: 4, CalendarID: 1, Title: "Quarterly planning", StartsAt: time.Date(2026, time.April, 22, 14, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, time.April, 22, 16, 0, 0, 0, time.UTC)},
		},
		calendars: []calendarstore.CalendarMeta{
			{RemoteID: 1, Name: "Work", Color: "#22c55e"},
			{RemoteID: 2, Name: "Personal", Color: "#0ea5e9"},
		},
	}
	out := c.monthView(120, 32)
	t.Logf("\n%s", out)
}

func TestCalendarWeekViewVisualSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	weekStart := time.Date(2026, time.April, 27, 0, 0, 0, 0, time.UTC) // Monday
	c := Calendar{
		mode:            calendarViewWeek,
		selectedDate:    weekStart.AddDate(0, 0, 2), // Wednesday
		weekStartsOn:    time.Monday,
		visibleHourFrom: 8,
		visibleHourTo:   18,
		slotsPerHour:    2,
		hourCursor:      10,
		items: []calendarstore.OccurrenceMeta{
			{EventID: 1, CalendarID: 1, Title: "Standup", StartsAt: weekStart.Add(9 * time.Hour), EndsAt: weekStart.Add(9*time.Hour + 30*time.Minute)},
			{EventID: 2, CalendarID: 1, Title: "Design review", StartsAt: weekStart.AddDate(0, 0, 2).Add(10 * time.Hour), EndsAt: weekStart.AddDate(0, 0, 2).Add(11 * time.Hour)},
			{EventID: 3, CalendarID: 2, Title: "Lunch", StartsAt: weekStart.AddDate(0, 0, 2).Add(12*time.Hour + 30*time.Minute), EndsAt: weekStart.AddDate(0, 0, 2).Add(13*time.Hour + 30*time.Minute)},
			{EventID: 4, CalendarID: 1, AllDay: true, Title: "Conference", StartsAt: weekStart.AddDate(0, 0, 4), EndsAt: weekStart.AddDate(0, 0, 5)},
		},
		calendars: []calendarstore.CalendarMeta{
			{RemoteID: 1, Name: "Work", Color: "#22c55e"},
			{RemoteID: 2, Name: "Personal", Color: "#0ea5e9"},
		},
	}
	out := c.weekView(140, 30)
	t.Logf("\n%s", out)
}
