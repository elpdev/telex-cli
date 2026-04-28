package screens

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/huh/v2"
	"github.com/elpdev/telex-cli/internal/calendar"
)

func calendarEventInputFromForm(data calendarEventFormData) (calendar.CalendarEventInput, error) {
	calendarID, err := strconv.ParseInt(strings.TrimSpace(data.CalendarID), 10, 64)
	if err != nil || calendarID <= 0 {
		return calendar.CalendarEventInput{}, fmt.Errorf("invalid calendar ID")
	}
	input := calendar.CalendarEventInput{
		CalendarID:  calendarID,
		Title:       strings.TrimSpace(data.Title),
		Description: strings.TrimSpace(data.Description),
		Location:    strings.TrimSpace(data.Location),
		StartDate:   strings.TrimSpace(data.StartDate),
		EndDate:     strings.TrimSpace(data.EndDate),
		TimeZone:    strings.TrimSpace(data.TimeZone),
		Status:      strings.TrimSpace(data.Status),
	}
	allDay := data.AllDay
	input.AllDay = &allDay
	if input.Title == "" {
		return input, fmt.Errorf("title is required")
	}
	if err := requiredDateString(input.StartDate); err != nil {
		return input, fmt.Errorf("invalid start date")
	}
	if err := requiredDateString(input.EndDate); err != nil {
		return input, fmt.Errorf("invalid end date")
	}
	if input.TimeZone != "" {
		if err := optionalTimeZoneString(input.TimeZone); err != nil {
			return input, err
		}
	}
	if !allDay {
		input.StartTime = strings.TrimSpace(data.StartTime)
		input.EndTime = strings.TrimSpace(data.EndTime)
		if input.StartTime == "" || input.EndTime == "" {
			return input, fmt.Errorf("start time and end time are required unless all day")
		}
		if err := optionalTimeString(input.StartTime); err != nil {
			return input, fmt.Errorf("invalid start time")
		}
		if err := optionalTimeString(input.EndTime); err != nil {
			return input, fmt.Errorf("invalid end time")
		}
	}
	if err := validateCalendarRange(input); err != nil {
		return input, err
	}
	return input, nil
}

func calendarInputFromForm(data calendarFormData) (calendar.CalendarInput, error) {
	input := calendar.CalendarInput{Name: strings.TrimSpace(data.Name), Color: strings.TrimSpace(data.Color), TimeZone: strings.TrimSpace(data.TimeZone)}
	if input.Name == "" {
		return input, fmt.Errorf("name is required")
	}
	if input.TimeZone != "" {
		if err := optionalTimeZoneString(input.TimeZone); err != nil {
			return input, err
		}
	}
	if strings.TrimSpace(data.Position) != "" {
		position, err := strconv.Atoi(strings.TrimSpace(data.Position))
		if err != nil || position <= 0 {
			return input, fmt.Errorf("invalid position")
		}
		input.Position = &position
	}
	return input, nil
}

func validateCalendarRange(input calendar.CalendarEventInput) error {
	if input.AllDay != nil && *input.AllDay {
		start, err := time.Parse("2006-01-02", input.StartDate)
		if err != nil {
			return err
		}
		end, err := time.Parse("2006-01-02", input.EndDate)
		if err != nil {
			return err
		}
		if end.Before(start) {
			return fmt.Errorf("end date cannot be before start date")
		}
		return nil
	}
	start, err := time.Parse("2006-01-02 15:04", input.StartDate+" "+input.StartTime)
	if err != nil {
		return err
	}
	end, err := time.Parse("2006-01-02 15:04", input.EndDate+" "+input.EndTime)
	if err != nil {
		return err
	}
	if end.Before(start) {
		return fmt.Errorf("end cannot be before start")
	}
	return nil
}

func requiredDateString(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("required")
	}
	_, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return errors.New("must be YYYY-MM-DD")
	}
	return nil
}

func optionalTimeString(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	_, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return errors.New("must be HH:MM")
	}
	return nil
}

func optionalTimeZoneString(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if _, err := time.LoadLocation(strings.TrimSpace(value)); err != nil {
		return errors.New("must be an IANA time zone")
	}
	return nil
}

func calendarFormKeyMap() *huh.KeyMap {
	keys := huh.NewDefaultKeyMap()
	keys.Input.Prev = key.NewBinding(key.WithKeys("up", "k", "shift+tab"), key.WithHelp("up/k", "previous"))
	keys.Input.Next = key.NewBinding(key.WithKeys("down", "j", "tab", "enter"), key.WithHelp("down/j", "next"))
	keys.Confirm.Prev = key.NewBinding(key.WithKeys("up", "k", "shift+tab"), key.WithHelp("up/k", "previous"))
	keys.Confirm.Next = key.NewBinding(key.WithKeys("down", "j", "tab", "enter"), key.WithHelp("down/j", "next"))
	keys.Note.Prev = key.NewBinding(key.WithKeys("up", "k", "shift+tab"), key.WithHelp("up/k", "previous"))
	keys.Note.Next = key.NewBinding(key.WithKeys("down", "j", "tab", "enter"), key.WithHelp("down/j", "next"))
	return keys
}

func calendarFormTitle(kind calendarFormKind) string {
	switch kind {
	case calendarFormEventEdit:
		return "Edit Event"
	case calendarFormCalendarCreate:
		return "New Calendar"
	case calendarFormCalendarEdit:
		return "Edit Calendar"
	default:
		return "New Event"
	}
}
