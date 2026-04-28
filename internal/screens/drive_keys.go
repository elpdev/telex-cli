package screens

import (
	"charm.land/bubbles/v2/key"
)

func DefaultDriveKeyMap() DriveKeyMap {
	return DriveKeyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "item up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "item down")),
		Open:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open folder")),
		Back:    key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "parent")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload cache")),
		Sync:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sync drive")),
		Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Details: key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "details")),
		Upload:  key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "upload")),
		NewDir:  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new folder")),
		Rename:  key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "rename")),
		Delete:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
	}
}

func (d Drive) KeyBindings() []key.Binding {
	return []key.Binding{d.keys.Up, d.keys.Down, d.keys.Open, d.keys.Back, d.keys.Refresh, d.keys.Sync, d.keys.Search, d.keys.Details, d.keys.Upload, d.keys.NewDir, d.keys.Rename, d.keys.Delete}
}
