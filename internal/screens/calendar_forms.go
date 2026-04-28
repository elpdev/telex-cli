package screens

import (
	"context"
	"errors"
	"strconv"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/elpdev/telex-cli/internal/calendarstore"
)

func (c Calendar) updateForm(msg tea.Msg) (Screen, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && key.Matches(keyMsg, c.keys.Back) {
		c.form = nil
		c.formKind = calendarFormNone
		c.status = "Cancelled"
		return c, nil
	}
	model, cmd := c.form.Update(msg)
	if form, ok := model.(*huh.Form); ok {
		c.form = form
	}
	if c.form.State == huh.StateAborted {
		c.form = nil
		c.formKind = calendarFormNone
		c.status = "Cancelled"
		return c, nil
	}
	if c.form.State == huh.StateCompleted {
		kind := c.formKind
		id := c.formID
		var eventData calendarEventFormData
		if c.formData != nil {
			eventData = *c.formData
		}
		var calendarData calendarFormData
		if c.calendarForm != nil {
			calendarData = *c.calendarForm
		}
		c.form = nil
		c.formKind = calendarFormNone
		c.loading = true
		c.status = "Saving calendar..."
		if kind == calendarFormEventCreate || kind == calendarFormEventEdit {
			c.status = "Saving event..."
			return c, c.saveEventFormCmd(kind, id, eventData)
		}
		return c, c.saveCalendarFormCmd(kind, id, calendarData)
	}
	return c, cmd
}

func (c Calendar) startEventForm(kind calendarFormKind, cached *calendarstore.CachedEvent) (Screen, tea.Cmd) {
	data := calendarEventFormData{StartDate: time.Now().Format("2006-01-02"), EndDate: time.Now().Format("2006-01-02"), StartTime: "09:00", EndTime: "10:00", Status: "confirmed"}
	if item, ok := c.selected(); ok {
		data.CalendarID = strconv.FormatInt(item.CalendarID, 10)
		data.StartDate = item.StartsAt.Format("2006-01-02")
		data.EndDate = item.EndsAt.Format("2006-01-02")
	}
	var id int64
	if cached != nil {
		id = cached.Meta.RemoteID
		data.CalendarID = strconv.FormatInt(cached.Meta.CalendarID, 10)
		data.Title = cached.Meta.Title
		data.Description = cached.Description
		data.Location = cached.Meta.Location
		data.AllDay = cached.Meta.AllDay
		data.StartDate = cached.Meta.StartsAt.Format("2006-01-02")
		data.EndDate = cached.Meta.EndsAt.Format("2006-01-02")
		if !cached.Meta.AllDay {
			data.StartTime = cached.Meta.StartsAt.Format("15:04")
			data.EndTime = cached.Meta.EndsAt.Format("15:04")
		}
		data.TimeZone = cached.Meta.TimeZone
		data.Status = cached.Meta.Status
	}
	c.formData = &data
	c.calendarForm = nil
	c.formID = id
	c.formKind = kind
	c.form = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Calendar ID").Value(&c.formData.CalendarID).Validate(requiredInt64String),
		huh.NewInput().Title("Title").Value(&c.formData.Title).Validate(requiredString),
		huh.NewInput().Title("Description").Value(&c.formData.Description),
		huh.NewInput().Title("Location").Value(&c.formData.Location),
		huh.NewConfirm().Title("All day").Value(&c.formData.AllDay),
		huh.NewInput().Title("Start date").Description("YYYY-MM-DD").Value(&c.formData.StartDate).Validate(requiredDateString),
		huh.NewInput().Title("Start time").Description("HH:MM, required unless all day").Value(&c.formData.StartTime).Validate(optionalTimeString),
		huh.NewInput().Title("End date").Description("YYYY-MM-DD").Value(&c.formData.EndDate).Validate(requiredDateString),
		huh.NewInput().Title("End time").Description("HH:MM, required unless all day").Value(&c.formData.EndTime).Validate(optionalTimeString),
		huh.NewInput().Title("Time zone").Description("Optional IANA time zone, e.g. UTC").Value(&c.formData.TimeZone).Validate(optionalTimeZoneString),
		huh.NewInput().Title("Status").Description("Optional, e.g. confirmed").Value(&c.formData.Status),
	).Title(calendarFormTitle(kind)).Description("Move between fields with up/down, j/k, or tab/shift+tab. Enter advances; submit from the last field."))
	c.form.WithKeyMap(calendarFormKeyMap()).WithShowHelp(true)
	return c, c.form.Init()
}

