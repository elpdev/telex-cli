package app

import (
	"context"
	"time"

	"github.com/elpdev/telex-cli/internal/contacts"
	"github.com/elpdev/telex-cli/internal/contactstore"
)

type contactsSyncResult struct {
	Contacts int
	Notes    int
}

func runContactsSync(ctx context.Context, store contactstore.Store, service *contacts.Service) (*contactsSyncResult, error) {
	syncedAt := time.Now()
	result := &contactsSyncResult{}
	page := 1
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
	return result, nil
}
