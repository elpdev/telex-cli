package screens

import (
	"charm.land/bubbles/v2/key"
)

func DefaultMailAdminKeyMap() MailAdminKeyMap {
	return MailAdminKeyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "move up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "move down")),
		Focus:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "domains/inboxes")),
		New:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
		Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Delete:   key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
		Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		Validate: key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "validate domain")),
		Pipeline: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pipeline")),
		Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
}

func (m MailAdmin) KeyBindings() []key.Binding {
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Focus, m.keys.New, m.keys.Edit, m.keys.Delete, m.keys.Refresh, m.keys.Validate, m.keys.Pipeline, m.keys.Back}
}