func (c Calendar) startCalendarForm(kind calendarFormKind, cached *calendarstore.CalendarMeta) (Screen, tea.Cmd) {
	data := calendarFormData{TimeZone: "UTC"}
	var id int64
	if cached != nil {
		id = cached.RemoteID
		data.Name = cached.Name
		data.Color = cached.Color
		data.TimeZone = cached.TimeZone
		if cached.Position > 0 {
			data.Position = strconv.Itoa(cached.Position)
		}
	}
	c.formData = nil
	c.calendarForm = &data
	c.formID = id
	c.formKind = kind
	c.form = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Name").Value(&c.calendarForm.Name).Validate(requiredString),
		huh.NewInput().Title("Color").Description("Optional, e.g. #22c55e or green").Value(&c.calendarForm.Color),
		huh.NewInput().Title("Time zone").Description("Optional IANA time zone, e.g. UTC").Value(&c.calendarForm.TimeZone).Validate(optionalTimeZoneString),
		huh.NewInput().Title("Position").Description("Optional positive sort number").Value(&c.calendarForm.Position).Validate(optionalIntString),
	).Title(calendarFormTitle(kind)).Description("Move between fields with up/down, j/k, or tab/shift+tab. Enter advances; submit from the last field."))
	c.form.WithKeyMap(calendarFormKeyMap()).WithShowHelp(true)
	return c, c.form.Init()
}

func (c Calendar) saveEventFormCmd(kind calendarFormKind, id int64, data calendarEventFormData) tea.Cmd {
	return func() tea.Msg {
		input, err := calendarEventInputFromForm(data)
		if err != nil {
			return calendarActionFinishedMsg{err: err}
		}
		switch kind {
		case calendarFormEventCreate:
			if c.createEvent == nil {
				return calendarActionFinishedMsg{err: errors.New("create is not configured")}
			}
			event, err := c.createEvent(context.Background(), input)
			loaded := calendarLoadedMsg{}
			if err == nil {
				loaded = c.load()
				err = loaded.err
			}
			status := "Created event"
			if event != nil && event.Title != "" {
				status = "Created " + event.Title
			}
			return calendarActionFinishedMsg{status: status, loaded: loaded, err: err}
		case calendarFormEventEdit:
			if c.updateEvent == nil {
				return calendarActionFinishedMsg{err: errors.New("edit is not configured")}
			}
			event, err := c.updateEvent(context.Background(), id, input)
			loaded := calendarLoadedMsg{}
			if err == nil {
				loaded = c.load()
				err = loaded.err
			}
			status := "Updated event"
			if event != nil && event.Title != "" {
				status = "Updated " + event.Title
			}
			return calendarActionFinishedMsg{status: status, loaded: loaded, err: err}
		}
		return calendarActionFinishedMsg{err: errors.New("unknown calendar form")}
	}
}

func (c Calendar) saveCalendarFormCmd(kind calendarFormKind, id int64, data calendarFormData) tea.Cmd {
	return func() tea.Msg {
		input, err := calendarInputFromForm(data)
		if err != nil {
			return calendarActionFinishedMsg{err: err}
		}
		switch kind {
		case calendarFormCalendarCreate:
			if c.createCalendar == nil {
				return calendarActionFinishedMsg{err: errors.New("calendar create is not configured")}
			}
			created, err := c.createCalendar(context.Background(), input)
			loaded := calendarLoadedMsg{}
			if err == nil {
				loaded = c.load()
				err = loaded.err
			}
			status := "Created calendar"
			if created != nil && created.Name != "" {
				status = "Created " + created.Name
			}
			return calendarActionFinishedMsg{status: status, loaded: loaded, err: err}
		case calendarFormCalendarEdit:
			if c.updateCalendar == nil {
				return calendarActionFinishedMsg{err: errors.New("calendar edit is not configured")}
			}
			updated, err := c.updateCalendar(context.Background(), id, input)
			loaded := calendarLoadedMsg{}
			if err == nil {
				loaded = c.load()
				err = loaded.err
			}
			status := "Updated calendar"
			if updated != nil && updated.Name != "" {
				status = "Updated " + updated.Name
			}
			return calendarActionFinishedMsg{status: status, loaded: loaded, err: err}
		}
		return calendarActionFinishedMsg{err: errors.New("unknown calendar form")}
	}
}
