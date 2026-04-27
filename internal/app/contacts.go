package app

import (
	"context"

	"github.com/elpdev/telex-cli/internal/contacts"
	"github.com/elpdev/telex-cli/internal/contactstore"
	"github.com/elpdev/telex-cli/internal/contactsync"
)

type contactsSyncResult = contactsync.Result

func runContactsSync(ctx context.Context, store contactstore.Store, service *contacts.Service) (*contactsSyncResult, error) {
	return contactsync.Run(ctx, store, service)
}
