package screens

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/api"
	"github.com/elpdev/telex-cli/internal/calendar"
	"github.com/elpdev/telex-cli/internal/calendarstore"
	"github.com/elpdev/telex-cli/internal/components/filepicker"
)

type CalendarSyncFunc func(context.Context, string, string) (CalendarSyncResult, error)
type CreateCalendarFunc func(context.Context, calendar.CalendarInput) (*calendar.Calendar, error)
type UpdateCalendarFunc func(context.Context, int64, calendar.CalendarInput) (*calendar.Calendar, error)
type DeleteCalendarFunc func(context.Context, int64) error
type ImportICSFunc func(context.Context, int64, string) (*calendar.ImportResult, error)
type CreateCalendarEventFunc func(context.Context, calendar.CalendarEventInput) (*calendar.CalendarEvent, error)
type UpdateCalendarEventFunc func(context.Context, int64, calendar.CalendarEventInput) (*calendar.CalendarEvent, error)
type DeleteCalendarEventFunc func(context.Context, int64) error
type ShowInvitationFunc func(context.Context, int64) (*calendar.Invitation, error)
type SyncInvitationFunc func(context.Context, int64) (*calendar.Invitation, error)
type RespondInvitationFunc func(context.Context, int64, calendar.InvitationInput) (*calendar.Invitation, error)

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

const calendarDetailMaxAttendees = 12

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
	showInvite     ShowInvitationFunc
	syncInvite     SyncInvitationFunc
	respondInvite  RespondInvitationFunc
	allItems       []calendarstore.OccurrenceMeta
	items          []calendarstore.OccurrenceMeta
	calendars      []calendarstore.CalendarMeta
	rangeStart     time.Time
	rangeEnd       time.Time
	mode           calendarViewMode
	index          int
	calendarIndex  int
	detail         bool
	filtering      bool
	filterInput    string
	filter         calendarAgendaFilter
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
	syncErr        error
	status         string
	lastSynced     time.Time
	cachedEvents   int
	invitation     *calendar.Invitation
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
	Prev    key.Binding
	Next    key.Binding
	View    key.Binding
	New     key.Binding
	Edit    key.Binding
	Delete  key.Binding
	Import  key.Binding
	Filter  key.Binding
	Clear   key.Binding
}

type calendarAgendaFilter struct {
	Calendar string
	Status   string
	Source   string
	Text     string
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
	items        []calendarstore.OccurrenceMeta
	calendars    []calendarstore.CalendarMeta
	lastSynced   time.Time
	cachedEvents int
	err          error
}

type calendarSyncedMsg struct {
	result  CalendarSyncResult
	loaded  calendarLoadedMsg
	syncErr error
}

