package commands

import tea "charm.land/bubbletea/v2"

type Command struct {
	ID          string
	Title       string
	Description string
	Keywords    []string
	Run         func() tea.Cmd
}
