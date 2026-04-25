package screens

import "testing"

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