type calendarActionFinishedMsg struct {
	status     string
	loaded     calendarLoadedMsg
	invitation *calendar.Invitation
	err        error
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

func (c Calendar) WithInvitationActions(show ShowInvitationFunc, sync SyncInvitationFunc, respond RespondInvitationFunc) Calendar {
	c.showInvite = show
	c.syncInvite = sync
	c.respondInvite = respond
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
		Prev:    key.NewBinding(key.WithKeys("["), key.WithHelp("[", "previous range")),
		Next:    key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next range")),
		View:    key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "agenda/calendars")),
		New:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new event")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit event")),
		Delete:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete event")),
		Import:  key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "import ics")),
		Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter agenda")),
		Clear:   key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "clear filters")),
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
			c.allItems = msg.items
			c.calendars = msg.calendars
			c.lastSynced = msg.lastSynced
			c.cachedEvents = msg.cachedEvents
			c.applyAgendaFilter()
		}
		return c, nil
	case calendarSyncedMsg:
		c.syncing = false
		c.syncErr = msg.syncErr
		if msg.loaded.err == nil {
			c.err = nil
			c.allItems = msg.loaded.items
			c.calendars = msg.loaded.calendars
			c.lastSynced = msg.loaded.lastSynced
			c.cachedEvents = msg.loaded.cachedEvents
			c.applyAgendaFilter()
		} else if msg.syncErr == nil {
			c.err = msg.loaded.err
			c.status = ""
			return c, nil
		}
		if msg.syncErr != nil {
			if msg.loaded.err != nil {
				c.err = msg.loaded.err
			}
			c.status = ""
			return c, nil
		}
		c.status = fmt.Sprintf("Synced %d calendar(s), %d event(s), %d occurrence(s)", msg.result.Calendars, msg.result.Events, msg.result.Occurrences)
		return c, nil
	case calendarActionFinishedMsg:
		c.loading = false
		c.err = msg.err
		if msg.err != nil {
			c.status = fmt.Sprintf("Calendar action failed: %v", msg.err)
			return c, nil
		}
		c.status = msg.status
		c.allItems = msg.loaded.items
		c.calendars = msg.loaded.calendars
		c.invitation = msg.invitation
		if msg.invitation != nil {
			c.detail = true
		} else {
			c.detail = false
		}
		c.form = nil
		c.formKind = calendarFormNone
		c.filePickerOpen = false
		c.importCalendar = 0
		c.confirm = ""
		c.confirmAction = ""
		c.confirmID = 0
		c.applyAgendaFilter()
		return c, nil
	case CalendarActionMsg:
		c.syncErr = nil
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
		return style.Render(calendarCacheErrorView(c.err))
	}
	var b strings.Builder
	b.WriteString("Calendar / " + c.modeTitle() + "\n")
	if c.status != "" {
		b.WriteString(c.status + "\n")
	}
	if c.mode == calendarViewAgenda {
		b.WriteString("Range: " + c.rangeLabel() + "\n")
	}
	if cache := c.cacheStatusLine(); cache != "" {
		b.WriteString(cache + "\n")
	}
	if c.syncErr != nil {
		b.WriteString(calendarRemoteErrorStatus(c.syncErr) + "\n")
	}
	if c.syncing {
		b.WriteString("Syncing remote Calendar...\n")
	}
	if c.mode == calendarViewAgenda && c.filtering {
		b.WriteString("Filter: " + c.filterInput + "\n")
		b.WriteString("Hint: calendar:<name|id> status:<value> source:<value> text terms\n")
	} else if c.mode == calendarViewAgenda && c.filter.active() {
		b.WriteString(fmt.Sprintf("Filters: %s (%d/%d)\n", c.filter.summary(), len(c.items), len(c.agendaSourceItems())))
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
	if len(c.items) == 0 && c.filter.active() {
		b.WriteString("No calendar occurrences match the active filters. Press ctrl+l to clear filters.\n")
		return style.Render(b.String())
	}
	if len(c.items) == 0 {
		b.WriteString(c.emptyAgendaView())
		return style.Render(b.String())
	}
	for i, item := range c.items {
		cursor := "  "
		if i == c.index {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("%s%s  %s  %s  %s\n", cursor, item.StartsAt.Format("Jan 02 15:04"), c.agendaCalendarMarker(item.CalendarID), item.Title, item.Status))
	}
	return style.Render(b.String())
}

func (c Calendar) Title() string { return "Calendar" }

func (c Calendar) KeyBindings() []key.Binding {
	return []key.Binding{c.keys.Up, c.keys.Down, c.keys.Open, c.keys.Back, c.keys.Refresh, c.keys.Sync, c.keys.Today, c.keys.Prev, c.keys.Next, c.keys.View, c.keys.New, c.keys.Edit, c.keys.Delete, c.keys.Import, c.keys.Filter, c.keys.Clear}
}

func (c Calendar) cacheStatusLine() string {
	if c.lastSynced.IsZero() {
		return ""
	}
	label := "Last synced: " + c.lastSynced.Format("2006-01-02 15:04")
	if time.Since(c.lastSynced) > 24*time.Hour {
		label += " (stale; press S to refresh)"
	}
	return label
}

func (c Calendar) emptyAgendaView() string {
	if len(c.calendars) == 0 {
		return "No calendars are cached. Press S to sync remote calendars, or press n to create a calendar.\n"
	}
	start, end := c.activeRange()
	if c.cachedEvents > 0 {
		return fmt.Sprintf("No events in this range (%s to %s). Press [ or ] to change range, t for today, or S to refresh.\n", start.Format("Jan 02, 2006"), end.AddDate(0, 0, -1).Format("Jan 02, 2006"))
	}
	return "Calendars are cached, but no events are cached yet. Press S to sync events for this range, or n to create an event.\n"
}

func calendarCacheErrorView(err error) string {
	return fmt.Sprintf("Calendar cache error: %v\n\nCheck that the local data directory is readable and writable, then press r to reload. Remote sync is available with S after the cache issue is fixed.", err)
}

