package mail

import (
	"context"
	"github.com/elpdev/telex-cli/internal/api"
)

func (s *Service) Mailboxes(ctx context.Context) (*MailboxBootstrap, error) {
	body, _, err := s.client.Get(ctx, "/api/v1/mailboxes", nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[MailboxBootstrap](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) Labels(ctx context.Context) ([]Label, error) {
	body, _, err := s.client.Get(ctx, "/api/v1/labels", nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[[]Label](body)
	if err != nil {
		return nil, err
	}
	return envelope.Data, nil
}
