package screens

import (
	"context"
	"fmt"
	"strconv"

	tea "charm.land/bubbletea/v2"
)

func (c Contacts) loadCmd() tea.Cmd {
	return func() tea.Msg {
		items, err := c.store.ListContacts()
		return contactsLoadedMsg{contacts: items, err: err}
	}
}

func (c Contacts) syncCmd() tea.Cmd {
	return func() tea.Msg {
		result, err := c.sync(context.Background())
		items, loadErr := c.store.ListContacts()
		if err == nil {
			err = loadErr
		}
		return contactsSyncedMsg{result: result, loaded: contactsLoadedMsg{contacts: items, err: loadErr}, err: err}
	}
}

func (c Contacts) deleteCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		err := c.delete(context.Background(), id)
		items, loadErr := c.store.ListContacts()
		if err == nil {
			err = loadErr
		}
		return contactActionFinishedMsg{status: "Deleted contact " + strconv.FormatInt(id, 10), loaded: contactsLoadedMsg{contacts: items, err: loadErr}, err: err}
	}
}

func (c Contacts) loadNoteCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		_, err := c.loadNote(context.Background(), id)
		items, loadErr := c.store.ListContacts()
		if err == nil {
			err = loadErr
		}
		return contactActionFinishedMsg{status: "Refreshed contact note", loaded: contactsLoadedMsg{contacts: items, err: loadErr}, err: err}
	}
}

func (c Contacts) editNoteCmd(id int64) tea.Cmd {
	selected := c.selectedContact()
	if c.detail != nil {
		selected = c.detail
	}
	title := defaultTitle
	body := ""
	if selected != nil {
		title = selected.Meta.DisplayName
		if selected.Note != nil {
			title = selected.Note.Meta.Title
			body = selected.Note.Body
		}
	}
	return func() tea.Msg {
		input, err := editContactNoteTemplate(title, body)
		if err != nil {
			return contactActionFinishedMsg{err: err}
		}
		_, err = c.updateNote(context.Background(), id, input)
		items, loadErr := c.store.ListContacts()
		if err == nil {
			err = loadErr
		}
		return contactActionFinishedMsg{status: "Updated contact note", loaded: contactsLoadedMsg{contacts: items, err: loadErr}, err: err}
	}
}

func (c Contacts) communicationsCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		communications, err := c.loadCommunications(context.Background(), id)
		items, loadErr := c.store.ListContacts()
		if err == nil {
			err = loadErr
		}
		return contactActionFinishedMsg{status: fmt.Sprintf("Loaded %d communication(s)", len(communications)), loaded: contactsLoadedMsg{contacts: items, err: loadErr}, err: err}
	}
}
