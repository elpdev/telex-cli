package contactsync

import (
	"context"
	"time"

	"github.com/elpdev/telex-cli/internal/contacts"
	"github.com/elpdev/telex-cli/internal/contactstore"
)

type Result struct {
	Contacts int
	Notes    int
}

func Run(ctx context.Context, store contactstore.Store, service *contacts.Service) (*Result, error) {
	syncedAt := time.Now()
	result := &Result{}
	page := 1
	keep := map[int64]bool{}
	for {
		items, pagination, err := service.ListContacts(ctx, contacts.ListContactsParams{ListParams: contacts.ListParams{Page: page, PerPage: 100}, Sort: "name"})
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			full, err := service.ShowContact(ctx, item.ID, true)
			if err != nil {
				return nil, err
			}
			if err := store.StoreContact(*full, syncedAt); err != nil {
				return nil, err
			}
			keep[full.ID] = true
			result.Contacts++
			if full.Note != nil {
				result.Notes++
			}
		}
		if pagination == nil || page*pagination.PerPage >= pagination.TotalCount {
			break
		}
		page++
	}
	if err := store.StoreSyncMeta(syncedAt); err != nil {
		return nil, err
	}
	if err := store.PruneMissingContacts(keep); err != nil {
		return nil, err
	}
	return result, nil
}

func latestContactUpdatedSince(store contactstore.Store) string {
	items, err := store.ListContacts()
	if err != nil {
		return ""
	}
	var latest time.Time
	for _, item := range items {
		if item.Meta.RemoteUpdatedAt.After(latest) {
			latest = item.Meta.RemoteUpdatedAt
		}
		for _, email := range item.Meta.EmailAddresses {
			if email.UpdatedAt.After(latest) {
				latest = email.UpdatedAt
			}
		}
		if item.Note != nil && item.Note.Meta.UpdatedAt != nil && item.Note.Meta.UpdatedAt.After(latest) {
			latest = *item.Note.Meta.UpdatedAt
		}
	}
	if latest.IsZero() {
		return ""
	}
	return latest.UTC().Format(time.RFC3339Nano)
}
