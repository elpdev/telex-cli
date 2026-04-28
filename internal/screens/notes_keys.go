package screens

import (
	"charm.land/bubbles/v2/key"
)

func DefaultNotesKeyMap() NotesKeyMap {
	return NotesKeyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "item up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "item down")),
		Open:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Back:    key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload cache")),
		Sync:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sync notes")),
		Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		New:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new note")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit note")),
		Delete:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete note")),
		Order:   key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "sort order")),
		Flat:    key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "flat view")),
	}
}

func (n Notes) KeyBindings() []key.Binding {
	return []key.Binding{n.keys.Up, n.keys.Down, n.keys.Open, n.keys.Back, n.keys.Refresh, n.keys.Sync, n.keys.Search, n.keys.New, n.keys.Edit, n.keys.Delete, n.keys.Order, n.keys.Flat}
}
