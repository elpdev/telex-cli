package screens

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/telex-cli/internal/calendarstore"
)

type CalendarSyncFunc func(context.Context) (CalendarSyncResult, error)
type DeleteCalendarEventFunc func(context.Context, int64) error

type CalendarSyncResult struct {
	Calendars   int
	Events      int
	Occurrences int
}

type Calendar struct {
	store   calendarstore.Store
	sync    CalendarSyncFunc
	delete  DeleteCalendarEventFunc
	items   []calendarstore.OccurrenceMeta
	index   int
	detail  bool
	confirm string
	loading bool
	syncing bool
	err     error
	status  string
	keys    CalendarKeyMap
}

type CalendarKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Open    key.Binding
	Back    key.Binding
	Refresh key.Binding
	Sync    key.Binding
	Today   key.Binding
	Delete  key.Binding
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

func (c Calendar) WithActions(delete DeleteCalendarEventFunc) Calendar {
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
		Delete:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete event")),
	}
}

func (c Calendar) Init() tea.Cmd { return c.loadCmd() }

func (c Calendar) Update(msg tea.Msg) (Screen, tea.Cmd) {
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
	return []key.Binding{c.keys.Up, c.keys.Down, c.keys.Open, c.keys.Back, c.keys.Refresh, c.keys.Sync, c.keys.Today, c.keys.Delete}
}

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
	if key.Matches(msg, c.keys.Delete) {
		return c.handleAction("delete")
	}
	return c, nil
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
