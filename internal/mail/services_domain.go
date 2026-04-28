package mail

import (
	"context"
	"fmt"
	"github.com/elpdev/telex-cli/internal/api"
)

func (s *Service) ListDomains(ctx context.Context, params DomainListParams) ([]Domain, *api.Pagination, error) {
	return api.List[Domain](s.client, ctx, "/api/v1/domains", domainQuery(params))
}

func (s *Service) ShowDomain(ctx context.Context, id int64) (*Domain, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/domains/%d", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Domain](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) CreateDomain(ctx context.Context, input DomainInput) (*Domain, error) {
	body, _, err := s.client.Post(ctx, "/api/v1/domains", map[string]any{"domain": domainInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Domain](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateDomain(ctx context.Context, id int64, input DomainInput) (*Domain, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/domains/%d", id), map[string]any{"domain": domainInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Domain](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) DeleteDomain(ctx context.Context, id int64) error {
	_, err := s.client.Delete(ctx, fmt.Sprintf("/api/v1/domains/%d", id))
	return err
}

func (s *Service) DomainOutboundStatus(ctx context.Context, id int64) (*DomainOutboundStatus, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/domains/%d/outbound_status", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[DomainOutboundStatus](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) ValidateDomainOutbound(ctx context.Context, id int64, input *DomainInput) (*DomainOutboundValidation, error) {
	var payload any
	if input != nil {
		payload = map[string]any{"domain": domainInputMap(*input)}
	}
	body, _, err := s.client.Post(ctx, fmt.Sprintf("/api/v1/domains/%d/validate_outbound", id), payload)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[DomainOutboundValidation](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}
