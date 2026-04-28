package screens

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	"context"
	"github.com/elpdev/telex-cli/internal/contacts"
	"github.com/elpdev/telex-cli/internal/contactstore"
)

type ContactsSyncFunc func(context.Context) (ContactsSyncResult, error)
type DeleteContactFunc func(context.Context, int64) error
type LoadContactNoteFunc func(context.Context, int64) (*contacts.ContactNote, error)
type UpdateContactNoteFunc func(context.Context, int64, contacts.ContactNoteInput) (*contacts.ContactNote, error)
type LoadContactCommunicationsFunc func(context.Context, int64) ([]contacts.ContactCommunication, error)

type ContactsSyncResult struct {
	Contacts int
	Notes    int
}

type Contacts struct {
	store              contactstore.Store
	sync               ContactsSyncFunc
	delete             DeleteContactFunc
	loadNote           LoadContactNoteFunc
	updateNote         UpdateContactNoteFunc
	loadCommunications LoadContactCommunicationsFunc
	contacts           []contactstore.CachedContact
	contactList        list.Model
	detailViewport     viewport.Model
	index              int
	detail             *contactstore.CachedContact
	filter             string
	editing            bool
	confirm            string
	loading            bool
	syncing            bool
	err                error
	status             string
	keys               ContactsKeyMap
}

type ContactsKeyMap struct {
	Up             key.Binding
	Down           key.Binding
	Open           key.Binding
	Back           key.Binding
	Refresh        key.Binding
	Sync           key.Binding
	Search         key.Binding
	Delete         key.Binding
	EditNote       key.Binding
	Note           key.Binding
	Communications key.Binding
}

type contactsLoadedMsg struct {
	contacts []contactstore.CachedContact
	err      error
}

type contactsSyncedMsg struct {
	result ContactsSyncResult
	loaded contactsLoadedMsg
	err    error
}

type contactActionFinishedMsg struct {
	status string
	loaded contactsLoadedMsg
	err    error
}

type ContactsActionMsg struct{ Action string }

type ContactsSelection struct {
	Kind    string
	Subject string
	HasItem bool
}

type contactListItem struct {
	contact contactstore.CachedContact
}

func (i contactListItem) FilterValue() string { return i.contact.Meta.DisplayName }
