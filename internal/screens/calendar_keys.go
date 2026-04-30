package screens

import "charm.land/bubbles/v2/key"

type CalendarKeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Left          key.Binding
	Right         key.Binding
	Open          key.Binding
	Back          key.Binding
	Refresh       key.Binding
	Sync          key.Binding
	Today         key.Binding
	Prev          key.Binding
	Next          key.Binding
	View          key.Binding
	ViewAgenda    key.Binding
	ViewWeek      key.Binding
	ViewMonth     key.Binding
	ViewCalendars key.Binding
	New           key.Binding
	Edit          key.Binding
	Delete        key.Binding
	Import        key.Binding
	Filter        key.Binding
	Clear         key.Binding
}

func DefaultCalendarKeyMap() CalendarKeyMap {
	return CalendarKeyMap{
		Up:            key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "up")),
		Down:          key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "down")),
		Left:          key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("left/h", "left")),
		Right:         key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("right/l", "right")),
		Open:          key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
		Back:          key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		Refresh:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload cache")),
		Sync:          key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sync calendar")),
		Today:         key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "today")),
		Prev:          key.NewBinding(key.WithKeys("["), key.WithHelp("[", "previous")),
		Next:          key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next")),
		View:          key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "cycle view")),
		ViewAgenda:    key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "agenda")),
		ViewWeek:      key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "week")),
		ViewMonth:     key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "month")),
		ViewCalendars: key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "calendars")),
		New:           key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new event")),
		Edit:          key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit event")),
		Delete:        key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete event")),
		Import:        key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "import ics")),
		Filter:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter agenda")),
		Clear:         key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "clear filters")),
	}
}

func (c Calendar) KeyBindings() []key.Binding {
	return []key.Binding{c.keys.Up, c.keys.Down, c.keys.Left, c.keys.Right, c.keys.Open, c.keys.Back, c.keys.Refresh, c.keys.Sync, c.keys.Today, c.keys.Prev, c.keys.Next, c.keys.View, c.keys.ViewAgenda, c.keys.ViewWeek, c.keys.ViewMonth, c.keys.ViewCalendars, c.keys.New, c.keys.Edit, c.keys.Delete, c.keys.Import, c.keys.Filter, c.keys.Clear}
}
