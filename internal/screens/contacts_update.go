package screens

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
)

func (c Contacts) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case contactsLoadedMsg:
		c.loading = false
		c.err = msg.err
		if msg.err == nil {
			c.contacts = msg.contacts
			c.clampIndex()
			c.refreshDetail()
		}
		return c, nil
	case contactsSyncedMsg:
		c.syncing = false
		c.err = msg.err
		if msg.err == nil {
			c.status = fmt.Sprintf("Synced %d contact(s), %d note(s)", msg.result.Contacts, msg.result.Notes)
			c.contacts = msg.loaded.contacts
			c.clampIndex()
			c.refreshDetail()
		} else {
			c.status = ""
		}
		return c, nil
	case contactActionFinishedMsg:
		c.loading = false
		c.err = msg.err
		if msg.err == nil {
			c.status = msg.status
			c.contacts = msg.loaded.contacts
			c.clampIndex()
			c.refreshDetail()
		} else {
			c.status = fmt.Sprintf("Contacts action failed: %v", msg.err)
		}
		return c, nil
	case ContactsActionMsg:
		return c.handleAction(msg.Action)
	case tea.KeyPressMsg:
		return c.handleKey(msg)
	}
	return c, nil
}
