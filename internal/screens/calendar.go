package screens

import (
	"context"
	"errors"
	"fmt"
	"os"
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
	"github.com/elpdev/telex-cli/internal/components/filepicker"
)

type CalendarSyncFunc func(context.Context) (CalendarSyncResult, error)
type CreateCalendarFunc func(context.Context, calendar.CalendarInput) (*calendar.Calendar, error)
type UpdateCalendarFunc func(context.Context, int64, calendar.CalendarInput) (*calendar.Calendar, error)
type DeleteCalendarFunc func(context.Context, int64) error
type ImportICSFunc func(context.Context, int64, string) (*calendar.ImportResult, error)
type CreateCalendarEventFunc func(context.Context, calendar.CalendarEventInput) (*calendar.CalendarEvent, error)
type UpdateCalendarEventFunc func(context.Context, int64, calendar.CalendarEventInput) (*calendar.CalendarEvent, error)
type DeleteCalendarEventFunc func(context.Context, int64) error

type calendarViewMode int

const (
	calendarViewAgenda calendarViewMode = iota
	calendarViewCalendars
)

type calendarFormKind int

const (
	calendarFormNone calendarFormKind = iota
	calendarFormEventCreate
	calendarFormEventEdit
	calendarFormCalendarCreate
	calendarFormCalendarEdit
)

type CalendarSyncResult struct {
	Calendars   int
	Events      int
	Occurrences int
}

type Calendar struct {
	store          calendarstore.Store
	sync           CalendarSyncFunc
	createCalendar CreateCalendarFunc
	updateCalendar UpdateCalendarFunc
	deleteCalendar DeleteCalendarFunc
	importICS      ImportICSFunc
	createEvent    CreateCalendarEventFunc
	updateEvent    UpdateCalendarEventFunc
	deleteEvent    DeleteCalendarEventFunc
	items          []calendarstore.OccurrenceMeta
	calendars      []calendarstore.CalendarMeta
	mode           calendarViewMode
	index          int
	calendarIndex  int
	detail         bool
	form           *huh.Form
	formKind       calendarFormKind
	formID         int64
	formData       *calendarEventFormData
	calendarForm   *calendarFormData
	filePicker     filepicker.Model
	filePickerOpen bool
	importCalendar int64
	confirm        string
	confirmAction  string
	confirmID      int64
	loading        bool
	syncing        bool
	err            error
	status         string
	keys           CalendarKeyMap
}

type CalendarKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Open    key.Binding
	Back    key.Binding
	Refresh key.Binding
	Sync    key.Binding
	Today   key.Binding
	View    key.Binding
	New     key.Binding
	Edit    key.Binding
	Delete  key.Binding
	Import  key.Binding
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

type calendarFormData struct {
	Name     string
	Color    string
	TimeZone string
	Position string
}

type calendarLoadedMsg struct {
	items     []calendarstore.OccurrenceMeta
	calendars []calendarstore.CalendarMeta
	err       error
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
	c.createEvent = create
	c.updateEvent = update
	c.deleteEvent = delete
	return c
}

func (c Calendar) WithCalendarActions(create CreateCalendarFunc, update UpdateCalendarFunc, delete DeleteCalendarFunc) Calendar {
	c.createCalendar = create
	c.updateCalendar = update
	c.deleteCalendar = delete
	return c
}

func (c Calendar) WithImportICS(importICS ImportICSFunc) Calendar {
	c.importICS = importICS
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
		View:    key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "agenda/calendars")),
		New:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new event")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit event")),
		Delete:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete event")),
		Import:  key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "import ics")),
	}
}

func (c Calendar) Init() tea.Cmd { return c.loadCmd() }

