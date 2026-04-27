package screens

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type simpleDelegate struct{}

func (simpleDelegate) Height() int                         { return 1 }
func (simpleDelegate) Spacing() int                        { return 0 }
func (simpleDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func newSimpleList(items []list.Item, delegate list.ItemDelegate, selected, width, height int) list.Model {
	m := list.New(items, delegate, width, height)
	m.SetShowTitle(false)
	m.SetShowFilter(false)
	m.SetFilteringEnabled(false)
	m.SetShowStatusBar(false)
	m.SetShowHelp(false)
	m.DisableQuitKeybindings()
	if len(items) > 0 {
		m.Select(clampIndex(selected, len(items)))
	}
	return m
}

func listCursor(selected bool) string {
	if selected {
		return "> "
	}
	return "  "
}

func clampIndex(selected, length int) int {
	if selected < 0 {
		return 0
	}
	if selected >= length {
		return length - 1
	}
	return selected
}
