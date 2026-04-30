package screens

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/contactstore"
)

func NewContacts(store contactstore.Store, sync ContactsSyncFunc) Contacts {
	return Contacts{store: store, sync: sync, loading: true, contactList: newContactsList(nil, 0, 0, 0), detailViewport: viewport.New(), keys: DefaultContactsKeyMap()}
}

func (c Contacts) WithActions(update UpdateContactFunc, delete DeleteContactFunc, loadNote LoadContactNoteFunc, updateNote UpdateContactNoteFunc, loadCommunications LoadContactCommunicationsFunc) Contacts {
	c.update = update
	c.delete = delete
	c.loadNote = loadNote
	c.updateNote = updateNote
	c.loadCommunications = loadCommunications
	return c
}

func (c Contacts) Init() tea.Cmd { return c.loadCmd() }

func (c Contacts) Title() string { return "Contacts" }
