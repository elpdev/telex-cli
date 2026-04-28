package screens

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/components/filepicker"
)

func (c Calendar) startImportICS() (Screen, tea.Cmd) {
	item, ok := c.selectedCalendar()
	if !ok {
		c.status = "Select a calendar to import ICS"
		return c, nil
	}
	if c.importICS == nil {
		c.status = "ICS import is not configured"
		return c, nil
	}
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		cwd, _ = os.UserHomeDir()
	}
	c.filePicker = filepicker.New("", cwd, filepicker.ModeOpenFile)
	c.filePickerOpen = true
	c.importCalendar = item.RemoteID
	c.status = fmt.Sprintf("Select .ics file for %s", item.Name)
	return c, c.filePicker.Init()
}

func (c Calendar) handleImportFileMsg(msg tea.Msg) (Screen, tea.Cmd) {
	picker, action, cmd := c.filePicker.Update(msg)
	c.filePicker = picker
	switch action.Type {
	case filepicker.ActionCancel:
		c.filePickerOpen = false
		c.importCalendar = 0
		c.status = "Cancelled"
		return c, nil
	case filepicker.ActionSelect:
		return c.importSelectedICS(action.Path)
	}
	if c.filePicker.Err != nil {
		c.status = fmt.Sprintf("File picker: %v", c.filePicker.Err)
	} else {
		c.status = "Select .ics file"
	}
	return c, cmd
}

func (c Calendar) importSelectedICS(path string) (Screen, tea.Cmd) {
	if c.importCalendar <= 0 {
		c.filePickerOpen = false
		c.status = "Select a calendar to import ICS"
		return c, nil
	}
	if strings.ToLower(strings.TrimSpace(path)) == "" || !strings.HasSuffix(strings.ToLower(path), ".ics") {
		c.status = "Select an .ics file"
		return c, nil
	}
	calendarID := c.importCalendar
	c.filePickerOpen = false
	c.importCalendar = 0
	c.loading = true
	c.status = "Importing ICS..."
	return c, c.importICSCmd(calendarID, path)
}