func (c Calendar) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if c.filePickerOpen {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			return c.handleImportFileKey(keyMsg)
		}
	}
	if c.form != nil {
		return c.updateForm(msg)
	}

	switch msg := msg.(type) {
	case calendarLoadedMsg:
		c.loading = false
		c.err = msg.err
		if msg.err == nil {
			c.items = msg.items
			c.calendars = msg.calendars
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
		c.calendars = msg.loaded.calendars
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
		c.calendars = msg.loaded.calendars
		c.detail = false
		c.form = nil
		c.formKind = calendarFormNone
		c.filePickerOpen = false
		c.importCalendar = 0
		c.confirm = ""
		c.confirmAction = ""
		c.confirmID = 0
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
	if c.filePickerOpen {
		return style.Render("Calendar / Import ICS\n" + c.status + "\n\n" + c.filePicker.View(width, max(1, height-3)))
	}
	if c.err != nil {
		return style.Render(fmt.Sprintf("Calendar cache error: %v\n\nRun `telex calendar sync` or press S to populate Calendar.", c.err))
	}
	var b strings.Builder
	b.WriteString("Calendar / " + c.modeTitle() + "\n")
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
	if c.mode == calendarViewCalendars {
		b.WriteString(c.calendarListView())
		return style.Render(b.String())
	}
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
	return []key.Binding{c.keys.Up, c.keys.Down, c.keys.Open, c.keys.Back, c.keys.Refresh, c.keys.Sync, c.keys.Today, c.keys.View, c.keys.New, c.keys.Edit, c.keys.Delete, c.keys.Import}
}

func (c Calendar) CapturesFocusKey(tea.KeyPressMsg) bool { return c.form != nil || c.filePickerOpen }

func (c Calendar) Selection() CalendarSelection {
	if c.mode == calendarViewCalendars {
		item, ok := c.selectedCalendar()
		if !ok {
			return CalendarSelection{Kind: "calendar", HasItem: false}
		}
		return CalendarSelection{Kind: "calendar", Subject: item.Name, HasItem: true}
	}
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
		if c.mode == calendarViewCalendars {
			if item, ok := c.selectedCalendar(); ok {
				c.confirm = fmt.Sprintf("Delete calendar %s?", strconv.FormatInt(item.RemoteID, 10))
				c.confirmAction = "delete-calendar"
				c.confirmID = item.RemoteID
			}
			return c, nil
		}
		if item, ok := c.selected(); ok {
			c.confirm = fmt.Sprintf("Delete event %s?", strconv.FormatInt(item.EventID, 10))
			c.confirmAction = "delete-event"
			c.confirmID = item.EventID
		}
		return c, nil
	case "new":
		return c.startEventForm(calendarFormEventCreate, nil)
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
		return c.startEventForm(calendarFormEventEdit, cached)
	case "today":
		c.jumpToToday()
		return c, nil
	case "toggle-view":
		c.toggleMode()
		return c, nil
	case "view-agenda":
		c.mode = calendarViewAgenda
		c.detail = false
		c.status = "Showing agenda"
		return c, nil
	case "view-calendars":
		c.mode = calendarViewCalendars
		c.detail = false
		c.status = "Showing calendars"
		return c, nil
	case "new-calendar":
		c.mode = calendarViewCalendars
		return c.startCalendarForm(calendarFormCalendarCreate, nil)
	case "edit-calendar":
		item, ok := c.selectedCalendar()
		if !ok {
			c.status = "Select a calendar to edit"
			return c, nil
		}
		return c.startCalendarForm(calendarFormCalendarEdit, &item)
	case "delete-calendar":
		item, ok := c.selectedCalendar()
		if !ok {
			c.status = "Select a calendar to delete"
			return c, nil
		}
		c.confirm = fmt.Sprintf("Delete calendar %s?", strconv.FormatInt(item.RemoteID, 10))
		c.confirmAction = "delete-calendar"
		c.confirmID = item.RemoteID
		return c, nil
	case "import-ics":
		if c.mode != calendarViewCalendars {
			c.mode = calendarViewCalendars
			c.detail = false
			c.status = "Select a calendar, then import ICS"
			return c, nil
		}
		return c.startImportICS()
	}
	return c, nil
}

