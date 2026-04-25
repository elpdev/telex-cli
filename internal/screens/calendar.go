package screens

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	huhkey "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/telex-cli/internal/calendar"
	"github.com/elpdev/telex-cli/internal/calendarstore"
)

type CalendarSyncFunc func(context.Context) (CalendarSyncResult, error)
type CreateCalendarEventFunc func(context.Context, calendar.CalendarEventInput) (*calendar.CalendarEvent, error)
type UpdateCalendarEventFunc func(context.Context, int64, calendar.CalendarEventInput) (*calendar.CalendarEvent, error)
type DeleteCalendarEventFunc func(context.Context, int64) error

type calendarFormKind int

const (
	calendarFormNone calendarFormKind = iota
	calendarFormCreate
	calendarFormEdit
)

type CalendarSyncResult struct {
	Calendars   int
	Events      int
	Occurrences int
}

type Calendar struct {
	store    calendarstore.Store
	sync     CalendarSyncFunc
	create   CreateCalendarEventFunc
	update   UpdateCalendarEventFunc
	delete   DeleteCalendarEventFunc
	items    []calendarstore.OccurrenceMeta
	index    int
	detail   bool
	form     *huh.Form
	formKind calendarFormKind
	formID   int64
	formData *calendarEventFormData
	confirm  string
	loading  bool
	syncing  bool
	err      error
	status   string
	keys     CalendarKeyMap
}

type CalendarKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Open    key.Binding
	Back    key.Binding
	Refresh key.Binding
	Sync    key.Binding
	Today   key.Binding
	New     key.Binding
	Edit    key.Binding
	Delete  key.Binding
}

type calendarEventFormData struct {
	CalendarID  string
	Title       string
	Description string
	Location    string
	AllDay      bool
	StartDate   string
	EndDate     string
	StartTime   string
	EndTime     string
	TimeZone    string
	Status      string
}

type calendarLoadedMsg struct {
	items []calendarstore.OccurrenceMeta
	err   error
}

type calendarSyncedMsg struct {
	result CalendarSyncResult
	loaded calendarLoadedMsg
	err    error
}

type calendarActionFinishedMsg struct {
	status string
	loaded calendarLoadedMsg
	err    error
}

type CalendarActionMsg struct{ Action string }

func NewCalendar(store calendarstore.Store, sync CalendarSyncFunc) Calendar {
	return Calendar{store: store, sync: sync, loading: true, keys: DefaultCalendarKeyMap()}
}

func (c Calendar) WithActions(create CreateCalendarEventFunc, update UpdateCalendarEventFunc, delete DeleteCalendarEventFunc) Calendar {
	c.create = create
	c.update = update
	c.delete = delete
	return c
}

func DefaultCalendarKeyMap() CalendarKeyMap {
	return CalendarKeyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "item up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "item down")),
		Open:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
		Back:    key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload cache")),
		Sync:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sync calendar")),
		Today:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "today")),
		New:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new event")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit event")),
		Delete:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete event")),
	}
}

func (c Calendar) Init() tea.Cmd { return c.loadCmd() }

func (c Calendar) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if c.form != nil {
		return c.updateForm(msg)
	}

	switch msg := msg.(type) {
	case calendarLoadedMsg:
		c.loading = false
		c.err = msg.err
		if msg.err == nil {
			c.items = msg.items
			c.clampIndex()
		}
		return c, nil
	case calendarSyncedMsg:
		c.syncing = false
		c.err = msg.err
		if msg.err != nil {
			c.status = ""
			return c, nil
		}
		c.status = fmt.Sprintf("Synced %d calendar(s), %d event(s), %d occurrence(s)", msg.result.Calendars, msg.result.Events, msg.result.Occurrences)
		c.items = msg.loaded.items
		c.clampIndex()
		return c, nil
	case calendarActionFinishedMsg:
		c.loading = false
		c.err = msg.err
		if msg.err != nil {
			c.status = fmt.Sprintf("Calendar action failed: %v", msg.err)
			return c, nil
		}
		c.status = msg.status
		c.items = msg.loaded.items
		c.detail = false
		c.form = nil
		c.formKind = calendarFormNone
		c.confirm = ""
		c.clampIndex()
		return c, nil
	case CalendarActionMsg:
		return c.handleAction(msg.Action)
	case tea.KeyPressMsg:
		return c.handleKey(msg)
	}
	return c, nil
}