func calendarRemoteErrorStatus(err error) string {
	if err == nil {
		return ""
	}
	var apiErr *api.Error
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == 401 || apiErr.StatusCode == 403 {
			return fmt.Sprintf("Calendar sync failed: authentication was rejected (%s). Run `telex auth login`, then press S.", emptyDash(apiErr.Error()))
		}
		if apiErr.StatusCode >= 500 {
			return fmt.Sprintf("Calendar sync failed: remote server error (%s). Cached data is still shown; press S to retry later.", emptyDash(apiErr.Error()))
		}
		return fmt.Sprintf("Calendar sync failed: remote API returned %d (%s). Cached data is still shown; press S to retry.", apiErr.StatusCode, emptyDash(apiErr.Error()))
	}
	message := err.Error()
	if strings.Contains(strings.ToLower(message), "config") || strings.Contains(strings.ToLower(message), "base url") || strings.Contains(strings.ToLower(message), "client") || strings.Contains(strings.ToLower(message), "secret") {
		return fmt.Sprintf("Calendar sync failed: configuration problem (%s). Open Settings and verify your Telex instance, then press S.", message)
	}
	return fmt.Sprintf("Calendar sync failed: %v. Cached data is still shown; press S to retry.", err)
}

func (c Calendar) CapturesFocusKey(tea.KeyPressMsg) bool {
	return c.form != nil || c.filePickerOpen || c.filtering
}

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
	selection := CalendarSelection{Kind: "calendar-event", Subject: item.Title, HasItem: true}
	if c.selectedInvitationMessageID() > 0 {
		selection.HasInvitation = true
	}
	return selection
}

type CalendarSelection struct {
	Kind          string
	Subject       string
	HasItem       bool
	HasInvitation bool
}

func (c Calendar) handleAction(action string) (Screen, tea.Cmd) {
	if c.confirm != "" || c.form != nil {
		return c, nil
	}
	switch action {
	case "filter":
		if c.mode != calendarViewAgenda {
			c.mode = calendarViewAgenda
			c.detail = false
		}
		c.filtering = true
		c.filterInput = c.filter.inputString()
		c.status = "Filter agenda"
		return c, nil
	case "clear-filter":
		c.filtering = false
		c.filterInput = ""
		c.filter = calendarAgendaFilter{}
		c.applyAgendaFilter()
		c.status = "Agenda filters cleared"
		return c, nil
	case "sync":
		if c.sync == nil || c.syncing {
			if c.sync == nil {
				c.status = "Calendar sync is not configured. Open Settings and verify your Telex instance, then run `telex auth login`."
			}
			return c, nil
		}
		c.syncing = true
		c.syncErr = nil
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
		c.jumpToTodayRange()
		c.loading = true
		c.status = "Showing " + c.rangeLabel()
		return c, c.loadCmd()
	case "previous-range":
		c.shiftRange(-1)
		c.loading = true
		c.status = "Showing " + c.rangeLabel()
		return c, c.loadCmd()
	case "next-range":
		c.shiftRange(1)
		c.loading = true
		c.status = "Showing " + c.rangeLabel()
		return c, c.loadCmd()
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
	case "invitation-show":
		messageID := c.selectedInvitationMessageID()
		if messageID <= 0 {
			c.status = "Selected event has no linked invitation message"
			return c, nil
		}
		if c.showInvite == nil {
			c.status = "Invitation details are not configured"
			return c, nil
		}
		c.loading = true
		c.detail = true
		return c, c.invitationCmd(messageID, "show", "")
	case "invitation-sync":
		messageID := c.selectedInvitationMessageID()
		if messageID <= 0 {
			c.status = "Selected event has no linked invitation message"
			return c, nil
		}
		if c.syncInvite == nil {
			c.status = "Invitation sync is not configured"
			return c, nil
		}
		c.loading = true
		return c, c.invitationCmd(messageID, "sync", "")
	case "invitation-accepted", "invitation-tentative", "invitation-declined", "invitation-needs-action":
		messageID := c.selectedInvitationMessageID()
		if messageID <= 0 {
			c.status = "Selected event has no linked invitation message"
			return c, nil
		}
		if c.respondInvite == nil {
			c.status = "Invitation responses are not configured"
			return c, nil
		}
		c.loading = true
		return c, c.invitationCmd(messageID, "respond", invitationStatusFromAction(action))
	}
	return c, nil
}

func (c Calendar) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if c.filtering {
		return c.handleFilterKey(msg)
	}
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
	if key.Matches(msg, c.keys.Prev) && c.mode == calendarViewAgenda && !c.detail {
		return c.handleAction("previous-range")
	}
	if key.Matches(msg, c.keys.Next) && c.mode == calendarViewAgenda && !c.detail {
		return c.handleAction("next-range")
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
	if key.Matches(msg, c.keys.Filter) && c.mode == calendarViewAgenda && !c.detail {
		return c.handleAction("filter")
	}
	if key.Matches(msg, c.keys.Clear) && c.mode == calendarViewAgenda && c.filter.active() {
		return c.handleAction("clear-filter")
	}
	return c, nil
}

