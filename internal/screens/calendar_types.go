package screens

import (
	"context"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/huh/v2"
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
	calendarList   list.Model
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

type calendarListItem struct {
	meta calendarstore.CalendarMeta
}

func (i calendarListItem) FilterValue() string {
	return strings.Join([]string{i.meta.Name, i.meta.Color, i.meta.TimeZone, i.meta.Source, strconv.FormatInt(i.meta.RemoteID, 10)}, " ")
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

type CalendarSelection struct {
	Kind          string
	Subject       string
	HasItem       bool
	HasInvitation bool
}