func (c Calendar) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if c.loading {
		return style.Render("Loading local calendar cache...")
	}
	if c.form != nil {
		return style.Render(c.form.WithWidth(max(40, width-4)).WithHeight(max(8, height-3)).View())
	}
	if c.err != nil {
		return style.Render(fmt.Sprintf("Calendar cache error: %v\n\nRun `telex calendar sync` or press S to populate Calendar.", c.err))
	}
	var b strings.Builder
	b.WriteString("Calendar\n")
	if c.status != "" {
		b.WriteString(c.status + "\n")
	}
	if c.syncing {
		b.WriteString("Syncing remote Calendar...\n")
	}
	if c.confirm != "" {
		b.WriteString(c.confirm + " [y/N]\n")
	}
	b.WriteString("\n")
	if c.detail {
		b.WriteString(c.detailView())
		return style.Render(b.String())
	}
	if len(c.items) == 0 {
		b.WriteString("No cached calendar occurrences found. Press S to sync.\n")
		return style.Render(b.String())
	}
	for i, item := range c.items {
		cursor := "  "
		if i == c.index {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("%s%s  %s  %s\n", cursor, item.StartsAt.Format("Jan 02 15:04"), item.Title, item.Status))
	}
	return style.Render(b.String())
}

func (c Calendar) Title() string { return "Calendar" }

func (c Calendar) KeyBindings() []key.Binding {
	return []key.Binding{c.keys.Up, c.keys.Down, c.keys.Open, c.keys.Back, c.keys.Refresh, c.keys.Sync, c.keys.Today, c.keys.New, c.keys.Edit, c.keys.Delete}
}

func (c Calendar) CapturesFocusKey(tea.KeyPressMsg) bool { return c.form != nil }

func (c Calendar) Selection() CalendarSelection {
	item, ok := c.selected()
	if !ok {
		return CalendarSelection{Kind: "calendar-event", HasItem: false}
	}
	return CalendarSelection{Kind: "calendar-event", Subject: item.Title, HasItem: true}
}

type CalendarSelection struct {
	Kind    string
	Subject string
	HasItem bool
}

func (c Calendar) handleAction(action string) (Screen, tea.Cmd) {
	if c.confirm != "" || c.form != nil {
		return c, nil
	}
	switch action {
	case "sync":
		if c.sync == nil || c.syncing {
			return c, nil
		}
		c.syncing = true
		c.status = ""
		return c, c.syncCmd()
	case "delete":
		if item, ok := c.selected(); ok {
			c.confirm = fmt.Sprintf("Delete event %s?", strconv.FormatInt(item.EventID, 10))
		}
		return c, nil
	case "new":
		return c.startEventForm(calendarFormCreate, nil)
	case "edit":
		item, ok := c.selected()
		if !ok {
			c.status = "Select an event to edit"
			return c, nil
		}
		cached, err := c.store.ReadEvent(item.EventID)
		if err != nil {
			c.status = fmt.Sprintf("Cannot load event: %v", err)
			return c, nil
		}
		return c.startEventForm(calendarFormEdit, cached)
	case "today":
		c.jumpToToday()
		return c, nil
	}
	return c, nil
}

func (c Calendar) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if c.confirm != "" {
		if msg.String() == "y" || msg.String() == "Y" {
			item, ok := c.selected()
			c.confirm = ""
			if ok && c.delete != nil {
				c.loading = true
				return c, c.deleteCmd(item.EventID)
			}
		}
		if key.Matches(msg, c.keys.Back) || msg.String() == "n" || msg.String() == "N" {
			c.confirm = ""
		}
		return c, nil
	}
	if key.Matches(msg, c.keys.Up) && c.index > 0 {
		c.index--
		return c, nil
	}
	if key.Matches(msg, c.keys.Down) && c.index < len(c.items)-1 {
		c.index++
		return c, nil
	}
	if key.Matches(msg, c.keys.Open) && len(c.items) > 0 {
		c.detail = true
		return c, nil
	}
	if key.Matches(msg, c.keys.Back) && c.detail {
		c.detail = false
		return c, nil
	}
	if key.Matches(msg, c.keys.Refresh) {
		c.loading = true
		return c, c.loadCmd()
	}
	if key.Matches(msg, c.keys.Sync) {
		return c.handleAction("sync")
	}
	if key.Matches(msg, c.keys.Today) {
		return c.handleAction("today")
	}
	if key.Matches(msg, c.keys.New) {
		return c.handleAction("new")
	}
	if key.Matches(msg, c.keys.Edit) {
		return c.handleAction("edit")
	}
	if key.Matches(msg, c.keys.Delete) {
		return c.handleAction("delete")
	}
	return c, nil
}

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
		data := *c.formData
		c.form = nil
		c.formKind = calendarFormNone
		c.loading = true
		c.status = "Saving event..."
		return c, c.saveFormCmd(kind, id, data)
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

