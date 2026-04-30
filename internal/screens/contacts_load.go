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
	if selected == nil {
		return func() tea.Msg { return contactActionFinishedMsg{} }
	}
	contact := *selected
	return func() tea.Msg {
		input, err := editContactDocumentTemplate(contact)
		if err != nil {
			return contactActionFinishedMsg{err: err}
		}
		if input.UpdateContact {
			if c.update == nil {
				return contactActionFinishedMsg{err: fmt.Errorf("contact edit is not configured")}
			}
			_, err = c.update(context.Background(), id, input.Contact)
			if err != nil {
				return contactActionFinishedMsg{err: err}
			}
		}
		if input.UpdateNote {
			_, err = c.updateNote(context.Background(), id, input.Note)
		}
		items, loadErr := c.store.ListContacts()
		if err == nil {
			err = loadErr
		}
		return contactActionFinishedMsg{status: "Updated contact", loaded: contactsLoadedMsg{contacts: items, err: loadErr}, err: err}
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
