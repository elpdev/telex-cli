package mail

import (
	"context"
	"fmt"
	"net/http"

	"github.com/elpdev/telex-cli/internal/api"
)

func (s *Service) ListMessages(ctx context.Context, params MessageListParams) ([]Message, *api.Pagination, error) {
	return api.List[Message](s.client, ctx, "/api/v1/messages", messageQuery(params))
}

func (s *Service) ShowMessage(ctx context.Context, id int64) (*Message, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/messages/%d", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Message](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) MessageBody(ctx context.Context, id int64) (*MessageBody, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/messages/%d/body", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[MessageBody](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) AssignMessageLabels(ctx context.Context, id int64, labelIDs []int64) (*Message, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/messages/%d/labels", id), map[string]any{"label_ids": labelIDs})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Message](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) ConversationTimeline(ctx context.Context, id int64) ([]ConversationTimelineEntry, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/conversations/%d/timeline", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[[]ConversationTimelineEntry](body)
	if err != nil {
		return nil, err
	}
	return envelope.Data, nil
}

func (s *Service) ArchiveMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "archive", nil)
}

func (s *Service) RestoreMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "restore", nil)
}

func (s *Service) TrashMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "trash", nil)
}

func (s *Service) JunkMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "junk", nil)
}

func (s *Service) NotJunkMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "not_junk", nil)
}

func (s *Service) MarkMessageRead(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "mark_read", nil)
}

func (s *Service) MarkMessageUnread(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "mark_unread", nil)
}

func (s *Service) StarMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "star", nil)
}

func (s *Service) UnstarMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "unstar", nil)
}

func (s *Service) BlockSender(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "block_sender", nil)
}

func (s *Service) UnblockSender(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "unblock_sender", nil)
}

func (s *Service) BlockDomain(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "block_domain", nil)
}

func (s *Service) UnblockDomain(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "unblock_domain", nil)
}

func (s *Service) TrustSender(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "trust_sender", nil)
}

func (s *Service) UntrustSender(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "untrust_sender", nil)
}

func (s *Service) Reply(ctx context.Context, id int64) (*OutboundMessage, error) {
	return s.outboundAction(ctx, id, "reply", nil)
}

func (s *Service) ReplyAll(ctx context.Context, id int64) (*OutboundMessage, error) {
	return s.outboundAction(ctx, id, "reply_all", nil)
}

func (s *Service) Forward(ctx context.Context, id int64, targetAddresses []string) (*OutboundMessage, error) {
	return s.outboundAction(ctx, id, "forward", map[string]any{"target_addresses": targetAddresses})
}

func (s *Service) messageAction(ctx context.Context, id int64, method, action string, payload any) (*Message, error) {
	path := fmt.Sprintf("/api/v1/messages/%d/%s", id, action)
	var (
		body []byte
		err  error
	)
	if method == http.MethodPatch {
		body, _, err = s.client.Patch(ctx, path, payload)
	} else {
		body, _, err = s.client.Post(ctx, path, payload)
	}
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Message](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}
