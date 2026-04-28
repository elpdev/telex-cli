package screens

import (
	"charm.land/bubbles/v2/key"
)

func DefaultContactsKeyMap() ContactsKeyMap {
	return ContactsKeyMap{
		Up:             key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "contact up")),
		Down:           key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "contact down")),
		Open:           key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Back:           key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		Refresh:        key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload cache")),
		Sync:           key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sync contacts")),
		Search:         key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Delete:         key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
		EditNote:       key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit note")),
		Note:           key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "refresh note")),
		Communications: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "communications")),
	}
}

func (c Contacts) KeyBindings() []key.Binding {
	return []key.Binding{c.keys.Up, c.keys.Down, c.keys.Open, c.keys.Back, c.keys.Refresh, c.keys.Sync, c.keys.Search, c.keys.Delete, c.keys.EditNote, c.keys.Note, c.keys.Communications}
}
