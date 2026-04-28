package screens

import (
	"charm.land/bubbles/v2/key"
)

func defaultSettingsKeyMap() settingsKeyMap {
	return settingsKeyMap{
		Up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "previous row")),
		Down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "next row")),
		Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "activate")),
		Back:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
}

func (s Settings) KeyBindings() []key.Binding {
	return []key.Binding{s.keys.Up, s.keys.Down, s.keys.Enter, s.keys.Back}
}
