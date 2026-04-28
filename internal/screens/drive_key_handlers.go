package screens

import (
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/components/filepicker"
)

func (d Drive) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if d.pickerOpen {
		return d.handlePickerMsg(msg)
	}
	if d.confirm != "" {
		return d.handleConfirmKey(msg)
	}
	if d.prompt != drivePromptNone {
		return d.handlePromptKey(msg)
	}
	if d.filtering {
		return d.handleFilterKey(msg)
	}
	entries := d.visibleEntries()
	if key.Matches(msg, d.keys.Open) {
		if len(entries) == 0 {
			return d, nil
		}
		entry := entries[d.index]
		if entry.Kind != "folder" {
			return d.openSelectedFile(entry)
		}
		path := entry.Path
		d.index = 0
		d.syncEntryList()
		return d, d.loadCmd(path)
	}
	switch {
	case key.Matches(msg, d.keys.Back):
		if filepath.Clean(d.path) == filepath.Clean(d.store.DriveRoot()) {
			return d, nil
		}
		d.index = 0
		d.syncEntryList()
		return d, d.loadCmd(filepath.Dir(d.path))
	case key.Matches(msg, d.keys.Refresh):
		return d, d.loadCmd(d.path)
	case key.Matches(msg, d.keys.Sync):
		if d.sync == nil || d.syncing {
			return d, nil
		}
		d.syncing = true
		d.status = ""
		return d, d.syncCmd()
	case key.Matches(msg, d.keys.Search):
		d.filtering = true
		d.filter = ""
		d.index = 0
		d.syncEntryList()
	case key.Matches(msg, d.keys.Details):
		d.details = !d.details
	case key.Matches(msg, d.keys.Upload):
		cwd, _ := filepath.Abs(".")
		d.picker = filepicker.New("", cwd, filepicker.ModeOpenFile)
		d.pickerOpen = true
		d.status = "Select file to upload"
		return d, d.picker.Init()
	case key.Matches(msg, d.keys.NewDir):
		d.prompt = drivePromptNewFolder
		d.promptInput = ""
		d.status = ""
	case key.Matches(msg, d.keys.Rename):
		if len(entries) == 0 {
			return d, nil
		}
		d.prompt = drivePromptRename
		d.promptInput = entries[d.index].Name
		d.status = ""
	case key.Matches(msg, d.keys.Delete):
		if len(entries) == 0 {
			return d, nil
		}
		d.confirm = "Delete " + entries[d.index].Name + "?"
	default:
		d.ensureEntryList(entries)
		updated, cmd := d.entryList.Update(msg)
		d.entryList = updated
		d.index = d.entryList.GlobalIndex()
		d.clampIndex()
		return d, cmd
	}
	return d, nil
}

func (d Drive) handleFilterKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		d.filtering = false
		d.filter = ""
		d.index = 0
	case "enter":
		d.filtering = false
	case "backspace":
		if len(d.filter) > 0 {
			d.filter = d.filter[:len(d.filter)-1]
		}
		d.index = 0
	default:
		if msg.Text != "" {
			d.filter += msg.Text
			d.index = 0
		}
	}
	d.syncEntryList()
	return d, nil
}

func (d Drive) handlePromptKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		d.prompt = drivePromptNone
		d.promptInput = ""
		d.status = "Cancelled"
		return d, nil
	case "enter":
		value := strings.TrimSpace(d.promptInput)
		prompt := d.prompt
		d.prompt = drivePromptNone
		d.promptInput = ""
		if value == "" {
			d.status = "Name is required"
			return d, nil
		}
		if prompt == drivePromptNewFolder {
			return d, d.createFolderCmd(value)
		}
		return d, d.renameCmd(value)
	case "backspace":
		if len(d.promptInput) > 0 {
			d.promptInput = d.promptInput[:len(d.promptInput)-1]
		}
		return d, nil
	}
	if msg.Text != "" {
		d.promptInput += msg.Text
	}
	return d, nil
}

func (d Drive) handleConfirmKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		d.confirm = ""
		return d, d.deleteCmd()
	case "n", "N", "esc":
		d.confirm = ""
		d.status = "Cancelled"
	}
	return d, nil
}

func (d Drive) handlePickerMsg(msg tea.Msg) (Screen, tea.Cmd) {
	picker, action, cmd := d.picker.Update(msg)
	d.picker = picker
	switch action.Type {
	case filepicker.ActionCancel:
		d.pickerOpen = false
		d.status = "Cancelled"
		return d, nil
	case filepicker.ActionSelect:
		d.pickerOpen = false
		return d, d.uploadCmd(action.Path)
	}
	return d, cmd
}
