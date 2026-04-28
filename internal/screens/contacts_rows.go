package screens

import (
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	"github.com/elpdev/telex-cli/internal/contactstore"
)

func (c Contacts) visibleContacts() []contactstore.CachedContact {
	if c.filter == "" {
		return c.contacts
	}
	query := strings.ToLower(c.filter)
	out := []contactstore.CachedContact{}
	for _, contact := range c.contacts {
		text := strings.ToLower(strings.Join([]string{contact.Meta.DisplayName, contact.Meta.PrimaryEmailAddress, contact.Meta.CompanyName, contact.Meta.Title, contact.Meta.Phone}, " "))
		if strings.Contains(text, query) {
			out = append(out, contact)
		}
	}
	return out
}

func (c *Contacts) clampIndex() {
	visible := c.visibleContacts()
	if c.index >= len(visible) {
		c.index = len(visible) - 1
	}
	if c.index < 0 {
		c.index = 0
	}
	c.syncContactList()
}

func (c *Contacts) refreshDetail() {
	if c.detail == nil {
		return
	}
	id := c.detail.Meta.RemoteID
	c.detail = nil
	for i := range c.contacts {
		if c.contacts[i].Meta.RemoteID == id {
			contact := c.contacts[i]
			c.detail = &contact
			return
		}
	}
}

func (c Contacts) selectedContact() *contactstore.CachedContact {
	visible := c.visibleContacts()
	if len(visible) == 0 || c.index < 0 || c.index >= len(visible) {
		return nil
	}
	selected := visible[c.index]
	return &selected
}

func (c Contacts) selectedID() int64 {
	if c.detail != nil {
		return c.detail.Meta.RemoteID
	}
	if selected := c.selectedContact(); selected != nil {
		return selected.Meta.RemoteID
	}
	return 0
}

func (c Contacts) renderContactsList(contacts []contactstore.CachedContact, width, height int) string {
	c.ensureContactList(contacts)
	c.contactList.SetSize(width, height)
	return c.contactList.View()
}

func (c *Contacts) syncContactList() {
	c.contactList = newContactsList(c.visibleContacts(), c.index, c.contactList.Width(), c.contactList.Height())
}

func (c Contacts) ensureContactList(contacts []contactstore.CachedContact) {
	if len(c.contactList.Items()) != len(contacts) {
		c.contactList = newContactsList(contacts, c.index, c.contactList.Width(), c.contactList.Height())
		return
	}
	c.contactList.Select(c.index)
}

func newContactsList(contacts []contactstore.CachedContact, selected, width, height int) list.Model {
	items := make([]list.Item, 0, len(contacts))
	for _, contact := range contacts {
		items = append(items, contactListItem{contact: contact})
	}
	return newSimpleList(items, contactListDelegate{}, selected, width, height)
}

type contactListDelegate struct{ simpleDelegate }

func (d contactListDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	contactItem, ok := item.(contactListItem)
	if !ok {
		return
	}
	_, _ = io.WriteString(w, formatContactRow(contactItem.contact, index == m.Index(), m.Width()))
}

func formatContactRow(contact contactstore.CachedContact, selected bool, width int) string {
	cursor := listCursor(selected)
	name := contact.Meta.DisplayName
	if name == "" {
		name = "Unnamed contact"
	}
	parts := []string{name}
	if contact.Meta.PrimaryEmailAddress != "" {
		parts = append(parts, "<"+contact.Meta.PrimaryEmailAddress+">")
	}
	if contact.Meta.CompanyName != "" {
		parts = append(parts, contact.Meta.CompanyName)
	}
	return cursor + truncate(strings.Join(parts, " · "), max(0, width-2))
}
