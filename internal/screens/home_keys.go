package screens

import (
	"charm.land/bubbles/v2/key"
)

func DefaultHomeKeyMap() HomeKeyMap {
	return HomeKeyMap{
		Refresh:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		NextCard:   key.NewBinding(key.WithKeys("tab", "right", "l"), key.WithHelp("tab", "next card")),
		PrevCard:   key.NewBinding(key.WithKeys("shift+tab", "left", "h"), key.WithHelp("shift+tab", "prev card")),
		OpenCard:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open focused")),
		ClearFocus: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear focus")),
		Mail:       key.NewBinding(key.WithKeys("m", "1"), key.WithHelp("m/1", "open mail")),
		Calendar:   key.NewBinding(key.WithKeys("c", "2"), key.WithHelp("c/2", "open calendar")),
		Contacts:   key.NewBinding(key.WithKeys("o", "3"), key.WithHelp("o/3", "open contacts")),
		Notes:      key.NewBinding(key.WithKeys("n", "4"), key.WithHelp("n/4", "open notes")),
		Tasks:      key.NewBinding(key.WithKeys("t", "5"), key.WithHelp("t/5", "open tasks")),
		Drive:      key.NewBinding(key.WithKeys("d", "6"), key.WithHelp("d/6", "open drive")),
		News:       key.NewBinding(key.WithKeys("w", "7"), key.WithHelp("w/7", "open news")),
	}
}

func (h Home) KeyBindings() []key.Binding {
	return []key.Binding{h.keys.Mail, h.keys.Calendar, h.keys.Contacts, h.keys.Notes, h.keys.Tasks, h.keys.Drive, h.keys.News, h.keys.NextCard, h.keys.OpenCard, h.keys.Refresh}
}
