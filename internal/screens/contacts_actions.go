package screens

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

func (c Contacts) Selection() ContactsSelection {
	selected := c.selectedContact()
	if selected == nil {
		return ContactsSelection{Kind: "contact"}
	}
	return ContactsSelection{Kind: "contact", Subject: selected.Meta.DisplayName, HasItem: true}
}

func (c Contacts) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if c.confirm != "" {
		switch msg.String() {
		case "y", "Y":
			id := c.selectedID()
			c.confirm = ""
			if id == 0 || c.delete == nil {
				return c, nil
			}
			c.loading = true
			return c, c.deleteCmd(id)
		default:
			c.confirm = ""
			return c, nil
		}
	}
	if c.editing {
		switch msg.String() {
		case "enter":
			c.editing = false
			c.index = 0
			return c, nil
		case "esc":
			c.editing = false
			c.filter = ""
			c.index = 0
			return c, nil
		case "backspace":
			if len(c.filter) > 0 {
				c.filter = c.filter[:len(c.filter)-1]
				c.clampIndex()
			}
			return c, nil
		default:
			if len(msg.String()) == 1 {
				c.filter += msg.String()
				c.clampIndex()
			}
			return c, nil
		}
	}
	switch {
	case key.Matches(msg, c.keys.Up):
		if c.detail != nil {
			c.detailViewport.ScrollUp(1)
		} else if c.index > 0 {
			c.index--
			c.syncContactList()
		}
	case key.Matches(msg, c.keys.Down):
		if c.detail != nil {
			c.detailViewport.ScrollDown(1)
		} else if c.index < len(c.visibleContacts())-1 {
			c.index++
			c.syncContactList()
		}
	case key.Matches(msg, c.keys.Open):
		if c.detail == nil {
			c.detail = c.selectedContact()
			c.detailViewport = viewport.New()
		}
	case key.Matches(msg, c.keys.Back):
		c.detail = nil
		c.detailViewport = viewport.New()
	case key.Matches(msg, c.keys.Refresh):
		c.loading = true
		return c, c.loadCmd()
	case key.Matches(msg, c.keys.Sync):
		return c.handleAction("sync")
	case key.Matches(msg, c.keys.Search):
		c.editing = true
	case key.Matches(msg, c.keys.Delete):
		if selected := c.selectedContact(); selected != nil {
			c.confirm = fmt.Sprintf("Delete contact %s?", selected.Meta.DisplayName)
		}
	case key.Matches(msg, c.keys.EditNote):
		return c.handleAction("edit-note")
	case key.Matches(msg, c.keys.Note):
		return c.handleAction("refresh-note")
	case key.Matches(msg, c.keys.Communications):
		return c.handleAction("communications")
	}
	return c, nil
}

func (c Contacts) handleAction(action string) (Screen, tea.Cmd) {
	switch action {
	case "sync":
		if c.sync == nil {
			return c, nil
		}
		c.syncing = true
		return c, c.syncCmd()
	case "search":
		c.editing = true
		return c, nil
	case "delete":
		if selected := c.selectedContact(); selected != nil {
			c.confirm = fmt.Sprintf("Delete contact %s?", selected.Meta.DisplayName)
		}
		return c, nil
	case "edit-note":
		id := c.selectedID()
		if id == 0 || c.updateNote == nil {
			return c, nil
		}
		c.loading = true
		return c, c.editNoteCmd(id)
	case "refresh-note":
		id := c.selectedID()
		if id == 0 || c.loadNote == nil {
			return c, nil
		}
		c.loading = true
		return c, c.loadNoteCmd(id)
	case "communications":
		id := c.selectedID()
		if id == 0 || c.loadCommunications == nil {
			return c, nil
		}
		c.loading = true
		return c, c.communicationsCmd(id)
	}
	return c, nil
}
