package app

import "github.com/charmbracelet/bubbles/key"

type FocusArea int

const (
	FocusSidebar FocusArea = iota
	FocusMain
)

type KeyMap struct {
	Quit      key.Binding
	ForceQuit key.Binding
	Commands  key.Binding
	Help      key.Binding
	Cancel    key.Binding
	Focus     key.Binding
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:      key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		ForceQuit: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		Commands:  key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("ctrl+k", "commands")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Cancel:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
		Focus:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "focus")),
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "down")),
		Enter:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Commands, k.Focus, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Help, k.Commands, k.Focus, k.Cancel}, {k.Up, k.Down, k.Enter, k.Quit, k.ForceQuit}}
}
