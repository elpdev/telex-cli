package mail

import (
	"context"
	"fmt"
	"github.com/elpdev/telex-cli/internal/api"
)

func (s *Service) ListInboxes(ctx context.Context, params InboxListParams) ([]Inbox, *api.Pagination, error) {
	return api.List[Inbox](s.client, ctx, "/api/v1/inboxes", inboxQuery(params))
}

func (s *Service) ShowInbox(ctx context.Context, id int64) (*Inbox, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/inboxes/%d", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Inbox](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) CreateInbox(ctx context.Context, input InboxInput) (*Inbox, error) {
	body, _, err := s.client.Post(ctx, "/api/v1/inboxes", map[string]any{"inbox": inboxInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Inbox](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateInbox(ctx context.Context, id int64, input InboxInput) (*Inbox, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/inboxes/%d", id), map[string]any{"inbox": inboxInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Inbox](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) DeleteInbox(ctx context.Context, id int64) error {
	_, err := s.client.Delete(ctx, fmt.Sprintf("/api/v1/inboxes/%d", id))
	return err
}

func (s *Service) InboxPipeline(ctx context.Context, id int64) (*InboxPipeline, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/inboxes/%d/pipeline", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[InboxPipeline](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) TestInboxForwardingRules(ctx context.Context, id int64, rules []ForwardingRule) (*ForwardingRuleValidation, error) {
	body, _, err := s.client.Post(ctx, fmt.Sprintf("/api/v1/inboxes/%d/test_forwarding_rules", id), map[string]any{"inbox": map[string]any{"forwarding_rules": rules}})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[ForwardingRuleValidation](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}
