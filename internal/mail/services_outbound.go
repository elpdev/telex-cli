package mail

import (
	"context"
	"fmt"

	"github.com/elpdev/telex-cli/internal/api"
)

func (s *Service) ListOutboundMessages(ctx context.Context, params OutboundMessageListParams) ([]OutboundMessage, *api.Pagination, error) {
	return api.List[OutboundMessage](s.client, ctx, "/api/v1/outbound_messages", outboundMessageQuery(params))
}

func (s *Service) CreateOutboundMessage(ctx context.Context, input *OutboundMessageInput, queue bool) (*OutboundMessage, error) {
	payload := map[string]any{"outbound_message": outboundInputMap(input)}
	if queue {
		payload["queue"] = true
	}
	body, _, err := s.client.Post(ctx, "/api/v1/outbound_messages", payload)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[OutboundMessage](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateOutboundMessage(ctx context.Context, id int64, input *OutboundMessageInput) (*OutboundMessage, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/outbound_messages/%d", id), map[string]any{"outbound_message": outboundInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[OutboundMessage](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) DeleteOutboundMessage(ctx context.Context, id int64) error {
	_, err := s.client.Delete(ctx, fmt.Sprintf("/api/v1/outbound_messages/%d", id))
	return err
}

func (s *Service) SendOutboundMessage(ctx context.Context, id int64) (*OutboundMessage, error) {
	body, _, err := s.client.Post(ctx, fmt.Sprintf("/api/v1/outbound_messages/%d/send_message", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[OutboundMessage](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) AttachOutboundMessageFile(ctx context.Context, outboundMessageID int64, filePath string) ([]Attachment, error) {
	body, _, err := s.client.PostMultipartFile(ctx, fmt.Sprintf("/api/v1/outbound_messages/%d/attachments", outboundMessageID), "file", filePath)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[[]Attachment](body)
	if err != nil {
		return nil, err
	}
	return envelope.Data, nil
}

func (s *Service) ShowOutboundMessage(ctx context.Context, id int64) (*OutboundMessage, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/outbound_messages/%d", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[OutboundMessage](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) outboundAction(ctx context.Context, id int64, action string, payload any) (*OutboundMessage, error) {
	body, _, err := s.client.Post(ctx, fmt.Sprintf("/api/v1/messages/%d/%s", id, action), payload)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[OutboundMessage](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}
