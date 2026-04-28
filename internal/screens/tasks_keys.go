package screens

import "charm.land/bubbles/v2/key"

type TasksKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Open     key.Binding
	Back     key.Binding
	Refresh  key.Binding
	Sync     key.Binding
	Search   key.Binding
	Project  key.Binding
	New      key.Binding
	Edit     key.Binding
	Delete   key.Binding
	MoveNext key.Binding
	MovePrev key.Binding
	MoveTo   key.Binding
}

func DefaultTasksKeyMap() TasksKeyMap {
	return TasksKeyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "item up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "item down")),
		Open:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Back:     key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload cache")),
		Sync:     key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sync tasks")),
		Search:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Project:  key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "projects")),
		New:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new card/project")),
		Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit card")),
		Delete:   key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete card")),
		MoveNext: key.NewBinding(key.WithKeys(">", "L"), key.WithHelp(">/L", "move to next column")),
		MovePrev: key.NewBinding(key.WithKeys("<", "H"), key.WithHelp("</H", "move to previous column")),
		MoveTo:   key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "move to column…")),
	}
}

func (t Tasks) KeyBindings() []key.Binding {
	return []key.Binding{t.keys.Up, t.keys.Down, t.keys.Open, t.keys.Back, t.keys.Refresh, t.keys.Sync, t.keys.Search, t.keys.Project, t.keys.New, t.keys.Edit, t.keys.Delete, t.keys.MovePrev, t.keys.MoveNext, t.keys.MoveTo}
}