func (c Calendar) saveFormCmd(kind calendarFormKind, id int64, data calendarEventFormData) tea.Cmd {
	return func() tea.Msg {
		input, err := calendarEventInputFromForm(data)
		if err != nil {
			return calendarActionFinishedMsg{err: err}
		}
		switch kind {
		case calendarFormCreate:
			if c.create == nil {
				return calendarActionFinishedMsg{err: errors.New("create is not configured")}
			}
			event, err := c.create(context.Background(), input)
			loaded := calendarLoadedMsg{}
			if err == nil {
				items, loadErr := c.store.ListOccurrences()
				loaded = calendarLoadedMsg{items: items, err: loadErr}
				if loadErr != nil {
					err = loadErr
				}
			}
			status := "Created event"
			if event != nil && event.Title != "" {
				status = "Created " + event.Title
			}
			return calendarActionFinishedMsg{status: status, loaded: loaded, err: err}
		case calendarFormEdit:
			if c.update == nil {
				return calendarActionFinishedMsg{err: errors.New("edit is not configured")}
			}
			event, err := c.update(context.Background(), id, input)
			loaded := calendarLoadedMsg{}
			if err == nil {
				items, loadErr := c.store.ListOccurrences()
				loaded = calendarLoadedMsg{items: items, err: loadErr}
				if loadErr != nil {
					err = loadErr
				}
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

func (c Calendar) detailView() string {
	item, ok := c.selected()
	if !ok {
		return "No event selected.\n"
	}
	return strings.Join([]string{
		item.Title,
		"",
		"Event ID: " + strconv.FormatInt(item.EventID, 10),
		"Calendar ID: " + strconv.FormatInt(item.CalendarID, 10),
		"Starts: " + item.StartsAt.Format("2006-01-02 15:04"),
		"Ends: " + item.EndsAt.Format("2006-01-02 15:04"),
		"All day: " + strconv.FormatBool(item.AllDay),
		"Location: " + item.Location,
		"Status: " + item.Status,
	}, "\n") + "\n"
}

func (c *Calendar) jumpToToday() {
	today := time.Now().Format("2006-01-02")
	for i, item := range c.items {
		if item.StartsAt.Format("2006-01-02") >= today {
			c.index = i
			return
		}
	}
}

func (c Calendar) selected() (calendarstore.OccurrenceMeta, bool) {
	if c.index < 0 || c.index >= len(c.items) {
		return calendarstore.OccurrenceMeta{}, false
	}
	return c.items[c.index], true
}

func (c *Calendar) clampIndex() {
	if c.index < 0 {
		c.index = 0
	}
	if c.index >= len(c.items) && len(c.items) > 0 {
		c.index = len(c.items) - 1
	}
	if len(c.items) == 0 {
		c.index = 0
		c.detail = false
	}
}

func (c Calendar) loadCmd() tea.Cmd {
	return func() tea.Msg {
		items, err := c.store.ListOccurrences()
		return calendarLoadedMsg{items: items, err: err}
	}
}

func (c Calendar) syncCmd() tea.Cmd {
	return func() tea.Msg {
		result, err := c.sync(context.Background())
		loaded := calendarLoadedMsg{}
		if err == nil {
			items, loadErr := c.store.ListOccurrences()
			loaded = calendarLoadedMsg{items: items, err: loadErr}
			if loadErr != nil {
				err = loadErr
			}
		}
		return calendarSyncedMsg{result: result, loaded: loaded, err: err}
	}
}

func (c Calendar) deleteCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		err := c.delete(context.Background(), id)
		loaded := calendarLoadedMsg{}
		if err == nil {
			items, loadErr := c.store.ListOccurrences()
			loaded = calendarLoadedMsg{items: items, err: loadErr}
			if loadErr != nil {
				err = loadErr
			}
		}
		return calendarActionFinishedMsg{status: "Deleted event", loaded: loaded, err: err}
	}
}

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
	keys.Input.Prev = huhkey.NewBinding(huhkey.WithKeys("up", "k", "shift+tab"), huhkey.WithHelp("up/k", "previous"))
	keys.Input.Next = huhkey.NewBinding(huhkey.WithKeys("down", "j", "tab", "enter"), huhkey.WithHelp("down/j", "next"))
	keys.Confirm.Prev = huhkey.NewBinding(huhkey.WithKeys("up", "k", "shift+tab"), huhkey.WithHelp("up/k", "previous"))
	keys.Confirm.Next = huhkey.NewBinding(huhkey.WithKeys("down", "j", "tab", "enter"), huhkey.WithHelp("down/j", "next"))
	keys.Note.Prev = huhkey.NewBinding(huhkey.WithKeys("up", "k", "shift+tab"), huhkey.WithHelp("up/k", "previous"))
	keys.Note.Next = huhkey.NewBinding(huhkey.WithKeys("down", "j", "tab", "enter"), huhkey.WithHelp("down/j", "next"))
	return keys
}

func calendarFormTitle(kind calendarFormKind) string {
	if kind == calendarFormEdit {
		return "Edit Event"
	}
	return "New Event"
}
