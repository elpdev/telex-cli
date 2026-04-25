package mail

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/elpdev/telex-cli/internal/api"
)

type Client interface {
	Get(context.Context, string, url.Values) ([]byte, int, error)
	Post(context.Context, string, any) ([]byte, int, error)
	PostMultipartFile(context.Context, string, string, string) ([]byte, int, error)
	Patch(context.Context, string, any) ([]byte, int, error)
	Delete(context.Context, string) (int, error)
}

type Service struct {
	client Client
}

func NewService(client Client) *Service { return &Service{client: client} }

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

func (s *Service) ListMessages(ctx context.Context, params MessageListParams) ([]Message, *api.Pagination, error) {
	body, _, err := s.client.Get(ctx, "/api/v1/messages", messageQuery(params))
	if err != nil {
		return nil, nil, err
	}
	envelope, err := api.DecodeEnvelope[[]Message](body)
	if err != nil {
		return nil, nil, err
	}
	return envelope.Data, api.DecodePagination(envelope.Meta), nil
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

func (s *Service) ArchiveMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "archive", nil)
}

func (s *Service) RestoreMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "restore", nil)
}

func (s *Service) TrashMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "trash", nil)
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

func (s *Service) Reply(ctx context.Context, id int64) (*OutboundMessage, error) {
	return s.outboundAction(ctx, id, "reply", nil)
}

func (s *Service) ReplyAll(ctx context.Context, id int64) (*OutboundMessage, error) {
	return s.outboundAction(ctx, id, "reply_all", nil)
}

func (s *Service) Forward(ctx context.Context, id int64, targetAddresses []string) (*OutboundMessage, error) {
	return s.outboundAction(ctx, id, "forward", map[string]any{"target_addresses": targetAddresses})
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

func messageQuery(params MessageListParams) url.Values {
	query := url.Values{}
	setInt(query, "page", params.Page)
	setInt(query, "per_page", params.PerPage)
	setInt64(query, "inbox_id", params.InboxID)
	setInt64(query, "conversation_id", params.ConversationID)
	setString(query, "mailbox", params.Mailbox)
	setInt64(query, "label_id", params.LabelID)
	setString(query, "q", params.Query)
	setString(query, "sender", params.Sender)
	setString(query, "recipient", params.Recipient)
	setString(query, "status", params.Status)
	setString(query, "subaddress", params.Subaddress)
	setString(query, "received_from", params.ReceivedFrom)
	setString(query, "received_to", params.ReceivedTo)
	setString(query, "sort", params.Sort)
	return query
}

func setString(query url.Values, key, value string) {
	if value != "" {
		query.Set(key, value)
	}
}

func setInt(query url.Values, key string, value int) {
	if value > 0 {
		query.Set(key, fmt.Sprintf("%d", value))
	}
}

func setInt64(query url.Values, key string, value int64) {
	if value > 0 {
		query.Set(key, fmt.Sprintf("%d", value))
	}
}

func outboundInputMap(input *OutboundMessageInput) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	payload := map[string]any{}
	if input.DomainID != nil {
		payload["domain_id"] = *input.DomainID
	}
	if input.InboxID != nil {
		payload["inbox_id"] = *input.InboxID
	}
	if input.SourceMessageID != nil {
		payload["source_message_id"] = *input.SourceMessageID
	}
	if input.ConversationID != nil {
		payload["conversation_id"] = *input.ConversationID
	}
	if len(input.ToAddresses) > 0 {
		payload["to_addresses"] = input.ToAddresses
	}
	if len(input.CCAddresses) > 0 {
		payload["cc_addresses"] = input.CCAddresses
	}
	if len(input.BCCAddresses) > 0 {
		payload["bcc_addresses"] = input.BCCAddresses
	}
	if input.Subject != "" {
		payload["subject"] = input.Subject
	}
	if input.Body != "" {
		payload["body"] = input.Body
	}
	if input.Status != "" {
		payload["status"] = input.Status
	}
	if input.InReplyToMessageID != "" {
		payload["in_reply_to_message_id"] = input.InReplyToMessageID
	}
	if len(input.ReferenceMessageIDs) > 0 {
		payload["reference_message_ids"] = input.ReferenceMessageIDs
	}
	if len(input.Metadata) > 0 {
		payload["metadata"] = input.Metadata
	}
	return payload
}
