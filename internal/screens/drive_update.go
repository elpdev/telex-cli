package screens

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
)

func (d Drive) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if d.pickerOpen {
		return d.handlePickerMsg(msg)
	}

	switch msg := msg.(type) {
	case driveLoadedMsg:
		d.loading = false
		d.err = msg.err
		if msg.err == nil {
			d.path = msg.path
			d.entries = msg.entries
			d.clampIndex()
			d.syncEntryList()
			d.breadcrumbs = d.pathParts()
		}
		return d, nil
	case driveSyncedMsg:
		d.syncing = false
		d.err = msg.err
		if msg.err == nil {
			d.status = fmt.Sprintf("Synced %d folder(s), %d file(s), downloaded %d", msg.result.Folders, msg.result.Files, msg.result.DownloadedFiles)
			if msg.result.DownloadFailures > 0 {
				d.status += fmt.Sprintf("; %d download warning(s)", msg.result.DownloadFailures)
			}
			d.path = msg.loaded.path
			d.entries = msg.loaded.entries
			d.clampIndex()
			d.syncEntryList()
			d.breadcrumbs = d.pathParts()
		} else {
			d.status = ""
		}
		return d, nil
	case driveActionFinishedMsg:
		d.loading = false
		d.err = msg.err
		if msg.err != nil {
			d.status = fmt.Sprintf("Drive action failed: %v", msg.err)
			return d, nil
		}
		d.status = msg.status
		loaded := d.load(msg.path)
		d.path = loaded.path
		d.entries = loaded.entries
		d.err = loaded.err
		d.breadcrumbs = d.pathParts()
		d.clampIndex()
		d.syncEntryList()
		return d, nil
	case DriveActionMsg:
		return d.handleAction(msg.Action)
	case tea.KeyPressMsg:
		return d.handleKey(msg)
	}
	return d, nil
}