func (c Calendar) handleFilterKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		c.filtering = false
		c.filterInput = ""
		c.status = "Filter cancelled"
		return c, nil
	case "enter":
		c.filtering = false
		c.filter = parseCalendarAgendaFilter(c.filterInput)
		c.filterInput = ""
		c.applyAgendaFilter()
		if c.filter.active() {
			c.status = fmt.Sprintf("Filtered agenda: %d occurrence(s)", len(c.items))
		} else {
			c.status = "Agenda filters cleared"
		}
		return c, nil
	case "backspace":
		if len(c.filterInput) > 0 {
			c.filterInput = c.filterInput[:len(c.filterInput)-1]
		}
		return c, nil
	case "ctrl+u":
		c.filterInput = ""
		return c, nil
	}
	if msg.Text != "" {
		c.filterInput += msg.Text
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
		if c.invitation != nil {
			return strings.Join(invitationView(*c.invitation), "\n") + "\n"
		}
		return "No event selected.\n"
	}
	cal, hasCalendar := c.calendarByID(item.CalendarID)
	event, err := c.store.ReadEvent(item.EventID)
	if err != nil {
		lines := occurrenceDetailLines(item, cal, hasCalendar)
		lines = append(lines, "", "Cached event details: unavailable")
		return strings.Join(lines, "\n") + "\n"
	}
	messageID := firstEventMessageID(event.Meta)
	if !hasCalendar && event.Meta.CalendarID != item.CalendarID {
		cal, hasCalendar = c.calendarByID(event.Meta.CalendarID)
	}
	lines := cachedEventDetailLines(*event, cal, hasCalendar)
	if c.invitation != nil && c.invitation.MessageID == messageID {
		lines = append(lines, "")
		lines = append(lines, invitationView(*c.invitation)...)
	}
	return strings.Join(lines, "\n") + "\n"
}

func occurrenceDetailLines(item calendarstore.OccurrenceMeta, cal calendarstore.CalendarMeta, hasCalendar bool) []string {
	return []string{
		item.Title,
		"",
		"Event ID: " + strconv.FormatInt(item.EventID, 10),
		"Calendar ID: " + strconv.FormatInt(item.CalendarID, 10),
		"Calendar: " + calendarDetailName(item.CalendarID, cal, hasCalendar),
		"Calendar color: " + calendarDetailColor(cal, hasCalendar),
		"Calendar time zone: " + calendarDetailTimeZone(cal, hasCalendar, ""),
		"Starts: " + item.StartsAt.Format("2006-01-02 15:04"),
		"Ends: " + item.EndsAt.Format("2006-01-02 15:04"),
		"All day: " + strconv.FormatBool(item.AllDay),
		"Location: " + item.Location,
		"Status: " + item.Status,
	}
}

func cachedEventDetailLines(event calendarstore.CachedEvent, cal calendarstore.CalendarMeta, hasCalendar bool) []string {
	meta := event.Meta
	lines := []string{
		meta.Title,
		"",
		"Event ID: " + strconv.FormatInt(meta.RemoteID, 10),
		"Calendar ID: " + strconv.FormatInt(meta.CalendarID, 10),
		"Calendar: " + calendarDetailName(meta.CalendarID, cal, hasCalendar),
		"Calendar color: " + calendarDetailColor(cal, hasCalendar),
		"Calendar time zone: " + calendarDetailTimeZone(cal, hasCalendar, meta.TimeZone),
		"Starts: " + meta.StartsAt.Format("2006-01-02 15:04"),
		"Ends: " + meta.EndsAt.Format("2006-01-02 15:04"),
		"All day: " + strconv.FormatBool(meta.AllDay),
		"Event time zone: " + emptyDash(meta.TimeZone),
		"Location: " + emptyDash(meta.Location),
		"Status: " + emptyDash(meta.Status),
	}
	lines = append(lines, descriptionView(event.Description)...)
	lines = append(lines, eventOrganizerView(meta)...)
	lines = append(lines, recurrenceView(meta)...)
	lines = append(lines, attendeeListView(meta.Attendees, meta.CurrentUserAttendee)...)
	lines = append(lines, linkListView(meta.Links)...)
	lines = append(lines, messageSummaryView(meta.Messages)...)
	if meta.Invitation {
		lines = append(lines, "", "Invitation: true")
	}
	return lines
}