func (c Calendar) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if c.confirm != "" {
		if msg.String() == "y" || msg.String() == "Y" {
			action := c.confirmAction
			id := c.confirmID
			c.confirm = ""
			c.confirmAction = ""
			c.confirmID = 0
			if action == "delete-event" && id > 0 && c.deleteEvent != nil {
				c.loading = true
				return c, c.deleteCmd(id)
			}
			if action == "delete-calendar" && id > 0 && c.deleteCalendar != nil {
				c.loading = true
				return c, c.deleteCalendarCmd(id)
			}
		}
		if key.Matches(msg, c.keys.Back) || msg.String() == "n" || msg.String() == "N" {
			c.confirm = ""
			c.confirmAction = ""
			c.confirmID = 0
		}
		return c, nil
	}
	if key.Matches(msg, c.keys.Up) && c.mode == calendarViewCalendars && c.calendarIndex > 0 {
		c.calendarIndex--
		return c, nil
	}
	if key.Matches(msg, c.keys.Down) && c.mode == calendarViewCalendars && c.calendarIndex < len(c.calendars)-1 {
		c.calendarIndex++
		return c, nil
	}
	if key.Matches(msg, c.keys.Up) && c.mode == calendarViewAgenda && c.index > 0 {
		c.index--
		return c, nil
	}
	if key.Matches(msg, c.keys.Down) && c.mode == calendarViewAgenda && c.index < len(c.items)-1 {
		c.index++
		return c, nil
	}
	if key.Matches(msg, c.keys.Open) && c.mode == calendarViewAgenda && len(c.items) > 0 {
		c.detail = true
		return c, nil
	}
	if key.Matches(msg, c.keys.Back) && c.detail {
		c.detail = false
		return c, nil
	}
	if key.Matches(msg, c.keys.Back) && c.mode == calendarViewCalendars {
		c.mode = calendarViewAgenda
		c.status = "Showing agenda"
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
	if key.Matches(msg, c.keys.View) {
		return c.handleAction("toggle-view")
	}
	if key.Matches(msg, c.keys.New) {
		if c.mode == calendarViewCalendars {
			return c.handleAction("new-calendar")
		}
		return c.handleAction("new")
	}
	if key.Matches(msg, c.keys.Edit) {
		if c.mode == calendarViewCalendars {
			return c.handleAction("edit-calendar")
		}
		return c.handleAction("edit")
	}
	if key.Matches(msg, c.keys.Delete) {
		return c.handleAction("delete")
	}
	if key.Matches(msg, c.keys.Import) && c.mode == calendarViewCalendars {
		return c.handleAction("import-ics")
	}
	return c, nil
}

func (c Calendar) startImportICS() (Screen, tea.Cmd) {
	item, ok := c.selectedCalendar()
	if !ok {
		c.status = "Select a calendar to import ICS"
		return c, nil
	}
	if c.importICS == nil {
		c.status = "ICS import is not configured"
		return c, nil
	}
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		cwd, _ = os.UserHomeDir()
	}
	c.filePicker = filepicker.New("", cwd, filepicker.ModeOpenFile)
	c.filePickerOpen = true
	c.importCalendar = item.RemoteID
	c.status = fmt.Sprintf("Select .ics file for %s", item.Name)
	return c, nil
}

func (c Calendar) handleImportFileKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	picker, action := c.filePicker.Update(msg)
	c.filePicker = picker
	switch action.Type {
	case filepicker.ActionCancel:
		c.filePickerOpen = false
		c.importCalendar = 0
		c.status = "Cancelled"
		return c, nil
	case filepicker.ActionSelect:
		return c.importSelectedICS(action.Path)
	}
	if c.filePicker.Err != nil {
		c.status = fmt.Sprintf("File picker: %v", c.filePicker.Err)
	} else if c.filePicker.Filtering {
		c.status = "ICS file filter: " + c.filePicker.Filter
	} else {
		c.status = "Select .ics file"
	}
	return c, nil
}

func (c Calendar) importSelectedICS(path string) (Screen, tea.Cmd) {
	if c.importCalendar <= 0 {
		c.filePickerOpen = false
		c.status = "Select a calendar to import ICS"
		return c, nil
	}
	if strings.ToLower(strings.TrimSpace(path)) == "" || !strings.HasSuffix(strings.ToLower(path), ".ics") {
		c.status = "Select an .ics file"
		return c, nil
	}
	calendarID := c.importCalendar
	c.filePickerOpen = false
	c.importCalendar = 0
	c.loading = true
	c.status = "Importing ICS..."
	return c, c.importICSCmd(calendarID, path)
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

func (c Calendar) detailView() string {
	item, ok := c.selected()
	if !ok {
		return "No event selected.\n"
	}
	lines := []string{
		item.Title,
		"",
		"Event ID: " + strconv.FormatInt(item.EventID, 10),
		"Calendar ID: " + strconv.FormatInt(item.CalendarID, 10),
		"Starts: " + item.StartsAt.Format("2006-01-02 15:04"),
		"Ends: " + item.EndsAt.Format("2006-01-02 15:04"),
		"All day: " + strconv.FormatBool(item.AllDay),
		"Location: " + item.Location,
		"Status: " + item.Status,
	}
	if event, err := c.store.ReadEvent(item.EventID); err == nil {
		lines = append(lines, "")
		lines = append(lines, linkedMessagesView(event.Meta.Messages)...)
	} else {
		lines = append(lines, "", "Linked messages: unavailable in cache")
	}
	return strings.Join(lines, "\n") + "\n"
}

func linkedMessagesView(messages []calendarstore.MessageMeta) []string {
	if len(messages) == 0 {
		return []string{"Linked messages: none"}
	}
	lines := []string{"Linked messages:"}
	for _, message := range messages {
		lines = append(lines, fmt.Sprintf("- %s | %s | %s | inbox:%d | %s", emptyDash(message.Subject), calendarMessageSender(message), formatCalendarMessageTime(message.ReceivedAt), message.InboxID, emptyDash(message.SystemState)))
	}
	return lines
}

func calendarMessageSender(message calendarstore.MessageMeta) string {
	if strings.TrimSpace(message.SenderDisplay) != "" {
		return message.SenderDisplay
	}
	if strings.TrimSpace(message.FromName) != "" && strings.TrimSpace(message.FromAddress) != "" {
		return fmt.Sprintf("%s <%s>", message.FromName, message.FromAddress)
	}
	if strings.TrimSpace(message.FromAddress) != "" {
		return message.FromAddress
	}
	return "-"
}

func formatCalendarMessageTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Format("2006-01-02 15:04")
}

func emptyDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func (c Calendar) calendarListView() string {
	if len(c.calendars) == 0 {
		return "No cached calendars found. Press S to sync or n to create one.\n"
	}
	var b strings.Builder
	for i, item := range c.calendars {
		cursor := "  "
		if i == c.calendarIndex {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("%s%s  %s  %s  pos:%d  %s\n", cursor, item.Name, item.Color, item.TimeZone, item.Position, item.Source))
	}
	if item, ok := c.selectedCalendar(); ok {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Calendar ID: %d\nName: %s\nColor: %s\nTime zone: %s\nPosition: %d\nSource: %s\n", item.RemoteID, item.Name, item.Color, item.TimeZone, item.Position, item.Source))
	}
	return b.String()
}

func (c Calendar) modeTitle() string {
	if c.mode == calendarViewCalendars {
		return "Calendars"
	}
	return "Agenda"
}

func (c *Calendar) toggleMode() {
	c.detail = false
	if c.mode == calendarViewCalendars {
		c.mode = calendarViewAgenda
		c.status = "Showing agenda"
		return
	}
	c.mode = calendarViewCalendars
	c.status = "Showing calendars"
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

func (c Calendar) selectedCalendar() (calendarstore.CalendarMeta, bool) {
	if c.calendarIndex < 0 || c.calendarIndex >= len(c.calendars) {
		return calendarstore.CalendarMeta{}, false
	}
	return c.calendars[c.calendarIndex], true
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
	if c.calendarIndex < 0 {
		c.calendarIndex = 0
	}
	if c.calendarIndex >= len(c.calendars) && len(c.calendars) > 0 {
		c.calendarIndex = len(c.calendars) - 1
	}
	if len(c.calendars) == 0 {
		c.calendarIndex = 0
	}
}

func (c Calendar) loadCmd() tea.Cmd {
	return func() tea.Msg { return c.load() }

}

func (c Calendar) load() calendarLoadedMsg {
	items, err := c.store.ListOccurrences()
	if err != nil {
		return calendarLoadedMsg{err: err}
	}
	calendars, err := c.store.ListCalendars()
	return calendarLoadedMsg{items: items, calendars: calendars, err: err}
}

func (c Calendar) syncCmd() tea.Cmd {
	return func() tea.Msg {
		result, err := c.sync(context.Background())
		loaded := calendarLoadedMsg{}
		if err == nil {
			loaded = c.load()
			err = loaded.err
		}
		return calendarSyncedMsg{result: result, loaded: loaded, err: err}
	}
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
	keys.Input.Prev = huhkey.NewBinding(huhkey.WithKeys("up", "k", "shift+tab"), huhkey.WithHelp("up/k", "previous"))
	keys.Input.Next = huhkey.NewBinding(huhkey.WithKeys("down", "j", "tab", "enter"), huhkey.WithHelp("down/j", "next"))
	keys.Confirm.Prev = huhkey.NewBinding(huhkey.WithKeys("up", "k", "shift+tab"), huhkey.WithHelp("up/k", "previous"))
	keys.Confirm.Next = huhkey.NewBinding(huhkey.WithKeys("down", "j", "tab", "enter"), huhkey.WithHelp("down/j", "next"))
	keys.Note.Prev = huhkey.NewBinding(huhkey.WithKeys("up", "k", "shift+tab"), huhkey.WithHelp("up/k", "previous"))
	keys.Note.Next = huhkey.NewBinding(huhkey.WithKeys("down", "j", "tab", "enter"), huhkey.WithHelp("down/j", "next"))
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
