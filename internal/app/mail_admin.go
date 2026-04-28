package app

import (
	"context"

	"github.com/elpdev/telex-cli/internal/mail"
)

func (m *Model) loadMailAdmin(ctx context.Context) ([]mail.Domain, []mail.Inbox, error) {
	service, err := m.mailService()
	if err != nil {
		return nil, nil, err
	}
	domains, _, err := service.ListDomains(ctx, mail.DomainListParams{ListParams: mail.ListParams{Page: 1, PerPage: 100}, Sort: "name"})
	if err != nil {
		return nil, nil, err
	}
	inboxes, _, err := service.ListInboxes(ctx, mail.InboxListParams{ListParams: mail.ListParams{Page: 1, PerPage: 250}, Count: "all", Sort: "address"})
	if err != nil {
		return nil, nil, err
	}
	return domains, inboxes, nil
}

func (m *Model) saveDomain(ctx context.Context, id *int64, input mail.DomainInput) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	if id == nil {
		_, err = service.CreateDomain(ctx, input)
		return err
	}
	_, err = service.UpdateDomain(ctx, *id, input)
	return err
}

func (m *Model) deleteDomain(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	return service.DeleteDomain(ctx, id)
}

func (m *Model) validateDomainOutbound(ctx context.Context, id int64) (*mail.DomainOutboundValidation, error) {
	service, err := m.mailService()
	if err != nil {
		return nil, err
	}
	return service.ValidateDomainOutbound(ctx, id, nil)
}

func (m *Model) saveInbox(ctx context.Context, id *int64, input mail.InboxInput) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	if id == nil {
		_, err = service.CreateInbox(ctx, input)
		return err
	}
	_, err = service.UpdateInbox(ctx, *id, input)
	return err
}

func (m *Model) deleteInbox(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	return service.DeleteInbox(ctx, id)
}

func (m *Model) inboxPipeline(ctx context.Context, id int64) (*mail.InboxPipeline, error) {
	service, err := m.mailService()
	if err != nil {
		return nil, err
	}
	return service.InboxPipeline(ctx, id)
}