func descriptionView(description string) []string {
	description = strings.TrimSpace(description)
	if description == "" {
		return nil
	}
	lines := []string{"", "Description:"}
	for _, line := range strings.Split(description, "\n") {
		lines = append(lines, strings.TrimRight(line, " \t"))
	}
	return lines
}

func eventOrganizerView(event calendarstore.EventMeta) []string {
	if event.OrganizerName == "" && event.OrganizerEmail == "" {
		return nil
	}
	return []string{"", "Organizer: " + organizerDisplay(event)}
}

func recurrenceView(event calendarstore.EventMeta) []string {
	if event.RecurrenceSummary == "" && event.RecurrenceRule == "" && len(event.NextOccurrences) == 0 && len(event.RecurrenceExceptions) == 0 {
		return nil
	}
	lines := []string{"", "Recurrence:"}
	if event.RecurrenceSummary != "" {
		lines = append(lines, "Summary: "+event.RecurrenceSummary)
	}
	if event.RecurrenceRule != "" {
		lines = append(lines, "Rule: "+event.RecurrenceRule)
	}
	if len(event.NextOccurrences) > 0 {
		lines = append(lines, fmt.Sprintf("Next occurrences: %d", len(event.NextOccurrences)))
		for _, occurrence := range event.NextOccurrences {
			lines = append(lines, "- "+formatCalendarMessageTime(occurrence))
		}
	}
	if len(event.RecurrenceExceptions) > 0 {
		lines = append(lines, fmt.Sprintf("Exceptions: %d", len(event.RecurrenceExceptions)))
		for _, exception := range event.RecurrenceExceptions {
			lines = append(lines, "- "+emptyDash(exception))
		}
	}
	return lines
}

func attendeeListView(attendees []calendarstore.AttendeeMeta, current *calendarstore.AttendeeMeta) []string {
	if len(attendees) == 0 {
		lines := []string{"", "Attendees: none"}
		if current != nil {
			lines = append(lines, "Current attendee: "+attendeeSummary(*current, false))
		}
		return lines
	}
	lines := []string{"", fmt.Sprintf("Attendees: %d", len(attendees))}
	if current != nil {
		lines = append(lines, "Current attendee: "+attendeeSummary(*current, false))
	}
	displayed := displayedAttendees(attendees, current)
	for _, attendee := range displayed {
		lines = append(lines, "- "+attendeeSummary(attendee, attendeeMatchesCurrent(attendee, current)))
	}
	if len(attendees) > len(displayed) {
		lines = append(lines, fmt.Sprintf("... %d more attendee(s) not shown", len(attendees)-len(displayed)))
	}
	return lines
}

func displayedAttendees(attendees []calendarstore.AttendeeMeta, current *calendarstore.AttendeeMeta) []calendarstore.AttendeeMeta {
	limit := min(len(attendees), calendarDetailMaxAttendees)
	displayed := append([]calendarstore.AttendeeMeta(nil), attendees[:limit]...)
	if current == nil || len(attendees) <= limit || attendeeListContains(displayed, current) {
		return displayed
	}
	for _, attendee := range attendees[limit:] {
		if attendeeMatchesCurrent(attendee, current) {
			displayed[len(displayed)-1] = attendee
			return displayed
		}
	}
	return displayed
}

func attendeeListContains(attendees []calendarstore.AttendeeMeta, current *calendarstore.AttendeeMeta) bool {
	for _, attendee := range attendees {
		if attendeeMatchesCurrent(attendee, current) {
			return true
		}
	}
	return false
}

func attendeeMatchesCurrent(attendee calendarstore.AttendeeMeta, current *calendarstore.AttendeeMeta) bool {
	if current == nil {
		return false
	}
	if current.ID != 0 && attendee.ID == current.ID {
		return true
	}
	return strings.TrimSpace(current.Email) != "" && strings.EqualFold(strings.TrimSpace(attendee.Email), strings.TrimSpace(current.Email))
}

func attendeeSummary(attendee calendarstore.AttendeeMeta, current bool) string {
	display := attendeeDisplay(attendee)
	if current {
		display += " [you]"
	}
	return fmt.Sprintf("%s | role:%s | status:%s | response requested:%t", display, emptyDash(attendee.Role), emptyDash(attendee.ParticipationStatus), attendee.ResponseRequested)
}

func linkListView(links []calendarstore.LinkMeta) []string {
	if len(links) == 0 {
		return []string{"", "Links: none"}
	}
	lines := []string{"", fmt.Sprintf("Links: %d", len(links))}
	for _, link := range links {
		lines = append(lines, fmt.Sprintf("- message:%d | uid:%s | method:%s | sequence:%d", link.MessageID, emptyDash(link.ICalUID), emptyDash(link.ICalMethod), link.SequenceNumber))
	}
	return lines
}

