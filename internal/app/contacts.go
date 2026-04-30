package app

import (
	"context"
	"time"

	"github.com/elpdev/telex-cli/internal/contacts"
	"github.com/elpdev/telex-cli/internal/contactstore"
	"github.com/elpdev/telex-cli/internal/contactsync"
	"github.com/elpdev/telex-cli/internal/screens"
)

type contactsSyncResult = contactsync.Result

func runContactsSync(ctx context.Context, store contactstore.Store, service *contacts.Service) (*contactsSyncResult, error) {
	return contactsync.Run(ctx, store, service)
}

func (m *Model) syncContacts(ctx context.Context) (screens.ContactsSyncResult, error) {
	service, err := m.contactsService()
	if err != nil {
		return screens.ContactsSyncResult{}, err
	}
	result, err := runContactsSync(ctx, contactstore.New(m.dataPath), service)
	return screens.ContactsSyncResult{Contacts: result.Contacts, Notes: result.Notes}, err
}

func (m *Model) deleteContact(ctx context.Context, id int64) error {
	service, err := m.contactsService()
	if err != nil {
		return err
	}
	if err := service.DeleteContact(ctx, id); err != nil {
		return err
	}
	return contactstore.New(m.dataPath).DeleteContact(id)
}

func (m *Model) updateContact(ctx context.Context, id int64, input contacts.ContactInput) (*contacts.Contact, error) {
	service, err := m.contactsService()
	if err != nil {
		return nil, err
	}
	contact, err := service.UpdateContact(ctx, id, input)
	if err != nil {
		return nil, err
	}
	if err := contactstore.New(m.dataPath).StoreContact(*contact, time.Now()); err != nil {
		return nil, err
	}
	return contact, nil
}

func (m *Model) loadContactNote(ctx context.Context, id int64) (*contacts.ContactNote, error) {
	service, err := m.contactsService()
	if err != nil {
		return nil, err
	}
	note, err := service.ContactNote(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := contactstore.New(m.dataPath).StoreContactNote(*note, time.Now()); err != nil {
		return nil, err
	}
	return note, nil
}

func (m *Model) updateContactNote(ctx context.Context, id int64, input contacts.ContactNoteInput) (*contacts.ContactNote, error) {
	service, err := m.contactsService()
	if err != nil {
		return nil, err
	}
	note, err := service.UpdateContactNote(ctx, id, input)
	if err != nil {
		return nil, err
	}
	if err := contactstore.New(m.dataPath).StoreContactNote(*note, time.Now()); err != nil {
		return nil, err
	}
	return note, nil
}

func (m *Model) loadContactCommunications(ctx context.Context, id int64) ([]contacts.ContactCommunication, error) {
	service, err := m.contactsService()
	if err != nil {
		return nil, err
	}
	communications, _, err := service.ContactCommunications(ctx, id, contacts.ListParams{Page: 1, PerPage: 100})
	if err != nil {
		return nil, err
	}
	if err := contactstore.New(m.dataPath).StoreCommunications(id, communications); err != nil {
		return nil, err
	}
	return communications, nil
}
