package screens

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

func (d Drive) loadCmd(path string) tea.Cmd {
	return func() tea.Msg {
		return d.load(path)
	}
}

func (d Drive) load(path string) driveLoadedMsg {
	entries, err := d.store.List(path)
	return driveLoadedMsg{path: path, entries: entries, err: err}
}

func (d Drive) syncCmd() tea.Cmd {
	path := d.path
	return func() tea.Msg {
		result, err := d.sync(context.Background())
		entries, loadErr := d.store.List(path)
		if err == nil {
			err = loadErr
		}
		return driveSyncedMsg{result: result, loaded: driveLoadedMsg{path: path, entries: entries, err: loadErr}, err: err}
	}
}