func messageSummaryView(messages []calendarstore.MessageMeta) []string {
	if len(messages) == 0 {
		return []string{"", "Messages: none"}
	}
	lines := []string{"", fmt.Sprintf("Messages: %d", len(messages))}
	for _, message := range messages {
		summary := fmt.Sprintf("- %s | %s | %s | inbox:%d | %s", emptyDash(message.Subject), calendarMessageSender(message), formatCalendarMessageTime(message.ReceivedAt), message.InboxID, emptyDash(message.SystemState))
		if strings.TrimSpace(message.PreviewText) != "" {
			summary += " | " + strings.TrimSpace(message.PreviewText)
		}
		lines = append(lines, summary)
	}
	return lines
}

func invitationView(invite calendar.Invitation) []string {
	lines := []string{"Invitation details:", "Message ID: " + strconv.FormatInt(invite.MessageID, 10), "Available: " + strconv.FormatBool(invite.Available)}
	if invite.CalendarEvent != nil {
		lines = append(lines, "Event: "+invite.CalendarEvent.Title, "Event ID: "+strconv.FormatInt(invite.CalendarEvent.ID, 10))
	}
	if invite.CurrentUserAttendee != nil {
		lines = append(lines, "Current response: "+emptyDash(invite.CurrentUserAttendee.ParticipationStatus))
	}
	return lines
}

func attendeeDisplay(attendee calendarstore.AttendeeMeta) string {
	if strings.TrimSpace(attendee.Name) != "" && strings.TrimSpace(attendee.Email) != "" {
		return fmt.Sprintf("%s <%s>", attendee.Name, attendee.Email)
	}
	if strings.TrimSpace(attendee.Email) != "" {
		return attendee.Email
	}
	return emptyDash(attendee.Name)
}

func organizerDisplay(event calendarstore.EventMeta) string {
	name := strings.TrimSpace(event.OrganizerName)
	email := strings.TrimSpace(event.OrganizerEmail)
	if name != "" && email != "" {
		return fmt.Sprintf("%s <%s>", name, email)
	}
	if email != "" {
		return email
	}
	return emptyDash(name)
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
		return "No calendars are cached. Press S to sync remote calendars, or press n to create one. If sync fails, run `telex auth login` and verify Settings.\n"
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

func (c Calendar) agendaCalendarMarker(calendarID int64) string {
	cal, ok := c.calendarByID(calendarID)
	label := calendarRowLabel(calendarID, cal, ok)
	color := strings.TrimSpace(cal.Color)
	marker := "##"
	if color != "" {
		marker = lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(marker)
	}
	return marker + " " + label
}

func (c Calendar) calendarByID(calendarID int64) (calendarstore.CalendarMeta, bool) {
	for _, cal := range c.calendars {
		if cal.RemoteID == calendarID {
			return cal, true
		}
	}
	return calendarstore.CalendarMeta{}, false
}

func calendarRowLabel(calendarID int64, cal calendarstore.CalendarMeta, ok bool) string {
	name := ""
	if ok {
		name = strings.TrimSpace(cal.Name)
	}
	if name != "" {
		return name
	}
	if calendarID > 0 {
		return "calendar:" + strconv.FormatInt(calendarID, 10)
	}
	return "calendar:-"
}

func calendarDetailName(calendarID int64, cal calendarstore.CalendarMeta, ok bool) string {
	name := ""
	if ok {
		name = strings.TrimSpace(cal.Name)
	}
	if name != "" {
		return name
	}
	if calendarID > 0 {
		return "#" + strconv.FormatInt(calendarID, 10)
	}
	return "-"
}

func calendarDetailColor(cal calendarstore.CalendarMeta, ok bool) string {
	if !ok {
		return "-"
	}
	return emptyDash(cal.Color)
}

func calendarDetailTimeZone(cal calendarstore.CalendarMeta, ok bool, fallback string) string {
	if ok && strings.TrimSpace(cal.TimeZone) != "" {
		return strings.TrimSpace(cal.TimeZone)
	}
	return emptyDash(fallback)
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

func (c *Calendar) jumpToTodayRange() {
	today := calendarRangeDate(time.Now())
	c.rangeStart = today
	c.rangeEnd = today.AddDate(0, 0, 30)
	c.detail = false
}

func (c *Calendar) shiftRange(direction int) {
	start, end := c.activeRange()
	duration := end.Sub(start)
	if duration <= 0 {
		duration = 30 * 24 * time.Hour
	}
	shift := time.Duration(direction) * duration
	c.rangeStart = start.Add(shift)
	c.rangeEnd = end.Add(shift)
	c.index = 0
	c.detail = false
}

func (c Calendar) activeRange() (time.Time, time.Time) {
	start := c.rangeStart
	end := c.rangeEnd
	if start.IsZero() {
		start = calendarRangeDate(time.Now())
	}
	if end.IsZero() || !end.After(start) {
		end = start.AddDate(0, 0, 30)
	}
	return start, end
}

func (c Calendar) rangeDates() (string, string) {
	start, end := c.activeRange()
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

func (c Calendar) rangeLabel() string {
	start, end := c.activeRange()
	return fmt.Sprintf("%s to %s", start.Format("Jan 02, 2006"), end.AddDate(0, 0, -1).Format("Jan 02, 2006"))
}

func calendarRangeDate(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func (c *Calendar) applyAgendaFilter() {
	source := c.agendaSourceItems()
	if !c.filter.active() {
		c.items = append([]calendarstore.OccurrenceMeta(nil), source...)
		c.clampIndex()
		return
	}
	items := make([]calendarstore.OccurrenceMeta, 0, len(source))
	for _, item := range source {
		if c.occurrenceMatchesFilter(item) {
			items = append(items, item)
		}
	}
	c.items = items
	c.index = 0
	c.clampIndex()
}

func (c Calendar) agendaSourceItems() []calendarstore.OccurrenceMeta {
	if c.allItems != nil {
		return c.allItems
	}
	return c.items
}

func (c Calendar) occurrenceMatchesFilter(item calendarstore.OccurrenceMeta) bool {
	if !calendarFilterMatch(c.filter.Status, item.Status) {
		return false
	}
	if !calendarFilterMatch(c.filter.Calendar, c.calendarFilterValue(item.CalendarID)) {
		return false
	}
	if !calendarFilterMatch(c.filter.Source, c.occurrenceSource(item)) {
		return false
	}
	if c.filter.Text != "" && !calendarTextMatch(item, c.filter.Text) {
		return false
	}
	return true
}

func (c Calendar) calendarFilterValue(calendarID int64) string {
	values := []string{strconv.FormatInt(calendarID, 10)}
	for _, cal := range c.calendars {
		if cal.RemoteID == calendarID {
			values = append(values, cal.Name)
			break
		}
	}
	return strings.Join(values, " ")
}

func (c Calendar) occurrenceSource(item calendarstore.OccurrenceMeta) string {
	if strings.TrimSpace(item.Source) != "" {
		return item.Source
	}
	event, err := c.store.ReadEvent(item.EventID)
	if err != nil || event == nil {
		return ""
	}
	return event.Meta.Source
}

func calendarFilterMatch(needle, haystack string) bool {
	needle = strings.ToLower(strings.TrimSpace(needle))
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(haystack), needle)
}

func calendarTextMatch(item calendarstore.OccurrenceMeta, text string) bool {
	haystack := strings.ToLower(item.Title + " " + item.Location)
	for _, term := range strings.Fields(strings.ToLower(strings.TrimSpace(text))) {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

func parseCalendarAgendaFilter(input string) calendarAgendaFilter {
	filter := calendarAgendaFilter{}
	text := []string{}
	for _, token := range strings.Fields(input) {
		key, value, ok := strings.Cut(token, ":")
		if !ok || strings.TrimSpace(value) == "" {
			text = append(text, token)
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "calendar", "cal":
			filter.Calendar = strings.TrimSpace(value)
		case "status":
			filter.Status = strings.TrimSpace(value)
		case "source", "src":
			filter.Source = strings.TrimSpace(value)
		default:
			text = append(text, token)
		}
	}
	filter.Text = strings.Join(text, " ")
	return filter
}

func (f calendarAgendaFilter) active() bool {
	return strings.TrimSpace(f.Calendar) != "" || strings.TrimSpace(f.Status) != "" || strings.TrimSpace(f.Source) != "" || strings.TrimSpace(f.Text) != ""
}

func (f calendarAgendaFilter) summary() string {
	parts := []string{}
	if strings.TrimSpace(f.Calendar) != "" {
		parts = append(parts, "calendar="+strings.TrimSpace(f.Calendar))
	}
	if strings.TrimSpace(f.Status) != "" {
		parts = append(parts, "status="+strings.TrimSpace(f.Status))
	}
	if strings.TrimSpace(f.Source) != "" {
		parts = append(parts, "source="+strings.TrimSpace(f.Source))
	}
	if strings.TrimSpace(f.Text) != "" {
		parts = append(parts, "text=\""+strings.TrimSpace(f.Text)+"\"")
	}
	return strings.Join(parts, " ")
}

func (f calendarAgendaFilter) inputString() string {
	parts := []string{}
	if strings.TrimSpace(f.Calendar) != "" {
		parts = append(parts, "calendar:"+strings.TrimSpace(f.Calendar))
	}
	if strings.TrimSpace(f.Status) != "" {
		parts = append(parts, "status:"+strings.TrimSpace(f.Status))
	}
	if strings.TrimSpace(f.Source) != "" {
		parts = append(parts, "source:"+strings.TrimSpace(f.Source))
	}
	if strings.TrimSpace(f.Text) != "" {
		parts = append(parts, strings.TrimSpace(f.Text))
	}
	return strings.Join(parts, " ")
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

func (c Calendar) selectedInvitationMessageID() int64 {
	item, ok := c.selected()
	if !ok {
		return 0
	}
	event, err := c.store.ReadEvent(item.EventID)
	if err != nil {
		return 0
	}
	return firstEventMessageID(event.Meta)
}

func firstEventMessageID(event calendarstore.EventMeta) int64 {
	for _, link := range event.Links {
		if link.MessageID > 0 {
			return link.MessageID
		}
	}
	for _, message := range event.Messages {
		if message.ID > 0 {
			return message.ID
		}
	}
	return 0
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
		if c.invitation == nil {
			c.detail = false
		}
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
	start, end := c.activeRange()
	items, err := c.store.ListOccurrencesRange(start, end)
	if err != nil {
		return calendarLoadedMsg{err: err}
	}
	calendars, err := c.store.ListCalendars()
	if err != nil {
		return calendarLoadedMsg{err: err}
	}
	events, err := c.store.ListEvents(0)
	if err != nil {
		return calendarLoadedMsg{err: err}
	}
	return calendarLoadedMsg{items: items, calendars: calendars, lastSynced: latestCalendarSync(items, calendars, events), cachedEvents: len(events)}
}

func (c Calendar) syncCmd() tea.Cmd {
	from, to := c.rangeDates()
	return func() tea.Msg {
		result, err := c.sync(context.Background(), from, to)
		loaded := c.load()
		return calendarSyncedMsg{result: result, loaded: loaded, syncErr: err}
	}
}

func latestCalendarSync(items []calendarstore.OccurrenceMeta, calendars []calendarstore.CalendarMeta, events []calendarstore.CachedEvent) time.Time {
	var latest time.Time
	for _, item := range items {
		if item.SyncedAt.After(latest) {
			latest = item.SyncedAt
		}
	}
	for _, cal := range calendars {
		if cal.SyncedAt.After(latest) {
			latest = cal.SyncedAt
		}
	}
	for _, event := range events {
		if event.Meta.SyncedAt.After(latest) {
			latest = event.Meta.SyncedAt
		}
	}
	return latest
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

func (c Calendar) invitationCmd(messageID int64, action, status string) tea.Cmd {
	return func() tea.Msg {
		var invite *calendar.Invitation
		var err error
		switch action {
		case "show":
			invite, err = c.showInvite(context.Background(), messageID)
		case "sync":
			invite, err = c.syncInvite(context.Background(), messageID)
		case "respond":
			invite, err = c.respondInvite(context.Background(), messageID, calendar.InvitationInput{ParticipationStatus: status})
		default:
			err = errors.New("unknown invitation action")
		}
		loaded := calendarLoadedMsg{}
		if err == nil {
			loaded = c.load()
			err = loaded.err
		}
		return calendarActionFinishedMsg{status: invitationActionStatus(action, status, invite), loaded: loaded, invitation: invite, err: err}
	}
}

func invitationStatusFromAction(action string) string {
	switch action {
	case "invitation-accepted":
		return "accepted"
	case "invitation-tentative":
		return "tentative"
	case "invitation-declined":
		return "declined"
	case "invitation-needs-action":
		return "needs_action"
	default:
		return ""
	}
}

func invitationActionStatus(action, status string, invite *calendar.Invitation) string {
	switch action {
	case "show":
		return "Loaded invitation details"
	case "sync":
		return "Synced invitation into Calendar"
	case "respond":
		if status != "" {
			return "Responded " + status
		}
	}
	if invite != nil && invite.CalendarEvent != nil && invite.CalendarEvent.Title != "" {
		return invite.CalendarEvent.Title
	}
	return "Updated invitation"
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
